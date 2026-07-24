package runner

import (
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
)

const snapshotDeliveryTimeout = 2 * time.Second

var (
	errSnapshotUnavailable       = errors.New("terminal snapshot unavailable")
	errOutputUnavailable         = errors.New("output destination unavailable")
	errRequesterCannotRecoverAll = errors.New("requester snapshot cannot recover destination")
)

type snapshotDelivery func([]byte) error

type snapshotAudience uint8

const (
	snapshotRequester snapshotAudience = iota
	snapshotDestination
)

func (r *PTYPodRelay) SendSnapshot(rc relay.RelayClient) {
	r.deliverCloudSnapshot(false, rc)
}

func (r *PTYPodRelay) RecoverSnapshot(rc relay.RelayClient) {
	r.deliverCloudSnapshot(true, rc)
}

func (r *PTYPodRelay) deliverCloudSnapshot(resetAll bool, expected relay.RelayClient) bool {
	if r.components == nil {
		return false
	}
	r.components.streamMu.Lock()
	defer r.components.streamMu.Unlock()
	return r.deliverCloudSnapshotLocked(resetAll, expected)
}

func (r *PTYPodRelay) deliverCloudSnapshotLocked(resetAll bool, expected relay.RelayClient) bool {
	return r.deliverSnapshotLocked(
		aggregator.OutputDestinationCloud,
		resetAll,
		snapshotDestination,
		func(data []byte) error {
			client := r.currentCloudClient()
			if client == nil && r.components.Aggregator == nil {
				client = expected
			}
			if client == nil || (expected != nil && client != expected) || !client.IsConnected() {
				return errOutputUnavailable
			}
			return client.Send(relay.MsgTypeSnapshot, data)
		},
	)
}

func (r *PTYPodRelay) sendSnapshotToLocal(reply relay.ReplyFunc) {
	if reply == nil || r.components == nil {
		return
	}
	r.deliverSnapshot(aggregator.OutputDestinationLocal, false, snapshotRequester, func(data []byte) error {
		reply(relay.MsgTypeSnapshot, data)
		return nil
	})
}

func (r *PTYPodRelay) deliverRequesterSnapshotLocked(deliver snapshotDelivery) bool {
	data := r.materializeSnapshotEnvelope(false)
	if data == nil {
		return false
	}
	if err := deliver(data); err != nil {
		logger.Pod().Warn("Failed to deliver PTY requester snapshot",
			"pod_key", r.podKey, "error", err)
		return false
	}
	return true
}

func (r *PTYPodRelay) deliverSnapshot(
	destination string,
	resetAll bool,
	audience snapshotAudience,
	deliver snapshotDelivery,
) bool {
	if deliver == nil || r.components == nil {
		return false
	}

	r.components.streamMu.Lock()
	defer r.components.streamMu.Unlock()
	return r.deliverSnapshotLocked(destination, resetAll, audience, deliver)
}

func (r *PTYPodRelay) deliverSnapshotLocked(
	destination string,
	resetAll bool,
	audience snapshotAudience,
	deliver snapshotDelivery,
) bool {
	var boundary *aggregator.OutputSnapshotBoundary
	if agg := r.components.Aggregator; agg != nil {
		if health, found := findOutputDestination(agg.OutputHealth(), destination); found {
			var ok bool
			boundary, ok = agg.PrepareSnapshot(destination, snapshotDeliveryTimeout)
			if !ok {
				logger.Pod().Warn("PTY snapshot boundary unavailable",
					"pod_key", r.podKey, "destination", destination)
				return false
			}
			if boundary.RecoveryRequired() {
				if audience == snapshotRequester {
					boundary.Abort(errRequesterCannotRecoverAll)
					return r.deliverRequesterSnapshotLocked(deliver)
				}
				resetAll = true
			} else {
				resetAll = resetAll || health.Desynced
			}
		} else if !agg.FlushAndWait(snapshotDeliveryTimeout) {
			logger.Pod().Warn("PTY snapshot output drain failed",
				"pod_key", r.podKey, "destination", destination)
			return false
		}
	}

	data := r.materializeSnapshotEnvelope(resetAll)
	if data == nil {
		if boundary != nil {
			boundary.Abort(errSnapshotUnavailable)
		}
		return false
	}
	if err := deliver(data); err != nil {
		if boundary != nil {
			boundary.Abort(err)
		}
		logger.Pod().Warn("Failed to deliver PTY snapshot",
			"pod_key", r.podKey, "destination", destination, "error", err)
		return false
	}
	if boundary != nil && !boundary.Commit() {
		logger.Pod().Warn("Failed to commit PTY snapshot boundary",
			"pod_key", r.podKey, "destination", destination)
		return false
	}
	return true
}

func findOutputDestination(health []aggregator.OutputHealth, destination string) (aggregator.OutputHealth, bool) {
	for _, state := range health {
		if state.Destination == destination {
			return state, true
		}
	}
	return aggregator.OutputHealth{}, false
}
