package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"payment-service/internal/config"
	"payment-service/internal/handlers"
	"payment-service/internal/services"

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

	metrics := initMetrics(logger)

	processor := services.NewPaymentProcessor(cfg, logger)

	deps := handlers.NewDependencies(cfg, logger, processor, metrics)

	registerRoutes(deps)

	logger.Info().
		Str("port", cfg.Port).
		Msg("Starting payment service server")

	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}

func initLogger(cfg *config.Config) zerolog.Logger {
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
		NoColor:    false,
	}

	var writers []io.Writer
	writers = append(writers, consoleWriter)

	if cfg.LoggingEnabled {
		logstashWriter, err := observe.NewLogWriter(observe.LogConfig{
			Host: cfg.LogstashHost,
		}, func(err error) {
			log.Printf("Logstash error: %v", err)
		})
		if err == nil {
			writers = append(writers, logstashWriter)
		}
	}

	logger := zerolog.New(zerolog.MultiLevelWriter(writers...)).
		With().
		Timestamp().
		Caller().
		Str("service", "payment-service").
		Logger().
		Level(zerolog.DebugLevel)

	logger.Info().
		Bool("metrics_enabled", cfg.MetricsEnabled).
		Bool("tracing_enabled", cfg.TracingEnabled).
		Bool("logging_enabled", cfg.LoggingEnabled).
		Msg("Logger initialized")

	return logger
}

func initTracing(cfg *config.Config, logger zerolog.Logger) *tracesdk.TracerProvider {
	if !cfg.TracingEnabled {
		logger.Info().Msg("Tracing disabled")
		return nil
	}

	tp, err := observe.InitTracer(observe.TracerConfig{
		ServiceName:    "payment-service",
		JaegerEndpoint: cfg.JaegerEndpoint,
		SampleRatio:    1.0,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize tracer")
	}

	logger.Info().Msg("Tracer initialized")
	return tp
}

func shutdownTracing(tp *tracesdk.TracerProvider, logger zerolog.Logger) {
	if tp == nil {
		return
	}
	
	if err := tp.Shutdown(context.Background()); err != nil {
		logger.Error().Err(err).Msg("Error shutting down tracer provider")
	}
}

func initMetrics(logger zerolog.Logger) *observe.Metrics {
	metrics := observe.NewMetrics(observe.MetricsConfig{
		ServiceName: "payment_service",
		Registry:    nil, // Use default registry
	})

	logger.Info().Msg("Metrics initialized")
	return metrics
}

func registerRoutes(deps *handlers.Dependencies) {
	http.Handle("/metrics", promhttp.Handler())
	
	handlers.RegisterRoutes(deps)

	deps.Logger.Info().Msg("All routes registered")
}
