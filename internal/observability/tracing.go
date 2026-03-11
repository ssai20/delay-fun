package observability

import (
	"context"
	"fmt"
	"go.uber.org/zap"
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
	logger := GetLogger() // Получаем логгер
	logger.Info("InitTracing started", zap.Bool("enabled", cfg.TraceEnabled))

	if !cfg.TraceEnabled {
		logger.Info("Tracing is disabled by configuration")
		return nil, nil
	}

	logger.Info("Step 1: Creating context")
	ctx := context.Background()

	logger.Info("Step 2: Creating OTLP HTTP exporter",
		zap.String("endpoint", cfg.TraceEndpoint),
		zap.Bool("insecure", true),
	)
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(cfg.TraceEndpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		logger.Error("Step 2 FAILED: Failed to create trace exporter", zap.Error(err))
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}
	logger.Info("Step 2 SUCCESS: Exporter created")

	logger.Info("Step 3: Creating resource",
		zap.String("service.name", cfg.TraceServiceName),
		zap.String("environment", cfg.TraceEnvironment),
	)
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.TraceServiceName),
			semconv.ServiceVersion("1.0.0"),
			attribute.String("environment", cfg.TraceEnvironment),
		),
	)
	if err != nil {
		logger.Error("Step 3 FAILED: Failed to create resource", zap.Error(err))
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}
	logger.Info("Step 3 SUCCESS: Resource created")

	logger.Info("Step 4: Creating TracerProvider")
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	logger.Info("Step 4 SUCCESS: TracerProvider created")

	logger.Info("Step 5: Setting global providers")
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	logger.Info("Step 5 SUCCESS: Global providers set")

	tracer = tp.Tracer(cfg.TraceServiceName)
	logger.Info("✅ InitTracing completed successfully")

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
