// Package client provides gRPC connection management for Runner.
package client

import (
	"sync/atomic"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// HeartbeatMonitor tracks the heartbeat send/ack cycle for upstream liveness detection.
//
// Each sent heartbeat increments missedAcks; each received HeartbeatAck resets it to 0.
// When missedAcks reaches maxMissedAcks (default 3), the onUnhealthy callback fires to
// trigger reconnection. This detects half-dead connections where the upstream path
// (Runner → Backend) is broken but the downstream (Backend → Runner) appears alive.
//
// Thread-safe: OnSent() is called from heartbeatLoop, OnAck() from readLoop.
type HeartbeatMonitor struct {
	missedAcks    atomic.Int32
	maxMissedAcks int32
	onUnhealthy   func() // called when consecutive unacked heartbeats exceed threshold
}

// NewHeartbeatMonitor creates a monitor that fires onUnhealthy after maxMissed
// consecutive heartbeats go unacknowledged.
func NewHeartbeatMonitor(maxMissed int32, onUnhealthy func()) *HeartbeatMonitor {
	return &HeartbeatMonitor{
		maxMissedAcks: maxMissed,
		onUnhealthy:   onUnhealthy,
	}
}

// OnSent is called after a heartbeat is successfully enqueued for sending.
// If the missed ack count reaches the threshold, triggers reconnection.
func (m *HeartbeatMonitor) OnSent() {
	missed := m.missedAcks.Add(1)
	if missed >= m.maxMissedAcks {
		logger.GRPC().Error("Heartbeat monitor: upstream unresponsive, triggering reconnect",
			"missed_acks", missed, "threshold", m.maxMissedAcks)
		if m.onUnhealthy != nil {
			m.onUnhealthy()
		}
	} else if missed > 1 {
		logger.GRPC().Warn("Heartbeat ack delayed",
			"missed_acks", missed, "threshold", m.maxMissedAcks)
	}
}

// OnAck is called when a HeartbeatAck is received from the backend.
// Resets the missed counter, confirming the upstream path is alive.
func (m *HeartbeatMonitor) OnAck() {
	m.missedAcks.Store(0)
}

// MissedCount returns the current number of unacknowledged heartbeats.
func (m *HeartbeatMonitor) MissedCount() int32 {
	return m.missedAcks.Load()
}
