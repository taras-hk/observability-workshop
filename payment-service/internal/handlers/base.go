package handlers

import (
	"payment-service/internal/config"
	"payment-service/internal/services"

	observe "observability"

	"github.com/rs/zerolog"
)

type Dependencies struct {
	Config    *config.Config
	Logger    zerolog.Logger
	Processor *services.PaymentProcessor
	Metrics   *observe.Metrics
}

func NewDependencies(
	cfg *config.Config,
	logger zerolog.Logger,
	processor *services.PaymentProcessor,
	metrics *observe.Metrics,
) *Dependencies {
	return &Dependencies{
		Config:    cfg,
		Logger:    logger,
		Processor: processor,
		Metrics:   metrics,
	}
}
