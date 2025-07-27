package observability

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// TracingV3 demonstrates EXCELLENT tracing practices (The "Right Way")
// - Full context propagation with baggage
// - Rich, semantic span names following OpenTelemetry conventions
// - Comprehensive attributes using semantic conventions
// - Proper error handling and status codes
// - Custom span events and annotations
// - Performance monitoring with span metrics
// - Business context and user journey tracking
// - Configurable sampling strategies
// - Resource attributes for deployment context
type TracingV3 struct {
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
	config     TracingV3Config
}

type TracingV3Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	DeploymentMode string
	SampleRatio    float64
	JaegerEndpoint string
	EnableMetrics  bool
	EnableBaggage  bool
}

func NewTracingV3(config TracingV3Config) *TracingV3 {
	// V3: Comprehensive configuration with defaults
	if config.ServiceName == "" {
		config.ServiceName = "unknown-service"
	}
	if config.ServiceVersion == "" {
		config.ServiceVersion = "unknown"
	}
	if config.Environment == "" {
		config.Environment = os.Getenv("ENVIRONMENT")
		if config.Environment == "" {
			config.Environment = "development"
		}
	}
	if config.DeploymentMode == "" {
		config.DeploymentMode = "container"
	}
	if config.SampleRatio == 0 {
		config.SampleRatio = 0.1 // V3: Conservative default
	}
	if config.JaegerEndpoint == "" {
		config.JaegerEndpoint = "http://jaeger:14268/api/traces"
	}

	// V3: Enhanced exporter configuration
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.JaegerEndpoint)))
	if err != nil {
		log.Printf("V3: Failed to create tracer exporter: %v", err)
		return &TracingV3{
			tracer:     otel.Tracer("noop"),
			propagator: propagation.NewCompositeTextMapPropagator(),
			config:     config,
		}
	}

	// V3: Sophisticated sampling strategy
	var sampler tracesdk.Sampler
	if config.Environment == "production" {
		// V3: Lower sampling in production with parent-based decisions
		sampler = tracesdk.ParentBased(
			tracesdk.TraceIDRatioBased(config.SampleRatio),
		)
	} else {
		// V3: Higher sampling in non-production
		sampler = tracesdk.ParentBased(
			tracesdk.TraceIDRatioBased(config.SampleRatio * 2),
		)
	}

	// V3: Rich resource attributes for deployment context
	hostname, _ := os.Hostname()
	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(config.ServiceName),
		semconv.ServiceVersion(config.ServiceVersion),
		semconv.DeploymentEnvironment(config.Environment),
		attribute.String("deployment.mode", config.DeploymentMode),
		attribute.String("host.name", hostname),
		attribute.String("telemetry.sdk.language", "go"),
		attribute.String("telemetry.sdk.version", runtime.Version()),
	)

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(sampler),
		tracesdk.WithBatcher(exporter,
			// V3: Optimized batching configuration
			tracesdk.WithMaxExportBatchSize(512),
			tracesdk.WithBatchTimeout(5*time.Second),
			tracesdk.WithMaxQueueSize(2048),
		),
		tracesdk.WithResource(resource),
	)
	otel.SetTracerProvider(tp)

	// V3: Full propagation setup with baggage for business context
	propagators := []propagation.TextMapPropagator{
		propagation.TraceContext{},
		propagation.Baggage{},
	}
	propagator := propagation.NewCompositeTextMapPropagator(propagators...)
	otel.SetTextMapPropagator(propagator)

	return &TracingV3{
		tracer:     otel.Tracer(config.ServiceName),
		propagator: propagator,
		config:     config,
	}
}

