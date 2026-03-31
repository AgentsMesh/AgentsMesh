package aggregator

import (
	"testing"
	"time"
)

// Tests for basic aggregation functionality

func TestSmartAggregator_BasicAggregation(t *testing.T) {
	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
		func() float64 { return 0 }, // No queue pressure
		WithSmartBaseDelay(10*time.Millisecond),
	)
	agg.SetRelayClient(relay)

	// Write some data
	agg.Write([]byte("hello"))
	agg.Write([]byte(" world"))

	// Wait for flush
	time.Sleep(100 * time.Millisecond)

	if string(relay.getData()) != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", string(relay.getData()))
	}
}

func TestSmartAggregator_AdaptiveDelay(t *testing.T) {
	agg := NewSmartAggregator(
		func() float64 { return 0 },
		WithSmartBaseDelay(16*time.Millisecond),
		WithSmartMaxDelay(200*time.Millisecond),
	)
	defer agg.Stop()

	tests := []struct {
		usage    float64
		expected time.Duration
	}{
		{0.0, 16 * time.Millisecond},  // No load: base delay
		{0.5, 64 * time.Millisecond},  // 50% load: 16 * (1 + 0.25*12) = 16 * 4 = 64
		{0.8, 124 * time.Millisecond}, // 80% load: 16 * (1 + 0.64*12) = 16 * 8.68 ≈ 139
		{1.0, 200 * time.Millisecond}, // 100% load: capped at maxDelay
	}

	for _, tc := range tests {
		delay := agg.calculateDelay(tc.usage)
		// Allow 20% tolerance for rounding
		minExpected := time.Duration(float64(tc.expected) * 0.8)
		maxExpected := time.Duration(float64(tc.expected) * 1.2)
		if tc.usage == 1.0 {
			// For max load, should be exactly maxDelay
			if delay != tc.expected {
				t.Errorf("usage=%.1f: expected %v, got %v", tc.usage, tc.expected, delay)
			}
		} else if delay < minExpected || delay > maxExpected {
			t.Errorf("usage=%.1f: expected ~%v, got %v", tc.usage, tc.expected, delay)
		}
	}
}

func TestSmartAggregator_NilQueueUsageFn(t *testing.T) {
	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
		nil,
		WithSmartBaseDelay(10*time.Millisecond),
	)
	agg.SetRelayClient(relay)

	agg.Write([]byte("test"))

	time.Sleep(100 * time.Millisecond)

	if len(relay.getData()) == 0 {
		t.Fatal("Expected data to be flushed")
	}

	agg.Stop()
}

func TestSmartAggregator_Flush(t *testing.T) {
	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
		func() float64 { return 0 },
		WithSmartBaseDelay(1*time.Second),
	)
	agg.SetRelayClient(relay)
	defer agg.Stop()

	agg.Write([]byte("data"))
	agg.Flush()

	time.Sleep(50 * time.Millisecond)

	if string(relay.getData()) != "data" {
		t.Errorf("Expected 'data', got '%s'", string(relay.getData()))
	}
}

func TestSmartAggregator_IsStopped(t *testing.T) {
	agg := NewSmartAggregator(
		func() float64 { return 0 },
	)

	if agg.IsStopped() {
		t.Error("Should not be stopped initially")
	}

	agg.Stop()

	if !agg.IsStopped() {
		t.Error("Should be stopped after Stop()")
	}

	// Double stop should not panic
	agg.Stop()
}
