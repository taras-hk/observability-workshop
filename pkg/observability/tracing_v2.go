package observability

import (
	"context"
	"log"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// TracingV2 demonstrates BETTER tracing practices (Improved but inconsistent)
// - Basic context propagation
// - Some meaningful span names
// - Limited attributes
// - Inconsistent error handling
// - Some custom attributes but not standardized
type TracingV2 struct {
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
}

func NewTracingV2(serviceName string) *TracingV2 {
	// V2: Better setup with some configuration
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://jaeger:14268/api/traces")))
	if err != nil {
		log.Printf("V2: Failed to create exporter: %v", err)
		return &TracingV2{
			tracer:     otel.Tracer("noop"),
			propagator: propagation.NewCompositeTextMapPropagator(),
		}
	}

	// V2: Better sampling configuration
	sampler := tracesdk.ParentBased(
		tracesdk.TraceIDRatioBased(0.5), // V2: Higher sampling rate but not configurable
	)

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(sampler),
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("v2"), // V2: Add version but hardcoded
		)),
	)
	otel.SetTracerProvider(tp)

	// V2: Set up propagation (but limited)
	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
	)
	otel.SetTextMapPropagator(propagator)

	return &TracingV2{
		tracer:     otel.Tracer(serviceName),
		propagator: propagator,
	}
}

// V2: Better middleware - basic context propagation, some span attributes
func (t *TracingV2) InstrumentHandler(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// V2: Extract context from incoming request
		ctx := t.propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		// V2: Better span naming (includes HTTP method)
		spanName := r.Method + " " + r.URL.Path
		ctx, span := t.tracer.Start(ctx, spanName)
		defer span.End()

		// V2: Add some basic attributes
		span.SetAttributes(
			semconv.HTTPMethod(r.Method),
			semconv.HTTPTarget(r.URL.Path),
		)

		// V2: Create a response wrapper to capture status code
		wrapper := &responseWrapper{ResponseWriter: w, statusCode: 200}

		// V2: Pass context to handler
		handler(wrapper, r.WithContext(ctx))

		// V2: Record response status
		span.SetAttributes(semconv.HTTPStatusCode(wrapper.statusCode))

		// V2: Basic error detection (only 5xx errors)
		if wrapper.statusCode >= 500 {
			span.SetStatus(codes.Error, "Server error")
		}
	}
}

// V2: Better operation tracing with context
func (t *TracingV2) TraceOperation(ctx context.Context, name string, operation func(context.Context) error) error {
	// V2: Use provided context for span hierarchy
	ctx, span := t.tracer.Start(ctx, name)
	defer span.End()

	// V2: Add operation type attribute
	span.SetAttributes(attribute.String("operation.type", "business"))

	err := operation(ctx)

	// V2: Record errors (but inconsistently)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}

	return err
}

// V2: Context-aware span creation
func (t *TracingV2) StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name)
}

// V2: Basic error tracking
func (t *TracingV2) RecordError(span trace.Span, err error) {
	span.SetStatus(codes.Error, err.Error())
	span.RecordError(err)
}

// V2: Some custom attributes support (but inconsistent)
func (t *TracingV2) AddAttributes(span trace.Span, attributes map[string]interface{}) {
	for key, value := range attributes {
		// V2: Basic type handling (limited types)
		switch v := value.(type) {
		case string:
			span.SetAttributes(attribute.String(key, v))
		case int:
			span.SetAttributes(attribute.Int(key, v))
		case bool:
			span.SetAttributes(attribute.Bool(key, v))
		default:
			// V2: Convert to string for unknown types
			span.SetAttributes(attribute.String(key, ""+interface{}(v).(string)))
		}
	}
}

// V2: Database operation tracing (basic)
func (t *TracingV2) TraceDBOperation(ctx context.Context, operation, table string, query func(context.Context) error) error {
	ctx, span := t.tracer.Start(ctx, operation)
	defer span.End()

	// V2: Some database attributes
	span.SetAttributes(
		attribute.String("db.operation", operation),
		attribute.String("db.table", table),
	)

	err := query(ctx)
	if err != nil {
		t.RecordError(span, err)
	}

	return err
}

// V2: HTTP client tracing (basic)
func (t *TracingV2) TraceHTTPClient(ctx context.Context, method, url string, request func(context.Context) (*http.Response, error)) (*http.Response, error) {
	ctx, span := t.tracer.Start(ctx, method+" "+url)
	defer span.End()

	// V2: Basic HTTP client attributes
	span.SetAttributes(
		semconv.HTTPMethod(method),
		semconv.HTTPTarget(url),
	)

	// V2: Inject context into outgoing request (basic)
	resp, err := request(ctx)

	if err != nil {
		t.RecordError(span, err)
		return nil, err
	}

	// V2: Record response status
	if resp != nil {
		span.SetAttributes(semconv.HTTPStatusCode(resp.StatusCode))
	}

	return resp, err
}

// Helper type for V2
type responseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWrapper) Write(b []byte) (int, error) {
	return rw.ResponseWriter.Write(b)
}
