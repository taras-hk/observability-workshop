package observability

import (
	"context"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// TracingV1 demonstrates TERRIBLE tracing practices (The "What Not To Do" Example)
// - No context propagation
// - No meaningful span names
// - No attributes or metadata
// - Manual timing instead of spans
// - No error handling
type TracingV1 struct {
	tracer trace.Tracer
}

func NewTracingV1(serviceName string) *TracingV1 {
	// V1: Minimal setup with poor configuration
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://jaeger:14268/api/traces")))
	if err != nil {
		log.Printf("V1: Failed to create exporter: %v", err)
		// V1: Poor error handling - just continue without tracing
		return &TracingV1{tracer: otel.Tracer("noop")}
	}

	// V1: Poor sampling - either all or nothing
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		)),
	)
	otel.SetTracerProvider(tp)

	return &TracingV1{
		tracer: otel.Tracer(serviceName),
	}
}

// V1: Poor middleware - no context propagation, minimal span information
func (t *TracingV1) InstrumentHandler(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// V1: Create span with poor naming (generic, not descriptive)
		_, span := t.tracer.Start(context.Background(), "request")
		defer span.End()

		// V1: No context propagation - breaks distributed tracing
		handler(w, r)
	}
}

// V1: Manual timing instead of proper span hierarchy
func (t *TracingV1) TraceOperation(name string, operation func()) {
	start := time.Now()

	// V1: No actual tracing, just logging
	log.Printf("V1: Starting operation: %s", name)

	operation()

	duration := time.Since(start)
	log.Printf("V1: Completed operation: %s in %v", name, duration)
}

// V1: No context awareness
func (t *TracingV1) StartSpan(name string) (context.Context, trace.Span) {
	// V1: Always use background context - breaks trace continuity
	return t.tracer.Start(context.Background(), name)
}

// V1: No error tracking in spans
func (t *TracingV1) RecordError(span trace.Span, err error) {
	// V1: Do nothing - errors are not tracked in traces
	log.Printf("V1: Error occurred: %v", err)
}

// V1: No custom attributes
func (t *TracingV1) AddAttributes(span trace.Span, attributes map[string]interface{}) {
	// V1: Do nothing - no custom attributes support
}