// V3: Comprehensive HTTP middleware with full observability
func (t *TracingV3) InstrumentHandler(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// V3: Extract full context including baggage
		ctx := t.propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		// V3: Semantic span naming following OpenTelemetry conventions
		spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		ctx, span := t.tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		// V3: Comprehensive HTTP semantic attributes
		span.SetAttributes(
			semconv.HTTPMethod(r.Method),
			semconv.HTTPTarget(r.URL.Path),
			semconv.HTTPRoute(r.URL.Path), // In real app, this would be the route pattern
			semconv.HTTPScheme(r.URL.Scheme),
			attribute.String("http.host", r.Host),
			semconv.HTTPUserAgent(r.UserAgent()),
			attribute.Int64("http.request.content_length", r.ContentLength),
			attribute.String("net.peer.ip", r.RemoteAddr),
		)

		// V3: Add custom business context from headers
		if userID := r.Header.Get("X-User-ID"); userID != "" {
			span.SetAttributes(attribute.String("user.id", userID))
		}
		if tenantID := r.Header.Get("X-Tenant-ID"); tenantID != "" {
			span.SetAttributes(attribute.String("tenant.id", tenantID))
		}
		if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
			span.SetAttributes(attribute.String("request.id", requestID))
		}

		// V3: Add span event for request start
		span.AddEvent("request.started", trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.target", r.URL.Path),
		))

		// V3: Enhanced response wrapper with timing
		wrapper := &responseWrapperV3{
			ResponseWriter: w,
			statusCode:     200,
			bytesWritten:   0,
		}

		// V3: Execute handler with enriched context
		handler(wrapper, r.WithContext(ctx))

		duration := time.Since(start)

		// V3: Record comprehensive response attributes
		span.SetAttributes(
			semconv.HTTPStatusCode(wrapper.statusCode),
			attribute.Int64("http.response.content_length", int64(wrapper.bytesWritten)),
			attribute.Int64("http.response.duration_ms", duration.Milliseconds()),
		)

		// V3: Sophisticated error detection and categorization
		if wrapper.statusCode >= 400 {
			if wrapper.statusCode >= 500 {
				span.SetStatus(codes.Error, "Server error")
				span.SetAttributes(attribute.String("error.type", "server_error"))
			} else {
				span.SetStatus(codes.Error, "Client error")
				span.SetAttributes(attribute.String("error.type", "client_error"))
			}
		} else {
			span.SetStatus(codes.Ok, "Success")
		}

		// V3: Add span event for request completion
		span.AddEvent("request.completed", trace.WithAttributes(
			attribute.Int("http.status_code", wrapper.statusCode),
			attribute.Int64("duration_ms", duration.Milliseconds()),
		))

		// V3: Performance annotations
		if duration > 1*time.Second {
			span.AddEvent("slow_request", trace.WithAttributes(
				attribute.Int64("duration_ms", duration.Milliseconds()),
				attribute.String("performance.issue", "slow_response"),
			))
		}
	}
}

// V3: Advanced operation tracing with business context
func (t *TracingV3) TraceOperation(ctx context.Context, operationName string, operationType string, attributes map[string]interface{}, operation func(context.Context) error) error {
	ctx, span := t.tracer.Start(ctx, operationName,
		trace.WithSpanKind(trace.SpanKindInternal),
	)
	defer span.End()

	// V3: Standard operation attributes
	span.SetAttributes(
		attribute.String("operation.type", operationType),
		attribute.String("operation.name", operationName),
	)

	// V3: Add custom attributes
	t.AddAttributes(span, attributes)

	// V3: Add operation start event
	span.AddEvent("operation.started")

	err := operation(ctx)

	// V3: Comprehensive error handling
	if err != nil {
		t.RecordError(span, err, map[string]interface{}{
			"operation.name": operationName,
			"operation.type": operationType,
		})
	} else {
		span.SetStatus(codes.Ok, "Operation completed successfully")
	}

	// V3: Add operation completion event
	span.AddEvent("operation.completed", trace.WithAttributes(
		attribute.Bool("operation.success", err == nil),
	))

	return err
}

// V3: Database operation tracing with full semantic conventions
func (t *TracingV3) TraceDBOperation(ctx context.Context, operation, table, database string, query func(context.Context) error) error {
	spanName := fmt.Sprintf("db %s %s", operation, table)
	ctx, span := t.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	// V3: Full database semantic attributes
	span.SetAttributes(
		attribute.String("db.system", "postgresql"), // Assuming PostgreSQL
		semconv.DBName(database),
		semconv.DBOperation(operation),
		attribute.String("db.table", table),
		attribute.String("db.statement.type", operation),
	)

	span.AddEvent("db.query.started")

	err := query(ctx)

	if err != nil {
		t.RecordError(span, err, map[string]interface{}{
			"db.operation": operation,
			"db.table":     table,
		})
	} else {
		span.SetStatus(codes.Ok, "Database operation successful")
	}

	span.AddEvent("db.query.completed")
	return err
}

