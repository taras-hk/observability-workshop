package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"subscription-service/internal/models"

	observe "observability"
)

type V3Handler struct {
	deps *Dependencies
}

func NewV3Handler(deps *Dependencies) *V3Handler {
	return &V3Handler{deps: deps}
}

func (h *V3Handler) HandleSubscriptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createSubscription(w, r)
	case http.MethodGet:
		h.getSubscriptions(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleSubscriptionByID handles the /v3/subscriptions/{id} endpoint
func (h *V3Handler) HandleSubscriptionByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/v3/subscriptions/"):]

	switch r.Method {
	case http.MethodGet:
		h.getSubscription(w, r, id)
	case http.MethodPut:
		h.updateSubscription(w, r, id)
	case http.MethodDelete:
		h.deleteSubscription(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *V3Handler) createSubscription(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	ctx := r.Context()

	h.deps.Logger.Debug().
		Str("version", "v3").
		Str("method", "POST").
		Str("path", "/v3/subscriptions").
		Str("client_ip", r.RemoteAddr).
		Msg("Processing create subscription request")

	var reqData struct {
		UserID string `json:"user_id"`
		Plan   string `json:"plan"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		h.deps.Logger.Error().
			Err(err).
			Str("version", "v3").
			Str("method", "POST").
			Str("path", "/v3/subscriptions").
			Str("error_type", "decode_error").
			Str("client_ip", r.RemoteAddr).
			Dur("duration_ms", time.Since(startTime)).
			Msg("Failed to decode subscription request")
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if reqData.UserID == "" || reqData.Plan == "" {
		h.deps.Logger.Warn().
			Str("version", "v3").
			Str("method", "POST").
			Str("path", "/v3/subscriptions").
			Str("error_type", "validation_error").
			Bool("user_id_missing", reqData.UserID == "").
			Bool("plan_missing", reqData.Plan == "").
			Str("client_ip", r.RemoteAddr).
			Dur("duration_ms", time.Since(startTime)).
			Msg("Missing required fields in subscription request")
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	if !models.IsValidPlan(reqData.Plan) {
		h.deps.Logger.Warn().
			Str("version", "v3").
			Str("method", "POST").
			Str("path", "/v3/subscriptions").
			Str("error_type", "validation_error").
			Str("plan", reqData.Plan).
			Str("client_ip", r.RemoteAddr).
			Dur("duration_ms", time.Since(startTime)).
			Msg("Invalid subscription plan")

		h.deps.MetricsV3.BusinessErrors.WithLabelValues("validation_error", "invalid_plan", "warning").Inc()

		http.Error(w, "Invalid plan", http.StatusBadRequest)
		return
	}

	sub := h.deps.Repository.Create(reqData.UserID, reqData.Plan)

	h.deps.Logger.Debug().
		Str("version", "v3").
		Str("method", "POST").
		Str("path", "/v3/subscriptions").
		Str("subscription_id", sub.ID).
		Str("user_id", sub.UserID).
		Str("plan", sub.Plan).
		Float64("amount", models.GetPlanPrice(sub.Plan)).
		Str("client_ip", r.RemoteAddr).
		Msg("Processing payment for subscription")

	paymentReq := models.PaymentRequest{
		SubscriptionID: sub.ID,
		Amount:         models.GetPlanPrice(sub.Plan),
		Plan:           sub.Plan,
	}

	paymentErr := h.deps.TracingV3.TraceOperation(ctx, "process_payment", "business", map[string]interface{}{
		"subscription_id": sub.ID,
		"plan":            sub.Plan,
		"amount":          paymentReq.Amount,
		"user_id":         sub.UserID,
	}, func(ctx context.Context) error {
		_, err := h.deps.PaymentService.ProcessPayment(ctx, paymentReq)
		return err
	})

	if paymentErr != nil {
		h.deps.Logger.Error().
			Err(paymentErr).
			Str("version", "v3").
			Str("method", "POST").
			Str("path", "/v3/subscriptions").
			Str("subscription_id", sub.ID).
			Str("user_id", sub.UserID).
			Str("plan", sub.Plan).
			Float64("amount", paymentReq.Amount).
			Str("error_type", "payment_error").
			Str("client_ip", r.RemoteAddr).
			Dur("duration_ms", time.Since(startTime)).
			Msg("Payment processing failed")

		h.deps.Repository.Delete(sub.ID)

		h.deps.MetricsV3.PaymentFailures.WithLabelValues("payment_service_error", "unknown", "critical").Inc()

		http.Error(w, "Payment processing failed", http.StatusInternalServerError)
		return
	}

	h.deps.MetricsV3.SubscriptionsActive.Inc()
	h.deps.MetricsV3.SubscriptionsCreated.WithLabelValues(sub.Plan, "default", "credit_card").Inc()

	h.deps.Logger.Info().
		Str("version", "v3").
		Str("method", "POST").
		Str("path", "/v3/subscriptions").
		Str("subscription_id", sub.ID).
		Str("user_id", sub.UserID).
		Str("plan", sub.Plan).
		Float64("amount", paymentReq.Amount).
		Str("client_ip", r.RemoteAddr).
		Dur("duration_ms", time.Since(startTime)).
		Msg("Subscription created successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub)
}

func (h *V3Handler) getSubscriptions(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	count := h.deps.Repository.Count()

	h.deps.Logger.Debug().
		Str("version", "v3").
		Str("method", "GET").
		Str("path", "/v3/subscriptions").
		Int("subscriptions_count", count).
		Str("client_ip", r.RemoteAddr).
		Msg("Processing get all subscriptions request")

	subs := h.deps.Repository.GetAll()

	h.deps.Logger.Info().
		Str("version", "v3").
		Str("method", "GET").
		Str("path", "/v3/subscriptions").
		Int("subscriptions_returned", len(subs)).
		Str("client_ip", r.RemoteAddr).
		Dur("duration_ms", time.Since(startTime)).
		Msg("Subscriptions retrieved successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subs)
}

func (h *V3Handler) getSubscription(w http.ResponseWriter, r *http.Request, id string) {
	startTime := time.Now()

	h.deps.Logger.Debug().
		Str("version", "v3").
		Str("method", "GET").
		Str("path", "/v3/subscriptions/{id}").
		Str("subscription_id", id).
		Str("client_ip", r.RemoteAddr).
		Msg("Processing get subscription request")

	sub, exists := h.deps.Repository.GetByID(id)
	if !exists {
		h.deps.Logger.Warn().
			Str("version", "v3").
			Str("method", "GET").
			Str("path", "/v3/subscriptions/{id}").
			Str("subscription_id", id).
			Str("client_ip", r.RemoteAddr).
			Dur("duration_ms", time.Since(startTime)).
			Msg("Subscription not found")
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	h.deps.Logger.Info().
		Str("version", "v3").
		Str("method", "GET").
		Str("path", "/v3/subscriptions/{id}").
		Str("subscription_id", id).
		Str("user_id", sub.UserID).
		Str("plan", sub.Plan).
		Str("client_ip", r.RemoteAddr).
		Dur("duration_ms", time.Since(startTime)).
		Msg("Subscription retrieved successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub)
}

func (h *V3Handler) updateSubscription(w http.ResponseWriter, r *http.Request, id string) {
	startTime := time.Now()

	h.deps.Logger.Debug().
		Str("version", "v3").
		Str("method", "PUT").
		Str("path", "/v3/subscriptions/{id}").
		Str("subscription_id", id).
		Str("client_ip", r.RemoteAddr).
		Msg("Processing update subscription request")

	var reqData struct {
		UserID string `json:"user_id"`
		Plan   string `json:"plan"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		h.deps.Logger.Error().
			Err(err).
			Str("version", "v3").
			Str("method", "PUT").
			Str("path", "/v3/subscriptions/{id}").
			Str("subscription_id", id).
			Str("error_type", "decode_error").
			Str("client_ip", r.RemoteAddr).
			Dur("duration_ms", time.Since(startTime)).
			Msg("Failed to decode update request")
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if !models.IsValidPlan(reqData.Plan) {
		h.deps.Logger.Warn().
			Str("version", "v3").
			Str("method", "PUT").
			Str("path", "/v3/subscriptions/{id}").
			Str("subscription_id", id).
			Str("plan", reqData.Plan).
			Str("error_type", "validation_error").
			Str("client_ip", r.RemoteAddr).
			Dur("duration_ms", time.Since(startTime)).
			Msg("Invalid plan for subscription update")
		http.Error(w, "Invalid plan", http.StatusBadRequest)
		return
	}

	oldSub, exists := h.deps.Repository.GetByID(id)
	if !exists {
		h.deps.Logger.Warn().
			Str("version", "v3").
			Str("method", "PUT").
			Str("path", "/v3/subscriptions/{id}").
			Str("subscription_id", id).
			Str("client_ip", r.RemoteAddr).
			Dur("duration_ms", time.Since(startTime)).
			Msg("Subscription not found for update")
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	sub, _ := h.deps.Repository.Update(id, reqData.UserID, reqData.Plan)

	h.deps.Logger.Info().
		Str("version", "v3").
		Str("method", "PUT").
		Str("path", "/v3/subscriptions/{id}").
		Str("subscription_id", id).
		Str("user_id", sub.UserID).
		Str("old_plan", oldSub.Plan).
		Str("new_plan", sub.Plan).
		Str("client_ip", r.RemoteAddr).
		Dur("duration_ms", time.Since(startTime)).
		Msg("Subscription updated successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub)
}

func (h *V3Handler) deleteSubscription(w http.ResponseWriter, r *http.Request, id string) {
	startTime := time.Now()

	h.deps.Logger.Debug().
		Str("version", "v3").
		Str("method", "DELETE").
		Str("path", "/v3/subscriptions/{id}").
		Str("subscription_id", id).
		Str("client_ip", r.RemoteAddr).
		Msg("Processing delete subscription request")

	sub, exists := h.deps.Repository.Delete(id)
	if !exists {
		h.deps.Logger.Warn().
			Str("version", "v3").
			Str("method", "DELETE").
			Str("path", "/v3/subscriptions/{id}").
			Str("subscription_id", id).
			Str("client_ip", r.RemoteAddr).
			Dur("duration_ms", time.Since(startTime)).
			Msg("Subscription not found for deletion")
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	h.deps.MetricsV3.SubscriptionsActive.Dec()

	if h.deps.Repository.Count() < 10 {
		h.deps.Logger.Warn().
			Str("version", "v3").
			Int("subscriptions_count", h.deps.Repository.Count()).
			Msg("Subscription count is getting low")
	}

	h.deps.Logger.Info().
		Str("version", "v3").
		Str("method", "DELETE").
		Str("path", "/v3/subscriptions/{id}").
		Str("subscription_id", id).
		Str("user_id", sub.UserID).
		Str("plan", sub.Plan).
		Str("client_ip", r.RemoteAddr).
		Dur("duration_ms", time.Since(startTime)).
		Msg("Subscription deleted successfully")

	w.WriteHeader(http.StatusNoContent)
}

func RegisterV3Routes(deps *Dependencies) {
	handler := NewV3Handler(deps)

	http.HandleFunc("/v3/subscriptions", deps.TracingV3.InstrumentHandler(observe.InstrumentHandlerV3(handler.HandleSubscriptions, deps.MetricsV3)))
	http.HandleFunc("/v3/subscriptions/", deps.TracingV3.InstrumentHandler(observe.InstrumentHandlerV3(handler.HandleSubscriptionByID, deps.MetricsV3)))
}
