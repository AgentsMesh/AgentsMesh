package terminal

import (
	"testing"
	"time"
)

func TestAdaptiveDelay_NewAdaptiveDelay(t *testing.T) {
	usage := 0.5
	ad := NewAdaptiveDelay(
		50*time.Millisecond,
		500*time.Millisecond,
		func() float64 { return usage },
	)

	if ad.BaseDelay() != 50*time.Millisecond {
		t.Errorf("Expected base delay 50ms, got %v", ad.BaseDelay())
	}
	if ad.MaxDelay() != 500*time.Millisecond {
		t.Errorf("Expected max delay 500ms, got %v", ad.MaxDelay())
	}
}

func TestAdaptiveDelay_Calculate(t *testing.T) {
	usage := 0.0
	ad := NewAdaptiveDelay(
		50*time.Millisecond,
		500*time.Millisecond,
		func() float64 { return usage },
	)

	// 0% usage
	delay := ad.Calculate()
	if delay != 50*time.Millisecond {
		t.Errorf("At 0%% usage, expected 50ms, got %v", delay)
	}

	// 50% usage
	usage = 0.5
	delay = ad.Calculate()
	expected := time.Duration(float64(50*time.Millisecond) * (1.0 + 0.25*12)) // 50ms * 4 = 200ms
	if delay != expected {
		t.Errorf("At 50%% usage, expected %v, got %v", expected, delay)
	}

	// 100% usage - should cap at maxDelay
	usage = 1.0
	delay = ad.Calculate()
	if delay != 500*time.Millisecond {
		t.Errorf("At 100%% usage, expected max delay 500ms, got %v", delay)
	}
}

func TestAdaptiveDelay_CalculateForUsage(t *testing.T) {
	ad := NewAdaptiveDelay(
		16*time.Millisecond,
		200*time.Millisecond,
		nil,
	)

	tests := []struct {
		usage    float64
		expected time.Duration
	}{
		{0.0, 16 * time.Millisecond},
		{0.5, 64 * time.Millisecond},   // 16 * (1 + 0.25*12) = 16 * 4 = 64
		{1.0, 200 * time.Millisecond},  // capped at max
	}

	for _, tc := range tests {
		delay := ad.CalculateForUsage(tc.usage)
		if delay != tc.expected {
			t.Errorf("usage=%.1f: expected %v, got %v", tc.usage, tc.expected, delay)
		}
	}
}

func TestAdaptiveDelay_GetUsage_NilFn(t *testing.T) {
	ad := NewAdaptiveDelay(50*time.Millisecond, 500*time.Millisecond, nil)

	usage := ad.GetUsage()
	if usage != 0.0 {
		t.Errorf("Expected 0.0 with nil fn, got %f", usage)
	}
}

func TestAdaptiveDelay_IsCriticalLoad(t *testing.T) {
	usage := 0.0
	ad := NewAdaptiveDelay(50*time.Millisecond, 500*time.Millisecond,
		func() float64 { return usage })

	if ad.IsCriticalLoad() {
		t.Error("0% should not be critical load")
	}

	usage = 0.5
	if ad.IsCriticalLoad() {
		t.Error("50% should not be critical load")
	}

	usage = 0.51
	if !ad.IsCriticalLoad() {
		t.Error("51% should be critical load")
	}
}

func TestAdaptiveDelay_IsHighLoad(t *testing.T) {
	usage := 0.0
	ad := NewAdaptiveDelay(50*time.Millisecond, 500*time.Millisecond,
		func() float64 { return usage })

	if ad.IsHighLoad() {
		t.Error("0% should not be high load")
	}

	usage = 0.8
	if ad.IsHighLoad() {
		t.Error("80% should not be high load")
	}

	usage = 0.81
	if !ad.IsHighLoad() {
		t.Error("81% should be high load")
	}
}

func TestAdaptiveDelay_SetBaseDelay(t *testing.T) {
	ad := NewAdaptiveDelay(50*time.Millisecond, 500*time.Millisecond, nil)

	ad.SetBaseDelay(100 * time.Millisecond)
	if ad.BaseDelay() != 100*time.Millisecond {
		t.Errorf("Expected 100ms after SetBaseDelay, got %v", ad.BaseDelay())
	}
}

func TestAdaptiveDelay_SetMaxDelay(t *testing.T) {
	ad := NewAdaptiveDelay(50*time.Millisecond, 500*time.Millisecond, nil)

	ad.SetMaxDelay(1000 * time.Millisecond)
	if ad.MaxDelay() != 1000*time.Millisecond {
		t.Errorf("Expected 1000ms after SetMaxDelay, got %v", ad.MaxDelay())
	}
}
