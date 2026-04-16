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
	tp          *sdktrace.TracerProvider
	mp          *sdkmetric.MeterProvider
	traceFile   *cappedWriter
	metricsFile *cappedWriter
	enabled     bool
}

func InitProvider(ctx context.Context, serviceName, version string) (*Provider, error) {
	if os.Getenv("OTEL_SDK_DISABLED") == "true" {
		slog.Info("OpenTelemetry disabled via OTEL_SDK_DISABLED")
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

	p := &Provider{enabled: true}
	useCollector := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != ""

	if useCollector {
		p.tp, err = initGRPCTracerProvider(ctx, res)
		if err != nil {
			return nil, err
		}
		p.mp, err = initGRPCMeterProvider(ctx, res)
		if err != nil {
			_ = p.tp.Shutdown(ctx)
			return nil, err
		}
		slog.Info("OpenTelemetry initialized (gRPC exporter)", "service", serviceName)
	} else {
		p.tp, p.traceFile, err = initFileTracerProvider(res)
		if err != nil {
			return nil, err
		}
		p.mp, p.metricsFile, err = initFileMeterProvider()
		if err != nil {
			_ = p.tp.Shutdown(ctx)
			p.traceFile.Close()
			return nil, err
		}
		slog.Info("OpenTelemetry initialized (file exporter)", "service", serviceName)
	}

	otel.SetTracerProvider(p.tp)
	otel.SetMeterProvider(p.mp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	InitMetrics()

	return p, nil
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
	if p.traceFile != nil {
		p.traceFile.Close()
	}
	if p.metricsFile != nil {
		p.metricsFile.Close()
	}
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

func initGRPCTracerProvider(ctx context.Context, res *resource.Resource) (*sdktrace.TracerProvider, error) {
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

func initGRPCMeterProvider(ctx context.Context, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	exporter, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		return nil, err
	}
	return sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(10*time.Second))),
		sdkmetric.WithResource(res),
	), nil
}
