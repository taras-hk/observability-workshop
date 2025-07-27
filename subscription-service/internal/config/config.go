package config

import (
	"os"
)

type Config struct {
	Port              string
	PaymentServiceURL string
	JaegerEndpoint    string
	LogstashHost      string
}

func NewConfig() *Config {
	cfg := &Config{
		Port:              ":8080",
		PaymentServiceURL: "http://payment-service:8081",
		JaegerEndpoint:    "",
		LogstashHost:      "localhost:5044",
	}

	if port := os.Getenv("PORT"); port != "" {
		cfg.Port = ":" + port
	}

	if paymentURL := os.Getenv("PAYMENT_SERVICE_URL"); paymentURL != "" {
		cfg.PaymentServiceURL = paymentURL
	}

	if jaegerEndpoint := os.Getenv("JAEGER_ENDPOINT"); jaegerEndpoint != "" {
		cfg.JaegerEndpoint = jaegerEndpoint
	}

	if logstashHost := os.Getenv("LOGSTASH_HOST"); logstashHost != "" {
		cfg.LogstashHost = logstashHost
	}

	return cfg
}
