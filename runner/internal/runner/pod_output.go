package runner

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// NewPTYOutputHandler creates the standard output handler closure that wires
// VirtualTerminal, StateDetector, and Aggregator together.
//
// This is the single source of truth for the PTY output pipeline, used by
// both OnCreatePod (via PodBuilder.Build) and session recovery.
//
// notifyDetector is typically Pod.NotifyStateDetectorWithScreen.
func NewPTYOutputHandler(
	podKey string,
	comps *PTYComponents,
	notifyDetector func(bytes int, screenLines []string),
) func([]byte) {
	virtualTerm := comps.VirtualTerminal
	agg := comps.Aggregator

	var panicked atomic.Bool

	return func(data []byte) {
		// Circuit breaker: skip processing after first panic to avoid
		// repeated panic-recover cycles on every output chunk.
		if panicked.Load() {
			return
		}

		defer func() {
			if r := recover(); r != nil {
				panicked.Store(true)
				logger.Terminal().Error("PANIC in OutputHandler — circuit breaker activated",
					"pod_key", podKey, "panic", fmt.Sprintf("%v", r))
			}
		}()

		var screenLines []string
		if virtualTerm != nil {
			startFeed := time.Now()
			screenLines = virtualTerm.Feed(data)
			if feedTime := time.Since(startFeed); feedTime > 100*time.Millisecond {
				logger.Terminal().Warn("VT Feed slow",
					"pod_key", podKey, "data_len", len(data), "feed_time", feedTime)
			}
		}

		// Notify state detector synchronously — lightweight counter/screen updates.
		// Separate recover isolates detector panics from VT+Aggregator pipeline.
		func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Terminal().Error("PANIC in state detector notification",
						"pod_key", podKey, "panic", fmt.Sprintf("%v", r))
				}
			}()
			if notifyDetector != nil {
				notifyDetector(len(data), screenLines)
			}
		}()

		if agg != nil {
			startWrite := time.Now()
			agg.Write(data)
			if writeTime := time.Since(startWrite); writeTime > 100*time.Millisecond {
				logger.Terminal().Warn("Aggregator Write slow",
					"pod_key", podKey, "data_len", len(data), "write_time", writeTime)
			}
		}
	}
}
