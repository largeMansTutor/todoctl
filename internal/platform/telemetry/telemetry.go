package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/thetrollfarmercodes/todoctl/todo/internal/platform/config"
)

// ConfigureTracerProvider sets up OpenTelemetry tracing. If no exporter endpoint
// is configured, a noop provider is installed.
func ConfigureTracerProvider(lc fx.Lifecycle, cfg *config.Config, logger *zap.Logger) (*sdktrace.TracerProvider, error) {
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.OTELServiceName),
		),
		resource.WithProcess(),
		resource.WithOS(),
	)
	if err != nil {
		return nil, err
	}

	var tp *sdktrace.TracerProvider

	if cfg.OTELExporterEndpoint == "" {
		tp = sdktrace.NewTracerProvider(sdktrace.WithResource(res))
	} else {
		client := otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(cfg.OTELExporterEndpoint),
			otlptracehttp.WithInsecure(),
			otlptracehttp.WithTimeout(10*time.Second),
		)
		exporter, err := otlptrace.New(context.Background(), client)
		if err != nil {
			return nil, err
		}
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
		)
	}

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("shutting down tracer provider")
			return tp.Shutdown(ctx)
		},
	})

	return tp, nil
}
