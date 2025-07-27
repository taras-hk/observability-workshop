package observability

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

type TracerConfig struct {
	ServiceName    string
	JaegerEndpoint string
	SampleRatio    float64
}

func InitTracer(cfg TracerConfig) (*tracesdk.TracerProvider, error) {
	if cfg.JaegerEndpoint == "" {
		cfg.JaegerEndpoint = "http://jaeger:14268/api/traces"
	}
	if cfg.SampleRatio == 0 {
		cfg.SampleRatio = 0.2
	}

	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(cfg.JaegerEndpoint)))
	if err != nil {
		return nil, err
	}

	sampler := tracesdk.ParentBased(
		tracesdk.TraceIDRatioBased(cfg.SampleRatio),
	)

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(sampler),
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
		)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, nil
}

func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
