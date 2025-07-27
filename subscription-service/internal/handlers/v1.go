package handlers

import (
	"encoding/json"
	"net/http"

	"subscription-service/internal/models"

	observe "observability"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type V1Handler struct {
	deps *Dependencies
}

func NewV1Handler(deps *Dependencies) *V1Handler {
	return &V1Handler{deps: deps}
}

func (h *V1Handler) HandleSubscriptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createSubscription(w, r)
	case http.MethodGet:
		h.getSubscriptions(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *V1Handler) HandleSubscriptionByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/v1/subscriptions/"):]

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

func (h *V1Handler) createSubscription(w http.ResponseWriter, r *http.Request) {
	h.deps.Logger.Info().Str("version", "v1").Msg("creating subscription")

	var reqData struct {
		UserID string `json:"user_id"`
		Plan   string `json:"plan"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		h.deps.Logger.Error().Err(err).Str("version", "v1").Msg("bad request")
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if reqData.UserID == "" || reqData.Plan == "" {
		h.deps.Logger.Warn().Str("version", "v1").Msg("missing fields")
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	if !models.IsValidPlan(reqData.Plan) {
		h.deps.Logger.Warn().Str("version", "v1").Str("plan", reqData.Plan).Msg("invalid plan")
		http.Error(w, "Invalid plan", http.StatusBadRequest)
		return
	}

	h.deps.TracingV1.TraceOperation("create_subscription", func() {
		h.deps.Logger.Debug().Str("version", "v1").Msg("Creating subscription")
	})

	sub := h.deps.Repository.Create(reqData.UserID, reqData.Plan)

	paymentReq := models.PaymentRequest{
		SubscriptionID: sub.ID,
		Amount:         models.GetPlanPrice(sub.Plan),
		Plan:           sub.Plan,
	}

	ctx := r.Context()
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(make(http.Header)))

	_, err := h.deps.PaymentService.ProcessPayment(ctx, paymentReq)
	if err != nil {
		h.deps.Logger.Error().Err(err).Str("version", "v1").Msg("payment failed")

		http.Error(w, "Payment processing failed", http.StatusInternalServerError)
		return
	}

	h.deps.Logger.Info().Str("version", "v1").Str("subscription_id", sub.ID).Msg("subscription created")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub)
}

func (h *V1Handler) getSubscriptions(w http.ResponseWriter, r *http.Request) {
	h.deps.Logger.Info().Str("version", "v1").Msg("getting subscriptions")

	subs := h.deps.Repository.GetAll()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subs)
}

func (h *V1Handler) getSubscription(w http.ResponseWriter, r *http.Request, id string) {
	h.deps.Logger.Info().Str("version", "v1").Str("subscription_id", id).Msg("getting subscription")

	sub, exists := h.deps.Repository.GetByID(id)
	if !exists {
		h.deps.Logger.Warn().Str("version", "v1").Str("subscription_id", id).Msg("not found")
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	h.deps.Logger.Info().Str("version", "v1").Str("subscription_id", id).Msg("found subscription")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub)
}

func (h *V1Handler) updateSubscription(w http.ResponseWriter, r *http.Request, id string) {
	h.deps.Logger.Info().Str("version", "v1").Str("subscription_id", id).Msg("updating subscription")

	var reqData struct {
		UserID string `json:"user_id"`
		Plan   string `json:"plan"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		h.deps.Logger.Error().Err(err).Str("version", "v1").Msg("decode error")
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if !models.IsValidPlan(reqData.Plan) {
		h.deps.Logger.Warn().Str("version", "v1").Str("plan", reqData.Plan).Msg("invalid plan")
		http.Error(w, "Invalid plan", http.StatusBadRequest)
		return
	}

	sub, exists := h.deps.Repository.Update(id, reqData.UserID, reqData.Plan)
	if !exists {
		h.deps.Logger.Warn().Str("version", "v1").Str("subscription_id", id).Msg("not found")
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	h.deps.Logger.Info().Str("version", "v1").Str("subscription_id", id).Msg("updated")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub)
}

func (h *V1Handler) deleteSubscription(w http.ResponseWriter, r *http.Request, id string) {
	h.deps.Logger.Info().Str("version", "v1").Str("subscription_id", id).Msg("deleting subscription")

	_, exists := h.deps.Repository.Delete(id)
	if !exists {
		h.deps.Logger.Warn().Str("version", "v1").Str("subscription_id", id).Msg("not found")
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	h.deps.Logger.Info().Str("version", "v1").Str("subscription_id", id).Msg("deleted")
	w.WriteHeader(http.StatusNoContent)
}

func RegisterV1Routes(deps *Dependencies) {
	handler := NewV1Handler(deps)

	http.HandleFunc("/v1/subscriptions", deps.TracingV1.InstrumentHandler(observe.InstrumentHandlerV1(handler.HandleSubscriptions, deps.MetricsV1)))
	http.HandleFunc("/v1/subscriptions/", deps.TracingV1.InstrumentHandler(observe.InstrumentHandlerV1(handler.HandleSubscriptionByID, deps.MetricsV1)))
}
