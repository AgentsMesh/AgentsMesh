// Package terminal provides terminal management for PTY sessions.
package aggregator

import "time"

// SmartAggregatorOption is a functional option for SmartAggregator.
type SmartAggregatorOption func(*SmartAggregator)

// WithSmartBaseDelay sets the base delay for aggregation.
func WithSmartBaseDelay(d time.Duration) SmartAggregatorOption {
	return func(a *SmartAggregator) {
		a.delay.SetBaseDelay(d)
	}
}

// WithSmartMaxDelay sets the maximum delay for aggregation.
func WithSmartMaxDelay(d time.Duration) SmartAggregatorOption {
	return func(a *SmartAggregator) {
		a.delay.SetMaxDelay(d)
	}
}

// WithSmartMaxSize sets the bulk buffer cap. UTF-8 boundary headroom is fixed.
func WithSmartMaxSize(size int) SmartAggregatorOption {
	return func(a *SmartAggregator) {
		a.buffer.SetMaxSize(size)
	}
}

// WithBackpressureCallbacks opts into callbacks for explicit Pause/Resume calls.
// Destination queue pressure and desynchronization never invoke them automatically.
func WithBackpressureCallbacks(onPause, onResume func()) SmartAggregatorOption {
	return func(a *SmartAggregator) {
		a.backpressure.SetCallbacks(onPause, onResume)
	}
}

// WithSerializeCallback sets a callback that returns serialized terminal content.
// When set, flushLocked() calls this callback to get compressed data instead of using raw buffer.
// This enables bandwidth optimization by using VirtualTerminal.Serialize() which compresses
// spaces to CSI CUF sequences.
//
// In serialize mode:
// - Write() only marks "has pending data", doesn't buffer the actual data
// - flushLocked() calls serializeCallback to get compressed output
// - The callback should return VT.Serialize() result
func WithSerializeCallback(fn func() []byte) SmartAggregatorOption {
	return func(a *SmartAggregator) {
		a.serializeCallback = fn
	}
}

// WithFullRedrawThrottling enables throttling for high-frequency full-screen redraws.
// When applications produce rapid full-screen refreshes (like `glab ci status --live`),
// this option detects the pattern and reduces transmission frequency to save bandwidth.
//
// This only applies to Legacy mode (non-Serialize mode). In Serialize mode, the
// VirtualTerminal already handles optimization.
//
// Default parameters:
//   - Window size: 2 seconds
//   - Threshold: 2.5 redraws/second (triggers throttling)
//   - Min delay: 200ms (at threshold)
//   - Max delay: 1000ms (at 10+ redraws/second)
//
// Example bandwidth savings:
//   - 10 redraws/sec → ~80% reduction (only ~2 flushes/sec)
//   - 20 redraws/sec → ~95% reduction (only ~1 flush/sec)
func WithFullRedrawThrottling(opts ...FullRedrawThrottlerOption) SmartAggregatorOption {
	return func(a *SmartAggregator) {
		a.fullRedrawThrottler = NewFullRedrawThrottler(opts...)
	}
}
