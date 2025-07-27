package observability

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// V2 Metrics - BETTER PRACTICES (Getting better but still issues)
// Improvements over V1:
// - Some labels/dimensions
// - Basic timing metrics
// - Better naming
// Still problems:
// - Inconsistent labeling
// - Missing business metrics
// - No SLI focus
// - Mixed abstraction levels

type MetricsV2 struct {
	RequestsTotal      *prometheus.CounterVec
	ErrorsTotal        *prometheus.CounterVec
	RequestDuration    *prometheus.HistogramVec
	ActiveRequests     prometheus.Gauge
	SubscriptionsTotal prometheus.Counter // Better: business metric
}

func NewMetricsV2(serviceName string, registry *prometheus.Registry) *MetricsV2 {
	m := &MetricsV2{}

	// Better: Has labels but inconsistent
	m.RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: serviceName + "_v2_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint"}, // Good: basic labels
	)

	// Better: Error classification but inconsistent with requests
	m.ErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: serviceName + "_v2_errors_total",
			Help: "Total number of errors",
		},
		[]string{"method", "status_code"}, // Inconsistent: different labels than requests
	)

	// Better: Has timing but wrong buckets
	m.RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    serviceName + "_v2_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets, // Problem: wrong buckets for API
		},
		[]string{"method"}, // Problem: missing endpoint label
	)

	// Good: Active requests gauge
	m.ActiveRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: serviceName + "_v2_active_requests",
			Help: "Number of requests currently being processed",
		},
	)

	// Better: Business metric but too simple
	m.SubscriptionsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: serviceName + "_v2_subscriptions_created_total",
			Help: "Total subscriptions created",
		},
	)

	// Register with provided registry (better practice)
	if registry != nil {
		registry.MustRegister(
			m.RequestsTotal,
			m.ErrorsTotal,
			m.RequestDuration,
			m.ActiveRequests,
			m.SubscriptionsTotal,
		)
	}

	return m
}

// V2 Handler - Better metrics collection but still inconsistent
func InstrumentHandlerV2(next http.HandlerFunc, metrics *MetricsV2) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		ctx := r.Context()
		tracer := otel.Tracer("http-middleware-v2")
		ctx, span := tracer.Start(ctx, fmt.Sprintf("V2 %s %s", r.Method, r.URL.Path),
			trace.WithAttributes(
				attribute.String("version", "v2"),
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.Path),
			))
		defer span.End()

		// Better: Track active requests
		metrics.ActiveRequests.Inc()
		defer metrics.ActiveRequests.Dec()

		// Better: Count with labels
		metrics.RequestsTotal.WithLabelValues(r.Method, r.URL.Path).Inc()

		wrapped := &ResponseWriter{ResponseWriter: w, Status: http.StatusOK}
		next.ServeHTTP(wrapped, r.WithContext(ctx))

		duration := time.Since(startTime).Seconds()

		// Problem: Inconsistent labels (missing endpoint)
		metrics.RequestDuration.WithLabelValues(r.Method).Observe(duration)

		// Problem: Inconsistent error labeling
		if wrapped.Status >= 400 {
			metrics.ErrorsTotal.WithLabelValues(r.Method, fmt.Sprintf("%d", wrapped.Status)).Inc()
		}

		// Missing: No success/failure rate tracking
		// Missing: No percentile tracking
		// Missing: No business metric correlation
	}
}
