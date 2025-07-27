package observability

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// V3 Metrics - BEST PRACTICES (The right way)
// Improvements:
// - Consistent labeling strategy
// - SLI/SLO focused metrics
// - Rich business metrics
// - Proper histogram buckets
// - Clear metric taxonomy
// - Follows the RED method (Rate, Errors, Duration)
// - Follows the USE method (Utilization, Saturation, Errors) where applicable

type MetricsV3 struct {
	// SLI Metrics - Service Level Indicators
	HTTPRequestsTotal    *prometheus.CounterVec
	HTTPRequestDuration  *prometheus.HistogramVec
	HTTPRequestsInFlight prometheus.Gauge

	// Business Metrics - Domain specific
	SubscriptionsCreated  *prometheus.CounterVec
	SubscriptionsActive   prometheus.Gauge
	SubscriptionRevenue   *prometheus.CounterVec
	PaymentProcessingTime *prometheus.HistogramVec
	PaymentFailures       *prometheus.CounterVec

	// System Metrics - Resource utilization
	ServiceUptime  prometheus.Gauge
	GoroutineCount prometheus.Gauge

	// Error Metrics - Detailed error classification
	BusinessErrors  *prometheus.CounterVec
	TechnicalErrors *prometheus.CounterVec
}

func NewMetricsV3(serviceName string, registry *prometheus.Registry) *MetricsV3 {
	m := &MetricsV3{}

	// Consistent labeling scheme across all metrics
	httpLabels := []string{"method", "endpoint", "status_class"}
	businessLabels := []string{"plan", "region", "payment_method"}
	errorLabels := []string{"error_type", "error_code", "severity"}

	// SLI Metrics - Perfect for SLO definition
	m.HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: serviceName + "_v3_http_requests_total",
			Help: "Total number of HTTP requests (SLI: Request Rate)",
		},
		httpLabels,
	)

	// Proper buckets for API response times (SLI: Latency)
	m.HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    serviceName + "_v3_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds (SLI: Latency)",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}, // API-appropriate buckets
		},
		httpLabels,
	)

	m.HTTPRequestsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: serviceName + "_v3_http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed (SLI: Saturation)",
		},
	)

	// Business Metrics - Critical for business monitoring
	m.SubscriptionsCreated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: serviceName + "_v3_subscriptions_created_total",
			Help: "Total number of subscriptions created by plan and region",
		},
		businessLabels,
	)

	m.SubscriptionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: serviceName + "_v3_subscriptions_active_current",
			Help: "Current number of active subscriptions",
		},
	)

	m.SubscriptionRevenue = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: serviceName + "_v3_subscription_revenue_total",
			Help: "Total revenue from subscriptions in USD cents",
		},
		businessLabels,
	)

	// Payment-specific metrics
	m.PaymentProcessingTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    serviceName + "_v3_payment_processing_duration_seconds",
			Help:    "Time spent processing payments",
			Buckets: []float64{.1, .25, .5, 1, 2, 5, 10}, // Payment-specific buckets
		},
		[]string{"payment_method", "plan"},
	)

	m.PaymentFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: serviceName + "_v3_payment_failures_total",
			Help: "Total number of payment failures by reason",
		},
		[]string{"failure_reason", "payment_method", "plan"},
	)

	// System health metrics
	m.ServiceUptime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: serviceName + "_v3_service_uptime_seconds",
			Help: "Service uptime in seconds",
		},
	)

	m.GoroutineCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: serviceName + "_v3_goroutines_current",
			Help: "Current number of goroutines",
		},
	)

	// Detailed error classification
	m.BusinessErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: serviceName + "_v3_business_errors_total",
			Help: "Business logic errors (invalid plans, insufficient funds, etc.)",
		},
		errorLabels,
	)

	m.TechnicalErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: serviceName + "_v3_technical_errors_total",
			Help: "Technical errors (timeouts, connection failures, etc.)",
		},
		errorLabels,
	)

	// Register all metrics
	if registry != nil {
		registry.MustRegister(
			m.HTTPRequestsTotal,
			m.HTTPRequestDuration,
			m.HTTPRequestsInFlight,
			m.SubscriptionsCreated,
			m.SubscriptionsActive,
			m.SubscriptionRevenue,
			m.PaymentProcessingTime,
			m.PaymentFailures,
			m.ServiceUptime,
			m.GoroutineCount,
			m.BusinessErrors,
			m.TechnicalErrors,
		)
	} else {
		// Use default registry when nil is passed
		prometheus.MustRegister(
			m.HTTPRequestsTotal,
			m.HTTPRequestDuration,
			m.HTTPRequestsInFlight,
			m.SubscriptionsCreated,
			m.SubscriptionsActive,
			m.SubscriptionRevenue,
			m.PaymentProcessingTime,
			m.PaymentFailures,
			m.ServiceUptime,
			m.GoroutineCount,
			m.BusinessErrors,
			m.TechnicalErrors,
		)
	}

	// Initialize uptime
	m.ServiceUptime.SetToCurrentTime()

	return m
}

// V3 Handler - Best practice metrics collection
func InstrumentHandlerV3(next http.HandlerFunc, metrics *MetricsV3) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		ctx := r.Context()
		tracer := otel.Tracer("http-middleware-v3")
		ctx, span := tracer.Start(ctx, fmt.Sprintf("V3 %s %s", r.Method, r.URL.Path),
			trace.WithAttributes(
				attribute.String("version", "v3"),
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.Path),
			))
		defer span.End()

		// Track saturation
		metrics.HTTPRequestsInFlight.Inc()
		defer metrics.HTTPRequestsInFlight.Dec()

		// Update system metrics
		metrics.GoroutineCount.Set(float64(getGoroutineCount()))

		wrapped := &ResponseWriter{ResponseWriter: w, Status: http.StatusOK}
		next.ServeHTTP(wrapped, r.WithContext(ctx))

		duration := time.Since(startTime).Seconds()
		statusClass := getStatusClass(wrapped.Status)

		// Consistent labeling for all HTTP metrics
		labels := []string{r.Method, r.URL.Path, statusClass}

		// SLI metrics with consistent labels
		metrics.HTTPRequestsTotal.WithLabelValues(labels...).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(labels...).Observe(duration)

		// Detailed error classification
		if wrapped.Status >= 400 {
			if wrapped.Status >= 500 {
				// Technical error
				metrics.TechnicalErrors.WithLabelValues(
					"http_server_error",
					strconv.Itoa(wrapped.Status),
					"high",
				).Inc()
			} else {
				// Business error
				metrics.BusinessErrors.WithLabelValues(
					"http_client_error",
					strconv.Itoa(wrapped.Status),
					"medium",
				).Inc()
			}
		}
	}
}

// Helper function to classify HTTP status codes
func getStatusClass(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "2xx"
	case status >= 300 && status < 400:
		return "3xx"
	case status >= 400 && status < 500:
		return "4xx"
	case status >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}

// Helper function to get goroutine count
func getGoroutineCount() int {
	// This is a placeholder - in real implementation you'd use runtime.NumGoroutine()
	// but we want to avoid importing runtime in this example
	return 10
}
