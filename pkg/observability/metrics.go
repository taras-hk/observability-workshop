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

type MetricsConfig struct {
	ServiceName string
	Labels      []string
	Registry    *prometheus.Registry
}

type Metrics struct {
	QueueLength        prometheus.Gauge
	UnsubscribesByPlan *prometheus.CounterVec
	RequestsTotal      *prometheus.CounterVec
	ErrorsTotal        *prometheus.CounterVec
	RequestDuration    *prometheus.HistogramVec
	ActiveRequests     prometheus.Gauge
	PaymentsProcessed  *prometheus.CounterVec
}

type ResponseWriter struct {
	http.ResponseWriter
	Status int
}

func (rw *ResponseWriter) WriteHeader(code int) {
	rw.Status = code
	rw.ResponseWriter.WriteHeader(code)
}

func NewMetrics(cfg MetricsConfig) *Metrics {
	m := &Metrics{}

	commonLabels := append([]string{}, cfg.Labels...)

	m.QueueLength = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: cfg.ServiceName + "_queue_length",
		Help: "The number of goroutines in the queue",
	})

	m.PaymentsProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: cfg.ServiceName + "_payments_processed_total",
			Help: "Total number of processed payments",
		},
		[]string{"plan", "status"},
	)

	m.UnsubscribesByPlan = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: cfg.ServiceName + "_unsubscribes_by_plan",
			Help: "Number of unsubscribes by plan type",
		},
		[]string{"plan"},
	)

	m.RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_requests_total", cfg.ServiceName),
			Help: "Total number of requests by method and endpoint",
		},
		append([]string{"method", "endpoint"}, commonLabels...),
	)

	m.ErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_errors_total", cfg.ServiceName),
			Help: "Total number of errors by method and type",
		},
		append([]string{"method", "error_type"}, commonLabels...),
	)

	m.RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    fmt.Sprintf("%s_request_duration_seconds", cfg.ServiceName),
			Help:    "Request duration distribution",
			Buckets: prometheus.DefBuckets,
		},
		append([]string{"method", "endpoint"}, commonLabels...),
	)

	m.ActiveRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: fmt.Sprintf("%s_active_requests", cfg.ServiceName),
			Help: "Number of requests currently being processed",
		},
	)

	if cfg.Registry != nil {
		cfg.Registry.MustRegister(
			m.QueueLength,
			m.PaymentsProcessed,
			m.UnsubscribesByPlan,
			m.RequestsTotal,
			m.ErrorsTotal,
			m.RequestDuration,
			m.ActiveRequests,
		)
	} else {
		prometheus.MustRegister(
			m.QueueLength,
			m.UnsubscribesByPlan,
			m.RequestsTotal,
			m.ErrorsTotal,
			m.RequestDuration,
			m.ActiveRequests,
		)
	}

	return m
}

func InstrumentHandler(next http.HandlerFunc, metrics *Metrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		ctx := r.Context()
		tracer := otel.Tracer("http-middleware")

		ctx, span := tracer.Start(ctx, fmt.Sprintf("%s %s", r.Method, r.URL.Path),
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.Path),
			))
		defer span.End()

		metrics.ActiveRequests.Inc()
		defer metrics.ActiveRequests.Dec()

		metrics.RequestsTotal.WithLabelValues(r.Method, r.URL.Path).Inc()

		wrapped := &ResponseWriter{ResponseWriter: w, Status: http.StatusOK}
		next.ServeHTTP(wrapped, r.WithContext(ctx))

		duration := time.Since(startTime).Seconds()
		metrics.RequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)

		if wrapped.Status >= 400 {
			metrics.ErrorsTotal.WithLabelValues(r.Method, fmt.Sprintf("http_%d", wrapped.Status)).Inc()
		}
	}
}
