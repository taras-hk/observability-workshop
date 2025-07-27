package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port            string
	JaegerEndpoint  string
	LogstashHost    string
	ProcessingDelay time.Duration
	EnableFailures  bool
	FailureRate     float64
	MetricsEnabled  bool
	TracingEnabled  bool
	LoggingEnabled  bool
}

func NewConfig() *Config {
	cfg := &Config{
		Port:            getEnv("PORT", "8081"),
		JaegerEndpoint:  getEnv("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
		LogstashHost:    getEnv("LOGSTASH_HOST", "logstash:5000"),
		ProcessingDelay: getDurationEnv("PROCESSING_DELAY", 100*time.Millisecond),
		EnableFailures:  getBoolEnv("ENABLE_FAILURES", false),
		FailureRate:     getFloatEnv("FAILURE_RATE", 0.1),
		MetricsEnabled:  getBoolEnv("METRICS_ENABLED", true),
		TracingEnabled:  getBoolEnv("TRACING_ENABLED", true),
		LoggingEnabled:  getBoolEnv("LOGGING_ENABLED", true),
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getFloatEnv(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
