package otel

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestOperationalMetricsRegistration(t *testing.T) {
	ctx := context.Background()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
		initMetrics(noop.NewMeterProvider().Meter(meterName))
	})

	initMetrics(provider.Meter(meterName))
	RecordHeartbeatProcessDuration(ctx, 500*time.Microsecond)
	RecordPodDispatchAttemptDuration(ctx, 1500*time.Microsecond, nil)
	RecordPodDispatchAttemptDuration(ctx, 2250*time.Microsecond, errors.New("runner unavailable"))
	RecordGRPCMessageHandleDuration(ctx, 2750*time.Microsecond, "PodCreated")

	var collected metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &collected); err != nil {
		t.Fatalf("collect metrics: %v", err)
	}

	assertHistogram(t, collected,
		"agentsmesh.backend.heartbeat.process.duration",
		"Time to synchronously process a runner heartbeat message",
		"", map[string]float64{"": 0.5})
	assertHistogram(t, collected,
		"agentsmesh.backend.pod.dispatch.attempt.duration",
		"Time to reserve runner capacity and attempt local create_pod enqueue, including failure rollback",
		"outcome", map[string]float64{"success": 1.5, "error": 2.25})
	assertHistogram(t, collected,
		"agentsmesh.backend.grpc.message.handle.duration",
		"Time to synchronously process a non-high-frequency runner gRPC message by type",
		"message.type", map[string]float64{"PodCreated": 2.75})
}

func TestOperationalMetricsDefaultNoop(t *testing.T) {
	ctx := context.Background()
	initMetrics(noop.NewMeterProvider().Meter(meterName))

	if HeartbeatProcessDuration.Enabled(ctx) || PodDispatchAttemptDuration.Enabled(ctx) || GRPCMessageHandleDuration.Enabled(ctx) {
		t.Fatal("noop operational metrics must be disabled")
	}

	RecordHeartbeatProcessDuration(ctx, time.Millisecond)
	RecordPodDispatchAttemptDuration(ctx, time.Millisecond, errors.New("expected"))
	RecordGRPCMessageHandleDuration(ctx, time.Millisecond, "PodCreated")
}

func assertHistogram(
	t *testing.T,
	collected metricdata.ResourceMetrics,
	name string,
	description string,
	attributeKey string,
	wantSums map[string]float64,
) {
	t.Helper()

	metricData := findMetric(t, collected, name)
	if metricData.Unit != "ms" {
		t.Errorf("%s unit = %q, want ms", name, metricData.Unit)
	}
	if metricData.Description != description {
		t.Errorf("%s description = %q, want %q", name, metricData.Description, description)
	}

	histogram, ok := metricData.Data.(metricdata.Histogram[float64])
	if !ok {
		t.Fatalf("%s data type = %T, want float64 histogram", name, metricData.Data)
	}
	if len(histogram.DataPoints) != len(wantSums) {
		t.Fatalf("%s data point count = %d, want %d", name, len(histogram.DataPoints), len(wantSums))
	}

	seen := make(map[string]bool, len(wantSums))
	for _, point := range histogram.DataPoints {
		attributeValue := ""
		if attributeKey == "" {
			if point.Attributes.Len() != 0 {
				t.Fatalf("%s attributes = %v, want none", name, point.Attributes.ToSlice())
			}
		} else {
			value, found := point.Attributes.Value(attribute.Key(attributeKey))
			if !found {
				t.Fatalf("%s missing attribute %q", name, attributeKey)
			}
			attributeValue = value.AsString()
		}

		wantSum, found := wantSums[attributeValue]
		if !found {
			t.Fatalf("%s unexpected %s=%q", name, attributeKey, attributeValue)
		}
		if point.Count != 1 {
			t.Errorf("%s[%s=%q] count = %d, want 1", name, attributeKey, attributeValue, point.Count)
		}
		if math.Abs(point.Sum-wantSum) > 1e-9 {
			t.Errorf("%s[%s=%q] sum = %v, want %v", name, attributeKey, attributeValue, point.Sum, wantSum)
		}
		seen[attributeValue] = true
	}

	for attributeValue := range wantSums {
		if !seen[attributeValue] {
			t.Errorf("%s missing %s=%q", name, attributeKey, attributeValue)
		}
	}
}

func findMetric(t *testing.T, collected metricdata.ResourceMetrics, name string) metricdata.Metrics {
	t.Helper()
	for _, scopeMetrics := range collected.ScopeMetrics {
		for _, metricData := range scopeMetrics.Metrics {
			if metricData.Name == name {
				return metricData
			}
		}
	}
	t.Fatalf("metric %q not collected", name)
	return metricdata.Metrics{}
}
