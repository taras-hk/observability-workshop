package observability

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// V1 Metrics - BAD PRACTICES (What NOT to do)
// Problems:
// - Very few metrics
// - No labels/dimensions
// - No business context
// - Hard to understand system health
// - No SLI/SLO alignment

type MetricsV1 struct {
	// Just basic counters with no dimensions
	TotalRequests prometheus.Counter
	TotalErrors   prometheus.Counter
}

func NewMetricsV1(serviceName string) *MetricsV1 {
	m := &MetricsV1{}

	// Bad: No labels, no context, hard-coded names
	m.TotalRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "requests_v1", // Bad: too generic but at least versioned
		Help: "requests",    // Bad: unhelpful description
	})

	m.TotalErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "errors_v1", // Bad: too generic but at least versioned
		Help: "errors",    // Bad: unhelpful description
	})

	// Register with default registry (bad practice in production)
	prometheus.MustRegister(m.TotalRequests, m.TotalErrors)

	return m
}

// V1 Handler - Minimal metrics collection
func InstrumentHandlerV1(next http.HandlerFunc, metrics *MetricsV1) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Bad: Only count total requests, no context
		metrics.TotalRequests.Inc()

		// Still need tracing for comparison
		ctx := r.Context()
		tracer := otel.Tracer("http-middleware-v1")
		ctx, span := tracer.Start(ctx, "V1 Request",
			trace.WithAttributes(
				attribute.String("version", "v1"),
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.Path),
			))
		defer span.End()

		wrapped := &ResponseWriter{ResponseWriter: w, Status: http.StatusOK}
		next.ServeHTTP(wrapped, r.WithContext(ctx))

		// Bad: Only count errors, no classification
		if wrapped.Status >= 400 {
			metrics.TotalErrors.Inc()
			log.Printf("Error occurred: %d", wrapped.Status) // Bad: unstructured logging
		}

		// Bad: No timing metrics
		// Bad: No business metrics
		// Bad: No detailed error context
	}
}
