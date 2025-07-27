package models

import "time"

type Subscription struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Plan      string    `json:"plan"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

type PaymentRequest struct {
	SubscriptionID string  `json:"subscription_id"`
	Amount         float64 `json:"amount"`
	Plan           string  `json:"plan"`
}

type PaymentResponse struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func GetPlanPrice(plan string) float64 {
	switch plan {
	case "basic":
		return 10.0
	case "premium":
		return 20.0
	default:
		return 0.0
	}
}

func IsValidPlan(plan string) bool {
	return plan == "basic" || plan == "premium"
}
