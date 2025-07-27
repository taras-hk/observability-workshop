package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"subscription-service/internal/config"
	"subscription-service/internal/handlers"
	"subscription-service/internal/services"

	observe "observability"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	cfg := config.NewConfig()

	logger := initLogger(cfg)

	tp := initTracing(cfg, logger)
	defer shutdownTracing(tp, logger)

	metricsV1, metricsV2, metricsV3 := initMetrics(logger)

	tracingV1, tracingV2, tracingV3 := initTracingVersions(logger)

	repository := services.NewSubscriptionRepository()
	paymentService := services.NewPaymentService(cfg.PaymentServiceURL)

	deps := handlers.NewDependencies(
		cfg,
		logger,
		repository,
		paymentService,
		metricsV1,
		metricsV2,
		metricsV3,
		tracingV1,
		tracingV2,
		tracingV3,
	)

	registerRoutes(deps)

	logger.Info().
		Str("port", cfg.Port).
		Msg("Starting subscription service server")

	log.Fatal(http.ListenAndServe(cfg.Port, nil))
}

func initLogger(cfg *config.Config) zerolog.Logger {
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
		NoColor:    false,
	}

	var writers []io.Writer
	writers = append(writers, consoleWriter)

	logstashWriter, err := observe.NewLogWriter(observe.LogConfig{
		Host: cfg.LogstashHost,
	}, func(err error) {
		log.Printf("Logstash error: %v", err)
	})
	if err == nil {
		writers = append(writers, logstashWriter)
	}

	logger := zerolog.New(zerolog.MultiLevelWriter(writers...)).
		With().
		Timestamp().
		Caller().
		Str("service", "subscription-service").
		Logger().
		Level(zerolog.DebugLevel)

	logger.Info().
		Bool("logstash_enabled", err == nil).
		Msg("Logger initialized")

	return logger
}

func initTracing(cfg *config.Config, logger zerolog.Logger) *tracesdk.TracerProvider {
	tp, err := observe.InitTracer(observe.TracerConfig{
		ServiceName:    "subscription-service",
		JaegerEndpoint: cfg.JaegerEndpoint,
		SampleRatio:    0.2,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize tracer")
	}

	logger.Info().Msg("Tracer initialized")
	return tp
}

func shutdownTracing(tp *tracesdk.TracerProvider, logger zerolog.Logger) {
	if err := tp.Shutdown(context.Background()); err != nil {
		logger.Error().Err(err).Msg("Error shutting down tracer provider")
	}
}

func initMetrics(logger zerolog.Logger) (*observe.MetricsV1, *observe.MetricsV2, *observe.MetricsV3) {
	metricsV1 := observe.NewMetricsV1("subscription_service")

	metricsV2 := observe.NewMetricsV2("subscription_service", nil) // nil = use default registry

	metricsV3 := observe.NewMetricsV3("subscription_service", nil) // nil = use default registry

	logger.Info().Msg("Metrics initialized for all versions")
	return metricsV1, metricsV2, metricsV3
}

func initTracingVersions(logger zerolog.Logger) (*observe.TracingV1, *observe.TracingV2, *observe.TracingV3) {
	tracingV1 := observe.NewTracingV1("subscription_service")

	tracingV2 := observe.NewTracingV2("subscription_service")

	tracingV3 := observe.NewTracingV3(observe.TracingV3Config{
		ServiceName:    "subscription_service",
		ServiceVersion: "1.0.0",
		Environment:    "demo",
		DeploymentMode: "container",
		SampleRatio:    0.1,
		JaegerEndpoint: "http://jaeger:14268/api/traces",
		EnableMetrics:  true,
		EnableBaggage:  true,
	})

	logger.Info().Msg("Tracing initialized for all versions")
	return tracingV1, tracingV2, tracingV3
}

func registerRoutes(deps *handlers.Dependencies) {
	http.Handle("/metrics", promhttp.Handler())

	handlers.RegisterV1Routes(deps)
	handlers.RegisterV2Routes(deps)
	handlers.RegisterV3Routes(deps)

	deps.Logger.Info().Msg("Routes registered for all API versions (/v1, /v2, /v3)")
}
