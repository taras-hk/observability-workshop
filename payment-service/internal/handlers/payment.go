package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"payment-service/internal/models"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type PaymentHandler struct {
	deps *Dependencies
}

func NewPaymentHandler(deps *Dependencies) *PaymentHandler {
	return &PaymentHandler{deps: deps}
}

func (h *PaymentHandler) ProcessPayment(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	propagator := otel.GetTextMapPropagator()
	ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

	h.deps.Logger.Debug().
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("remote_addr", r.RemoteAddr).
		Msg("Processing payment request")

	if r.Method != http.MethodPost {
		h.deps.Logger.Warn().
			Str("method", r.Method).
			Msg("Invalid HTTP method for payment processing")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.deps.Logger.Error().
			Err(err).
			Msg("Failed to decode payment request")
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	response, err := h.deps.Processor.ProcessPayment(ctx, req)
	if err != nil {
		h.handlePaymentError(w, err, req, startTime)
		return
	}

	if h.deps.Metrics != nil {
		h.deps.Metrics.PaymentsProcessed.WithLabelValues(req.Plan, response.Status).Inc()
	}

	h.deps.Logger.Info().
		Str("payment_id", response.ID).
		Str("status", response.Status).
		Dur("duration", time.Since(startTime)).
		Msg("Payment processed successfully")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.deps.Logger.Error().
			Err(err).
			Str("payment_id", response.ID).
			Msg("Failed to encode payment response")
	}
}

func (h *PaymentHandler) handlePaymentError(w http.ResponseWriter, err error, req models.PaymentRequest, startTime time.Time) {
	if h.deps.Metrics != nil {
		h.deps.Metrics.ErrorsTotal.WithLabelValues("POST", "payment_processing").Inc()
	}

	h.deps.Logger.Error().
		Err(err).
		Str("subscription_id", req.SubscriptionID).
		Dur("duration", time.Since(startTime)).
		Msg("Payment processing failed")

	if paymentErr, ok := err.(models.PaymentError); ok {
		status := http.StatusBadRequest
		if paymentErr.Type == models.ErrorTypeProcessingError ||
			paymentErr.Type == models.ErrorTypeNetworkError ||
			paymentErr.Type == models.ErrorTypeTimeout {
			status = http.StatusInternalServerError
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"code":    paymentErr.Code,
				"message": paymentErr.Message,
				"type":    paymentErr.Type,
			},
		})
		return
	}

	http.Error(w, "Payment processing failed", http.StatusInternalServerError)
}

func (h *PaymentHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.deps.Processor.HealthCheck(ctx); err != nil {
		h.deps.Logger.Error().
			Err(err).
			Msg("Health check failed")
		http.Error(w, "Service unhealthy", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "healthy",
		"service":   "payment-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func RegisterRoutes(deps *Dependencies) {
	handler := NewPaymentHandler(deps)

	if deps.Metrics != nil {
		http.HandleFunc("/process", func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			deps.Metrics.RequestsTotal.WithLabelValues(r.Method, "/process").Inc()
			deps.Metrics.ActiveRequests.Inc()
			defer func() {
				deps.Metrics.ActiveRequests.Dec()
				duration := time.Since(start).Seconds()
				deps.Metrics.RequestDuration.WithLabelValues(r.Method, "/process").Observe(duration)
			}()
			handler.ProcessPayment(w, r)
		})
	} else {
		http.HandleFunc("/process", handler.ProcessPayment)
	}

	http.HandleFunc("/health", handler.HealthCheck)

	deps.Logger.Info().Msg("Payment service routes registered")
}
