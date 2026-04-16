package otel

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type Provider struct {
	tp      *sdktrace.TracerProvider
	mp      *sdkmetric.MeterProvider
	enabled bool
}

func InitProvider(ctx context.Context, serviceName, version string) (*Provider, error) {
	if os.Getenv("OTEL_SDK_DISABLED") == "true" {
		slog.Info("OpenTelemetry disabled via OTEL_SDK_DISABLED")
		return &Provider{}, nil
	}
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		slog.Info("OpenTelemetry disabled: OTEL_EXPORTER_OTLP_ENDPOINT not set")
		return &Provider{}, nil
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
		),
	)
	if err != nil {
		return nil, err
	}

	tp, err := initTracerProvider(ctx, res)
	if err != nil {
		return nil, err
	}

	mp, err := initMeterProvider(ctx, res)
	if err != nil {
		_ = tp.Shutdown(ctx)
		return nil, err
	}

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	slog.Info("OpenTelemetry initialized", "service", serviceName)
	InitMetrics()
	return &Provider{tp: tp, mp: mp, enabled: true}, nil
}

func (p *Provider) Shutdown(ctx context.Context) {
	if p.tp != nil {
		if err := p.tp.Shutdown(ctx); err != nil {
			slog.Warn("Failed to shutdown TracerProvider", "error", err)
		}
	}
	if p.mp != nil {
		if err := p.mp.Shutdown(ctx); err != nil {
			slog.Warn("Failed to shutdown MeterProvider", "error", err)
		}
	}
}

func initTracerProvider(ctx context.Context, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, err
	}
	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(time.Second)),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(buildSampler()),
	), nil
}

const defaultSamplingRatio = 0.01

func buildSampler() sdktrace.Sampler {
	ratio := os.Getenv("OTEL_TRACES_SAMPLER_ARG")
	if ratio == "" {
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(defaultSamplingRatio))
	}
	r, err := strconv.ParseFloat(ratio, 64)
	if err != nil {
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(defaultSamplingRatio))
	}
	return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(r))
}

func initMeterProvider(ctx context.Context, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	exporter, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		return nil, err
	}
	return sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(10*time.Second))),
		sdkmetric.WithResource(res),
	), nil
}
