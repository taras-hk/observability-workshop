package models

import (
	"fmt"
	"math/rand"
	"time"
)

type PaymentRequest struct {
	SubscriptionID string  `json:"subscription_id"`
	Amount         float64 `json:"amount"`
	Plan           string  `json:"plan"`
	Currency       string  `json:"currency,omitempty"`
	Method         string  `json:"method,omitempty"`
}

type PaymentResponse struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"`
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	ProcessedAt time.Time `json:"processed_at"`
	Fees        float64   `json:"fees,omitempty"`
}

type PaymentError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

func (e PaymentError) Error() string {
	return fmt.Sprintf("payment error [%s]: %s", e.Code, e.Message)
}

const (
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusPending   = "pending"
	StatusCancelled = "cancelled"
)

const (
	ErrorTypeInsufficientFunds = "insufficient_funds"
	ErrorTypeInvalidCard       = "invalid_card"
	ErrorTypeNetworkError      = "network_error"
	ErrorTypeProcessingError   = "processing_error"
	ErrorTypeTimeout           = "timeout"
)

func ValidatePaymentRequest(req PaymentRequest) error {
	if req.SubscriptionID == "" {
		return PaymentError{
			Code:    "MISSING_SUBSCRIPTION_ID",
			Message: "subscription ID is required",
			Type:    "validation_error",
		}
	}

	if req.Amount <= 0 {
		return PaymentError{
			Code:    "INVALID_AMOUNT",
			Message: "amount must be greater than 0",
			Type:    "validation_error",
		}
	}

	if req.Plan == "" {
		return PaymentError{
			Code:    "MISSING_PLAN",
			Message: "plan is required",
			Type:    "validation_error",
		}
	}

	return nil
}

func GeneratePaymentID() string {
	return fmt.Sprintf("pmt_%d_%d", time.Now().UnixNano(), rand.Int31())
}

func CalculateFees(amount float64, plan string) float64 {
	baseRate := 0.029 // 2.9%

	switch plan {
	case "premium":
		baseRate = 0.025
	case "enterprise":
		baseRate = 0.02
	}

	fee := amount * baseRate

	if fee < 0.30 {
		fee = 0.30
	}

	return fee
}

func ShouldSimulateFailure(failureRate float64) bool {
	return rand.Float64() < failureRate
}

func GetRandomFailureType() PaymentError {
	failures := []PaymentError{
		{
			Code:    "INSUFFICIENT_FUNDS",
			Message: "insufficient funds in account",
			Type:    ErrorTypeInsufficientFunds,
		},
		{
			Code:    "INVALID_CARD",
			Message: "invalid or expired card",
			Type:    ErrorTypeInvalidCard,
		},
		{
			Code:    "NETWORK_ERROR",
			Message: "network connection failed",
			Type:    ErrorTypeNetworkError,
		},
		{
			Code:    "PROCESSING_ERROR",
			Message: "payment processor temporarily unavailable",
			Type:    ErrorTypeProcessingError,
		},
		{
			Code:    "TIMEOUT",
			Message: "payment processing timeout",
			Type:    ErrorTypeTimeout,
		},
	}

	return failures[rand.Intn(len(failures))]
}
