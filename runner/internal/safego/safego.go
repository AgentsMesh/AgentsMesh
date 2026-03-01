// Package safego provides panic-safe goroutine launching utilities.
// All goroutines started through this package are wrapped with panic recovery,
// preventing a single goroutine panic from crashing the entire process.
package safego

import (
	"fmt"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/AgentsMesh/AgentsMesh/runner/internal/logger"
)

// restartDelay is the delay between panic-triggered restarts in GoLoop.
const restartDelay = 1 * time.Second

// panicCount tracks the total number of recovered panics (for monitoring/testing).
var panicCount atomic.Int64

// PanicCount returns the total number of panics recovered by safego.
func PanicCount() int64 {
	return panicCount.Load()
}

// ResetPanicCount resets the panic counter (for testing).
func ResetPanicCount() {
	panicCount.Store(0)
}

// Go starts a goroutine with panic recovery.
// If fn panics, the panic is recovered, logged with the goroutine name and full stack trace,
// and the goroutine exits without crashing the process.
func Go(name string, fn func()) {
	go run(name, fn)
}

// GoLoop starts a goroutine that automatically restarts on panic.
// If fn panics, it is recovered, logged, and fn is restarted after a 1-second delay.
// After maxRestarts consecutive panics, the goroutine stops restarting.
// If maxRestarts <= 0, the goroutine restarts indefinitely.
// If fn returns normally (without panic), the goroutine exits.
func GoLoop(name string, fn func(), maxRestarts int) {
	go func() {
		restarts := 0
		for {
			panicked := runWithPanicFlag(name, fn)
			if !panicked {
				// Normal return — exit the loop
				return
			}

			restarts++
			if maxRestarts > 0 && restarts >= maxRestarts {
				logger.Runner().Error("Goroutine exceeded max restarts, giving up",
					"goroutine", name,
					"restarts", restarts,
					"max_restarts", maxRestarts,
				)
				return
			}

			// Delay before restart to avoid tight restart loops
			time.Sleep(restartDelay)
		}
	}()
}

// run executes fn with panic recovery. Used by Go().
func run(name string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			panicCount.Add(1)
			log := logger.Runner()
			log.Error("Goroutine panic recovered",
				"goroutine", name,
				"panic", fmt.Sprintf("%v", r),
				"stack", string(debug.Stack()),
			)
		}
	}()
	fn()
}

// runWithPanicFlag executes fn and returns true if fn panicked, false otherwise.
// Used by GoLoop() to distinguish between normal return and panic.
func runWithPanicFlag(name string, fn func()) (panicked bool) {
	panicked = true
	defer func() {
		if r := recover(); r != nil {
			panicCount.Add(1)
			log := logger.Runner()
			log.Error("Goroutine panic recovered (will restart)",
				"goroutine", name,
				"panic", fmt.Sprintf("%v", r),
				"stack", string(debug.Stack()),
			)
			// panicked remains true
		}
	}()
	fn()
	panicked = false
	return
}
