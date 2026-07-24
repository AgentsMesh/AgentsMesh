package runner

import (
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
)

const (
	snapshotRepairLogInterval = 6
	snapshotRepairMaxBackoff  = 3200 * time.Millisecond
)

func (r *PTYPodRelay) outputHealthHandler() aggregator.OutputHealthHandler {
	return func(health aggregator.OutputHealth) {
		state, epoch := r.repairState(health.Destination)
		if state != nil {
			r.scheduleOutputRepair(state, health.Destination, epoch)
		}
	}
}

func (r *PTYPodRelay) repairState(destination string) (*outputRepairState, uint64) {
	switch destination {
	case aggregator.OutputDestinationCloud:
		return &r.cloudRepair, r.cloudEpoch.Load()
	case aggregator.OutputDestinationLocal:
		return &r.localRepair, 1
	default:
		return nil, 0
	}
}

func (r *PTYPodRelay) scheduleOutputRepair(state *outputRepairState, destination string, epoch uint64) {
	requestRepairEpoch(state, epoch)
	if !state.active.CompareAndSwap(false, true) {
		return
	}
	safego.Go("pty-output-repair-"+destination, func() {
		r.runOutputRepair(state, destination)
	})
}

func requestRepairEpoch(state *outputRepairState, epoch uint64) {
	for current := state.requestedEpoch.Load(); epoch > current; current = state.requestedEpoch.Load() {
		if state.requestedEpoch.CompareAndSwap(current, epoch) {
			return
		}
	}
}

func (r *PTYPodRelay) runOutputRepair(state *outputRepairState, destination string) {
	for {
		epoch := state.requestedEpoch.Load()
		r.repairOutputUntilResolved(destination, epoch)
		if state.requestedEpoch.Load() != epoch {
			continue
		}

		state.active.Store(false)
		if state.requestedEpoch.Load() != epoch {
			if state.active.CompareAndSwap(false, true) {
				continue
			}
			return
		}
		if r.outputDestinationDesynced(destination) && state.active.CompareAndSwap(false, true) {
			continue
		}
		return
	}
}

func (r *PTYPodRelay) repairOutputUntilResolved(destination string, epoch uint64) {
	for failures := 0; ; failures++ {
		if r.destinationEpoch(destination) != epoch || !r.outputDestinationDesynced(destination) {
			return
		}
		if r.repairOutputDestination(destination) {
			return
		}
		failureCount := failures + 1
		if failureCount%snapshotRepairLogInterval == 0 {
			logger.Pod().Error("PTY output destination is still desynchronized",
				"pod_key", r.podKey, "destination", destination,
				"error", fmt.Sprintf("snapshot delivery failed %d times; retrying", failureCount))
		}
		time.Sleep(snapshotRepairBackoff(failureCount))
	}
}

func snapshotRepairBackoff(failureCount int) time.Duration {
	if failureCount < 1 {
		failureCount = 1
	}
	delay := time.Duration(1<<min(failureCount-1, 5)) * 100 * time.Millisecond
	return min(delay, snapshotRepairMaxBackoff)
}

func (r *PTYPodRelay) destinationEpoch(destination string) uint64 {
	if destination == aggregator.OutputDestinationCloud {
		return r.cloudEpoch.Load()
	}
	return 1
}

func (r *PTYPodRelay) repairOutputDestination(destination string) bool {
	switch destination {
	case aggregator.OutputDestinationCloud:
		return r.deliverCloudSnapshot(true, nil)
	case aggregator.OutputDestinationLocal:
		return r.deliverSnapshot(destination, true, snapshotDestination, func(data []byte) error {
			if r.localServer == nil || !r.localServer.IsPodConnected(r.podKey) {
				return errOutputUnavailable
			}
			return r.localServer.Send(r.podKey, relay.MsgTypeSnapshot, data)
		})
	default:
		return true
	}
}

func (r *PTYPodRelay) outputDestinationDesynced(destination string) bool {
	if r.components == nil || r.components.Aggregator == nil {
		return false
	}
	health, found := findOutputDestination(r.components.Aggregator.OutputHealth(), destination)
	return found && health.Desynced
}
