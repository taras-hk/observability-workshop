package handlers

import (
	"subscription-service/internal/config"
	"subscription-service/internal/services"

	observe "observability"

	"github.com/rs/zerolog"
)

type Dependencies struct {
	Config         *config.Config
	Logger         zerolog.Logger
	Repository     *services.SubscriptionRepository
	PaymentService *services.PaymentService
	MetricsV1      *observe.MetricsV1
	MetricsV2      *observe.MetricsV2
	MetricsV3      *observe.MetricsV3
	TracingV1      *observe.TracingV1
	TracingV2      *observe.TracingV2
	TracingV3      *observe.TracingV3
}

func NewDependencies(
	cfg *config.Config,
	logger zerolog.Logger,
	repo *services.SubscriptionRepository,
	paymentService *services.PaymentService,
	metricsV1 *observe.MetricsV1,
	metricsV2 *observe.MetricsV2,
	metricsV3 *observe.MetricsV3,
	tracingV1 *observe.TracingV1,
	tracingV2 *observe.TracingV2,
	tracingV3 *observe.TracingV3,
) *Dependencies {
	return &Dependencies{
		Config:         cfg,
		Logger:         logger,
		Repository:     repo,
		PaymentService: paymentService,
		MetricsV1:      metricsV1,
		MetricsV2:      metricsV2,
		MetricsV3:      metricsV3,
		TracingV1:      tracingV1,
		TracingV2:      tracingV2,
		TracingV3:      tracingV3,
	}
}
