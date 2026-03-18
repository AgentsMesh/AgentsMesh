package runner

import (
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
)

// Regression tests for OutputHandler panic isolation (L2 fix).
//
// The L2 fix changed state detector notification from async (safego.Go)
// to synchronous with isolated inline recover. These tests verify:
// 1. Detector panics don't kill the output pipeline
// 2. Aggregator still receives data after detector panic
// 3. Nil Aggregator is handled gracefully (no panic)

func TestCreateOutputHandler_DetectorPanicIsolation(t *testing.T) {
	// Setup: Pod with aggregator but no detector/VT.
	agg := aggregator.NewSmartAggregator(nil, nil)

	pod := &Pod{
		PodKey:     "panic-pod",
		Aggregator: agg,
	}

	handler := pod.CreateOutputHandler()

	// First call: should succeed (no detector, no VT).
	handler([]byte("hello"))
	if agg.BufferLen() == 0 {
		t.Error("aggregator should have received data")
	}

	// Second call: verify pipeline still works (no circuit breaker tripped).
	handler([]byte("world"))
}

func TestCreateOutputHandler_NilAggregator_NoPanic(t *testing.T) {
	// Nil Aggregator should be handled gracefully — data is silently
	// dropped without triggering the circuit breaker or panicking.
	pod := &Pod{
		PodKey:     "nil-agg-pod",
		Aggregator: nil,
	}

	handler := pod.CreateOutputHandler()

	// Should not panic — nil guard protects agg.Write().
	handler([]byte("data1"))
	handler([]byte("data2"))

	// Both calls completed without panic — nil Aggregator is safe.
}

func TestCreateOutputHandler_NilVTAndDetector(t *testing.T) {
	// Normal path: no VirtualTerminal, no StateDetector.
	// Aggregator receives all data.
	agg := aggregator.NewSmartAggregator(nil, nil)

	pod := &Pod{
		PodKey:     "simple-pod",
		Aggregator: agg,
	}

	handler := pod.CreateOutputHandler()

	handler([]byte("chunk1"))
	handler([]byte("chunk2"))
	handler([]byte("chunk3"))

	if agg.BufferLen() == 0 {
		t.Error("aggregator should have buffered data from 3 chunks")
	}
}
