package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"subscription-service/internal/models"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type PaymentService struct {
	baseURL string
	client  *http.Client
}

func NewPaymentService(baseURL string) *PaymentService {
	return &PaymentService{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *PaymentService) ProcessPayment(ctx context.Context, req models.PaymentRequest) (*models.PaymentResponse, error) {
	paymentData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payment request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/payments", bytes.NewBuffer(paymentData))
	if err != nil {
		return nil, fmt.Errorf("failed to create payment request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(httpReq.Header))

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send payment request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("payment failed with status: %d", resp.StatusCode)
	}

	var paymentResp models.PaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&paymentResp); err != nil {
		return nil, fmt.Errorf("failed to decode payment response: %w", err)
	}

	return &paymentResp, nil
}
