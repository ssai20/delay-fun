package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func InitTracing(cfg *Config) (*sdktrace.TracerProvider, error) {
	if !cfg.TraceEnabled {
		return nil, nil
	}

	ctx := context.Background()

	// Создаем экспортер
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(cfg.TraceEndpoint),
		otlptracehttp.WithInsecure(), // Для разработки, в production используйте TLS
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Создаем ресурс с информацией о сервисе
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.TraceServiceName),
			semconv.ServiceVersion("1.0.0"),
			attribute.String("environment", cfg.TraceEnvironment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Создаем TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // В production используйте ProbabilisticSampler
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer = tp.Tracer(cfg.TraceServiceName)

	return tp, nil
}

func GetTracer() trace.Tracer {
	if tracer == nil {
		return trace.NewNoopTracerProvider().Tracer("noop")
	}
	return tracer
}

// Helper functions for common tracing patterns
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return GetTracer().Start(ctx, name, opts...)
}

func RecordError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("error", true))
	}
}
