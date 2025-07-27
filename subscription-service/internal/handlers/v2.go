package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"subscription-service/internal/models"

	observe "observability"
)

type V2Handler struct {
	deps *Dependencies
}

func NewV2Handler(deps *Dependencies) *V2Handler {
	return &V2Handler{deps: deps}
}

func (h *V2Handler) HandleSubscriptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createSubscription(w, r)
	case http.MethodGet:
		h.getSubscriptions(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *V2Handler) HandleSubscriptionByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/v2/subscriptions/"):]

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

func (h *V2Handler) createSubscription(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	h.deps.Logger.Info().Str("version", "v2").Str("method", "POST").Str("path", "/v2/subscriptions").Msg("Creating subscription")

	var reqData struct {
		UserID string `json:"user_id"`
		Plan   string `json:"plan"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		h.deps.Logger.Error().Err(err).Str("version", "v2").Str("error_type", "decode_error").Msg("Failed to decode request")
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if reqData.UserID == "" || reqData.Plan == "" {
		h.deps.Logger.Warn().Str("version", "v2").Msg("Missing required fields")
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	if !models.IsValidPlan(reqData.Plan) {
		h.deps.Logger.Warn().Str("version", "v2").Str("plan", reqData.Plan).Msg("Invalid plan")
		http.Error(w, "Invalid plan", http.StatusBadRequest)
		return
	}

	sub := h.deps.Repository.Create(reqData.UserID, reqData.Plan)

	h.deps.Logger.Debug().Str("version", "v2").Msgf("Processing payment - subscription_id=%s amount=%.2f plan=%s", sub.ID, models.GetPlanPrice(sub.Plan), sub.Plan)

	paymentReq := models.PaymentRequest{
		SubscriptionID: sub.ID,
		Amount:         models.GetPlanPrice(sub.Plan),
		Plan:           sub.Plan,
	}

	_, err := h.deps.PaymentService.ProcessPayment(r.Context(), paymentReq)
	if err != nil {
		h.deps.Logger.Error().Err(err).Str("version", "v2").Msgf("Payment request failed - subscription_id=%s error=%v", sub.ID, err)

		h.deps.Repository.Delete(sub.ID)
		http.Error(w, "Payment processing failed", http.StatusInternalServerError)
		return
	}

	h.deps.MetricsV2.SubscriptionsTotal.Inc()

	h.deps.Logger.Info().Str("version", "v2").Msgf("Subscription created successfully - subscription_id=%s user_id=%s plan=%s duration_ms=%d", sub.ID, sub.UserID, sub.Plan, time.Since(startTime).Milliseconds())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub)
}

func (h *V2Handler) getSubscriptions(w http.ResponseWriter, r *http.Request) {
	count := h.deps.Repository.Count()
	h.deps.Logger.Info().Str("version", "v2").Msgf("Getting all subscriptions - count=%d", count)

	subs := h.deps.Repository.GetAll()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subs)
}

func (h *V2Handler) getSubscription(w http.ResponseWriter, r *http.Request, id string) {
	h.deps.Logger.Info().Str("version", "v2").Msgf("Getting subscription - subscription_id=%s", id)

	sub, exists := h.deps.Repository.GetByID(id)
	if !exists {
		h.deps.Logger.Warn().Str("version", "v2").Msgf("Subscription not found - subscription_id=%s", id)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	h.deps.Logger.Info().Str("version", "v2").Msgf("Found subscription - subscription_id=%s user_id=%s plan=%s", sub.ID, sub.UserID, sub.Plan)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub)
}

func (h *V2Handler) updateSubscription(w http.ResponseWriter, r *http.Request, id string) {
	startTime := time.Now()
	h.deps.Logger.Info().Str("version", "v2").Msgf("Updating subscription - subscription_id=%s", id)

	var reqData struct {
		UserID string `json:"user_id"`
		Plan   string `json:"plan"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		h.deps.Logger.Error().Err(err).Str("version", "v2").Msgf("Failed to decode update request - subscription_id=%s", id)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if !models.IsValidPlan(reqData.Plan) {
		h.deps.Logger.Warn().Str("version", "v2").Str("plan", reqData.Plan).Msgf("Invalid plan for update - subscription_id=%s", id)
		http.Error(w, "Invalid plan", http.StatusBadRequest)
		return
	}

	oldSub, exists := h.deps.Repository.GetByID(id)
	if !exists {
		h.deps.Logger.Warn().Str("version", "v2").Msgf("Subscription not found for update - subscription_id=%s", id)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	sub, _ := h.deps.Repository.Update(id, reqData.UserID, reqData.Plan)

	h.deps.Logger.Info().Str("version", "v2").Msgf("Subscription updated successfully - subscription_id=%s old_plan=%s new_plan=%s duration_ms=%d", id, oldSub.Plan, sub.Plan, time.Since(startTime).Milliseconds())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub)
}

func (h *V2Handler) deleteSubscription(w http.ResponseWriter, r *http.Request, id string) {
	startTime := time.Now()
	h.deps.Logger.Info().Str("version", "v2").Msgf("Deleting subscription - subscription_id=%s", id)

	sub, exists := h.deps.Repository.Delete(id)
	if !exists {
		h.deps.Logger.Warn().Str("version", "v2").Msgf("Subscription not found for deletion - subscription_id=%s", id)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	h.deps.Logger.Info().Str("version", "v2").Msgf("Subscription deleted successfully - subscription_id=%s user_id=%s plan=%s duration_ms=%d", id, sub.UserID, sub.Plan, time.Since(startTime).Milliseconds())

	w.WriteHeader(http.StatusNoContent)
}

func RegisterV2Routes(deps *Dependencies) {
	handler := NewV2Handler(deps)

	http.HandleFunc("/v2/subscriptions", deps.TracingV2.InstrumentHandler(observe.InstrumentHandlerV2(handler.HandleSubscriptions, deps.MetricsV2)))
	http.HandleFunc("/v2/subscriptions/", deps.TracingV2.InstrumentHandler(observe.InstrumentHandlerV2(handler.HandleSubscriptionByID, deps.MetricsV2)))
}
