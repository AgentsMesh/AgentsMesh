// Package terminal provides terminal management for PTY sessions.
package terminal

import "sync"

// BackpressureController manages backpressure state for flow control.
// When paused, producers should stop or slow down data production.
//
// Implements ttyd-style flow control where backpressure propagates
// from consumer (network/gRPC) all the way to producer (PTY).
type BackpressureController struct {
	mu       sync.RWMutex
	paused   bool
	resumeCh chan struct{}

	// Callbacks for propagating backpressure to upstream (e.g., Terminal.PauseRead)
	onPause  func()
	onResume func()
}

// NewBackpressureController creates a new backpressure controller.
//
// Parameters:
// - onPause: called when transitioning to paused state (optional)
// - onResume: called when transitioning from paused to resumed state (optional)
func NewBackpressureController(onPause, onResume func()) *BackpressureController {
	return &BackpressureController{
		resumeCh: make(chan struct{}, 1),
		onPause:  onPause,
		onResume: onResume,
	}
}

// Pause signals that consumer is overloaded.
// If transitioning from unpaused to paused, calls onPause callback.
func (c *BackpressureController) Pause() {
	c.mu.Lock()
	wasPaused := c.paused
	c.paused = true
	c.mu.Unlock()

	// Call callback only on transition (not when already paused)
	if !wasPaused && c.onPause != nil {
		c.onPause()
	}
}

// Resume signals that consumer is ready for more data.
// If transitioning from paused to unpaused, calls onResume callback and signals resumeCh.
// Returns true if was actually paused before.
func (c *BackpressureController) Resume() bool {
	c.mu.Lock()
	wasPaused := c.paused
	c.paused = false
	c.mu.Unlock()

	if wasPaused {
		// Call callback only on transition
		if c.onResume != nil {
			c.onResume()
		}

		// Signal resume channel (non-blocking)
		select {
		case c.resumeCh <- struct{}{}:
		default:
		}
	}

	return wasPaused
}

// IsPaused returns whether currently paused.
func (c *BackpressureController) IsPaused() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.paused
}

// ResumeCh returns the resume signal channel.
// Readers can select on this to be notified when resume happens.
func (c *BackpressureController) ResumeCh() <-chan struct{} {
	return c.resumeCh
}

// SetCallbacks updates the pause/resume callbacks.
func (c *BackpressureController) SetCallbacks(onPause, onResume func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onPause = onPause
	c.onResume = onResume
}
