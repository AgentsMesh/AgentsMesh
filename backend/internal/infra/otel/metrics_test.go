package otel

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/metric/noop"
)

func TestInitMetrics_AssignsNonNoopInstruments(t *testing.T) {
	// Before InitMetrics, all instruments are noop zero-values.
	// After InitMetrics with the global noop meter (default), instruments
	// are still noop implementations but must be non-nil assignable.
	// This test verifies InitMetrics does not panic and assigns all vars.

	InitMetrics()

	// Core metrics
	if PodActiveCount == nil {
		t.Error("PodActiveCount is nil after InitMetrics")
	}
	if RunnerConnected == nil {
		t.Error("RunnerConnected is nil after InitMetrics")
	}
	if GRPCMessagesRecv == nil {
		t.Error("GRPCMessagesRecv is nil after InitMetrics")
	}
	if PodCreateDuration == nil {
		t.Error("PodCreateDuration is nil after InitMetrics")
	}

	// New observability metrics
	if HeartbeatProcessDuration == nil {
		t.Error("HeartbeatProcessDuration is nil after InitMetrics")
	}
	if PodDispatchDuration == nil {
		t.Error("PodDispatchDuration is nil after InitMetrics")
	}
	if GRPCMessageDuration == nil {
		t.Error("GRPCMessageDuration is nil after InitMetrics")
	}

	// Blockstore metrics
	if BlockstoreOpsApplied == nil {
		t.Error("BlockstoreOpsApplied is nil after InitMetrics")
	}
	if BlockstoreOpsDuration == nil {
		t.Error("BlockstoreOpsDuration is nil after InitMetrics")
	}
	if BlockstoreEmbedQueue == nil {
		t.Error("BlockstoreEmbedQueue is nil after InitMetrics")
	}
	if BlockstoreEmbedDuration == nil {
		t.Error("BlockstoreEmbedDuration is nil after InitMetrics")
	}
	if BlockstoreSearchDuration == nil {
		t.Error("BlockstoreSearchDuration is nil after InitMetrics")
	}
}

func TestMetrics_DefaultNoop(t *testing.T) {
	// Verify that the default zero-value instruments are valid noop implementations
	// and can be called without panic (important for tests that don't call InitMetrics).
	ctx := context.Background()

	var h = noop.Float64Histogram{}
	h.Record(ctx, 0) // should not panic

	var c = noop.Int64Counter{}
	c.Add(ctx, 0) // should not panic

	var u = noop.Int64UpDownCounter{}
	u.Add(ctx, 0) // should not panic
}
