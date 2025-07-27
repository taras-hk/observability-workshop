package services

import (
	"context"
	"math/rand"
	"payment-service/internal/config"
	"payment-service/internal/models"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type PaymentProcessor struct {
	config *config.Config
	logger zerolog.Logger
	tracer trace.Tracer
}

func NewPaymentProcessor(cfg *config.Config, logger zerolog.Logger) *PaymentProcessor {
	return &PaymentProcessor{
		config: cfg,
		logger: logger,
		tracer: otel.Tracer("payment-processor"),
	}
}

func (p *PaymentProcessor) ProcessPayment(ctx context.Context, req models.PaymentRequest) (*models.PaymentResponse, error) {
	ctx, span := p.tracer.Start(ctx, "process_payment",
		trace.WithAttributes(
			attribute.String("subscription_id", req.SubscriptionID),
			attribute.String("plan", req.Plan),
			attribute.Float64("amount", req.Amount),
		))
	defer span.End()

	p.logger.Info().
		Str("subscription_id", req.SubscriptionID).
		Str("plan", req.Plan).
		Float64("amount", req.Amount).
		Msg("Processing payment request")

	if err := models.ValidatePaymentRequest(req); err != nil {
		p.logger.Error().
			Err(err).
			Str("subscription_id", req.SubscriptionID).
			Msg("Payment validation failed")

		span.RecordError(err)
		span.SetAttributes(attribute.String("error.type", "validation"))
		return nil, err
	}

	if p.config.ProcessingDelay > 0 {
		p.logger.Debug().
			Dur("delay", p.config.ProcessingDelay).
			Msg("Simulating processing delay")

		time.Sleep(p.config.ProcessingDelay)
	}

	if p.config.EnableFailures && models.ShouldSimulateFailure(p.config.FailureRate) {
		failure := models.GetRandomFailureType()

		p.logger.Warn().
			Str("failure_type", failure.Type).
			Str("failure_code", failure.Code).
			Str("subscription_id", req.SubscriptionID).
			Msg("Simulated payment failure")

		span.RecordError(failure)
		span.SetAttributes(
			attribute.String("error.type", failure.Type),
			attribute.String("error.code", failure.Code),
		)

		return &models.PaymentResponse{
			ID:          models.GeneratePaymentID(),
			Status:      models.StatusFailed,
			Amount:      req.Amount,
			Currency:    p.getCurrency(req),
			ProcessedAt: time.Now(),
		}, failure
	}

	response := &models.PaymentResponse{
		ID:          models.GeneratePaymentID(),
		Status:      models.StatusCompleted,
		Amount:      req.Amount,
		Currency:    p.getCurrency(req),
		ProcessedAt: time.Now(),
		Fees:        models.CalculateFees(req.Amount, req.Plan),
	}

	if rand.Float64() < 0.1 {
		extraDelay := time.Duration(rand.Intn(200)) * time.Millisecond
		time.Sleep(extraDelay)

		span.SetAttributes(attribute.Int64("processing.extra_delay_ms", extraDelay.Milliseconds()))
	}

	p.logger.Info().
		Str("payment_id", response.ID).
		Str("subscription_id", req.SubscriptionID).
		Str("status", response.Status).
		Float64("amount", response.Amount).
		Float64("fees", response.Fees).
		Msg("Payment processed successfully")

	span.SetAttributes(
		attribute.String("payment.id", response.ID),
		attribute.String("payment.status", response.Status),
		attribute.Float64("payment.fees", response.Fees),
	)

	return response, nil
}

func (p *PaymentProcessor) getCurrency(req models.PaymentRequest) string {
	if req.Currency != "" {
		return req.Currency
	}
	return "USD"
}

func (p *PaymentProcessor) HealthCheck(ctx context.Context) error {
	_, span := p.tracer.Start(ctx, "health_check")
	defer span.End()

	p.logger.Debug().Msg("Payment processor health check")

	time.Sleep(10 * time.Millisecond)

	return nil
}