// V3: HTTP client tracing with full semantic conventions
func (t *TracingV3) TraceHTTPClient(ctx context.Context, method, url string, requestFunc func(context.Context, *http.Request) (*http.Response, error)) (*http.Response, error) {
	spanName := fmt.Sprintf("HTTP %s", method)
	ctx, span := t.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	// V3: Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		t.RecordError(span, err, map[string]interface{}{
			"http.method": method,
			"http.url":    url,
		})
		return nil, err
	}

	// V3: Full HTTP client semantic attributes
	span.SetAttributes(
		semconv.HTTPMethod(method),
		semconv.HTTPTarget(req.URL.Path),
		attribute.String("http.host", req.URL.Host),
		semconv.HTTPScheme(req.URL.Scheme),
	)

	// V3: Inject trace context into outgoing request
	t.propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))

	span.AddEvent("http.request.started")

	resp, err := requestFunc(ctx, req)

	if err != nil {
		t.RecordError(span, err, map[string]interface{}{
			"http.method": method,
			"http.url":    url,
		})
		return nil, err
	}

	// V3: Record response attributes
	span.SetAttributes(
		semconv.HTTPStatusCode(resp.StatusCode),
		attribute.Int64("http.response.content_length", resp.ContentLength),
	)

	// V3: Set span status based on HTTP status
	if resp.StatusCode >= 400 {
		if resp.StatusCode >= 500 {
			span.SetStatus(codes.Error, "Server error")
		} else {
			span.SetStatus(codes.Error, "Client error")
		}
	} else {
		span.SetStatus(codes.Ok, "HTTP request successful")
	}

	span.AddEvent("http.request.completed", trace.WithAttributes(
		attribute.Int("http.status_code", resp.StatusCode),
	))

	return resp, nil
}

// V3: Context-aware span creation with options
func (t *TracingV3) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, opts...)
}

// V3: Comprehensive error recording with context
func (t *TracingV3) RecordError(span trace.Span, err error, context map[string]interface{}) {
	span.SetStatus(codes.Error, err.Error())
	span.RecordError(err)

	// V3: Add error context attributes
	errorAttrs := []attribute.KeyValue{
		attribute.String("error.message", err.Error()),
		attribute.String("error.type", fmt.Sprintf("%T", err)),
	}

	// V3: Add custom context
	for key, value := range context {
		switch v := value.(type) {
		case string:
			errorAttrs = append(errorAttrs, attribute.String("error.context."+key, v))
		case int:
			errorAttrs = append(errorAttrs, attribute.Int("error.context."+key, v))
		case bool:
			errorAttrs = append(errorAttrs, attribute.Bool("error.context."+key, v))
		}
	}

	span.SetAttributes(errorAttrs...)

	// V3: Add error event
	span.AddEvent("error.occurred", trace.WithAttributes(errorAttrs...))
}

// V3: Rich attribute handling with type safety
func (t *TracingV3) AddAttributes(span trace.Span, attributes map[string]interface{}) {
	for key, value := range attributes {
		switch v := value.(type) {
		case string:
			span.SetAttributes(attribute.String(key, v))
		case int:
			span.SetAttributes(attribute.Int(key, v))
		case int64:
			span.SetAttributes(attribute.Int64(key, v))
		case float64:
			span.SetAttributes(attribute.Float64(key, v))
		case bool:
			span.SetAttributes(attribute.Bool(key, v))
		case []string:
			span.SetAttributes(attribute.StringSlice(key, v))
		case []int:
			span.SetAttributes(attribute.IntSlice(key, v))
		case time.Time:
			span.SetAttributes(attribute.String(key, v.Format(time.RFC3339)))
		case time.Duration:
			span.SetAttributes(attribute.Int64(key+"_ms", v.Milliseconds()))
		default:
			// V3: Fallback to string representation with type info
			span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
		}
	}
}

// V3: Business context tracking
func (t *TracingV3) AddBusinessContext(ctx context.Context, userID, tenantID, sessionID string) context.Context {
	if t.config.EnableBaggage {
		// V3: Add business context to baggage for cross-service propagation
		b := baggage.FromContext(ctx)
		var err error
		if userID != "" {
			member, _ := baggage.NewMember("user.id", userID)
			b, err = b.SetMember(member)
			if err != nil {
				log.Printf("V3: Failed to set user.id in baggage: %v", err)
			}
		}
		if tenantID != "" {
			member, _ := baggage.NewMember("tenant.id", tenantID)
			b, err = b.SetMember(member)
			if err != nil {
				log.Printf("V3: Failed to set tenant.id in baggage: %v", err)
			}
		}
		if sessionID != "" {
			member, _ := baggage.NewMember("session.id", sessionID)
			b, err = b.SetMember(member)
			if err != nil {
				log.Printf("V3: Failed to set session.id in baggage: %v", err)
			}
		}
		ctx = baggage.ContextWithBaggage(ctx, b)
	}
	return ctx
}

// V3: Performance monitoring
func (t *TracingV3) AddPerformanceEvent(span trace.Span, eventName string, duration time.Duration, threshold time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.Int64("duration_ms", duration.Milliseconds()),
	}

	if duration > threshold {
		attrs = append(attrs, attribute.Bool("performance.slow", true))
		attrs = append(attrs, attribute.Int64("threshold_ms", threshold.Milliseconds()))
	}

	span.AddEvent(eventName, trace.WithAttributes(attrs...))
}

// Helper type for V3
type responseWrapperV3 struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (rw *responseWrapperV3) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWrapperV3) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}
