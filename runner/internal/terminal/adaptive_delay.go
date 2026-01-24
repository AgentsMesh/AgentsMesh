// Package terminal provides terminal management for PTY sessions.
package terminal

import "time"

// AdaptiveDelay calculates adaptive delay based on queue pressure.
// Higher queue usage results in longer delays to prevent overwhelming consumers.
type AdaptiveDelay struct {
	baseDelay    time.Duration
	maxDelay     time.Duration
	queueUsageFn func() float64
}

// NewAdaptiveDelay creates a new adaptive delay calculator.
//
// Parameters:
// - baseDelay: minimum delay (used at 0% load)
// - maxDelay: maximum delay cap
// - queueUsageFn: returns queue usage ratio (0.0 to 1.0)
func NewAdaptiveDelay(baseDelay, maxDelay time.Duration, queueUsageFn func() float64) *AdaptiveDelay {
	return &AdaptiveDelay{
		baseDelay:    baseDelay,
		maxDelay:     maxDelay,
		queueUsageFn: queueUsageFn,
	}
}

// Calculate computes the current delay based on queue usage.
//
// Load mapping (with quadratic scaling):
// - 0%   → baseDelay  (e.g., 16ms = 60 FPS)
// - 50%  → ~4x base   (e.g., 64ms = 15 FPS)
// - 80%  → ~8.7x base (e.g., 139ms ≈ 7 FPS)
// - 100% → capped at maxDelay
func (d *AdaptiveDelay) Calculate() time.Duration {
	usage := d.GetUsage()
	return d.CalculateForUsage(usage)
}

// CalculateForUsage computes delay for a given usage value.
func (d *AdaptiveDelay) CalculateForUsage(usage float64) time.Duration {
	// Quadratic scaling: factor = 1 + usage² * 12
	// This gives smooth ramp-up with aggressive throttling at high load
	factor := 1.0 + (usage * usage * 12)
	delay := time.Duration(float64(d.baseDelay) * factor)
	if delay > d.maxDelay {
		return d.maxDelay
	}
	return delay
}

// GetUsage returns the current queue usage ratio.
func (d *AdaptiveDelay) GetUsage() float64 {
	if d.queueUsageFn == nil {
		return 0.0
	}
	return d.queueUsageFn()
}

// IsCriticalLoad returns true if queue usage exceeds 50%.
// At critical load, flushes should be skipped to allow more aggregation.
func (d *AdaptiveDelay) IsCriticalLoad() bool {
	return d.GetUsage() > 0.5
}

// IsHighLoad returns true if queue usage exceeds 80%.
// At high load, only forced flushes (like Stop) should proceed.
func (d *AdaptiveDelay) IsHighLoad() bool {
	return d.GetUsage() > 0.8
}

// BaseDelay returns the configured base delay.
func (d *AdaptiveDelay) BaseDelay() time.Duration {
	return d.baseDelay
}

// MaxDelay returns the configured max delay.
func (d *AdaptiveDelay) MaxDelay() time.Duration {
	return d.maxDelay
}

// SetBaseDelay updates the base delay.
func (d *AdaptiveDelay) SetBaseDelay(delay time.Duration) {
	d.baseDelay = delay
}

// SetMaxDelay updates the max delay.
func (d *AdaptiveDelay) SetMaxDelay(delay time.Duration) {
	d.maxDelay = delay
}
