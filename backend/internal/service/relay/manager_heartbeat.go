package relay

import (
	"context"
	"fmt"
	"math"
	"time"
)

func (m *Manager) Heartbeat(relayID string, connections int, cpuUsage, memoryUsage float64) error {
	return m.HeartbeatWithLatency(relayID, connections, cpuUsage, memoryUsage, 0)
}

func (m *Manager) HeartbeatWithLatency(relayID string, connections int, cpuUsage, memoryUsage float64, latencyMs int) error {
	if connections < 0 {
		connections = 0
	}
	if math.IsNaN(cpuUsage) || math.IsInf(cpuUsage, 0) || cpuUsage < 0 {
		cpuUsage = 0
	}
	if math.IsNaN(memoryUsage) || math.IsInf(memoryUsage, 0) || memoryUsage < 0 {
		memoryUsage = 0
	}

	// Capture time before acquiring lock to avoid holding mutex during syscall
	now := time.Now()

	m.mu.Lock()

	relay, ok := m.relays[relayID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("relay not found: %s", relayID)
	}

	relay.CurrentConnections = connections
	relay.CPUUsage = cpuUsage
	relay.MemoryUsage = memoryUsage
	relay.LastHeartbeat = now
	relay.Healthy = true

	if latencyMs > 0 {
		if relay.AvgLatencyMs == 0 {
			relay.AvgLatencyMs = latencyMs
		} else {
			relay.AvgLatencyMs = int(math.Round(float64(relay.AvgLatencyMs)*0.7 + float64(latencyMs)*0.3))
		}
	}

	hbTime := relay.LastHeartbeat
	m.mu.Unlock()

	// Sync heartbeat to persistent store (outside lock to avoid I/O under mutex)
	if m.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), storeOpTimeout)
		err := m.store.UpdateRelayHeartbeat(ctx, relayID, hbTime)
		cancel() // release immediately, don't defer past the store call
		if err != nil {
			m.logger.Warn("Failed to sync heartbeat to store", "relay_id", relayID, "error", err)
		}
	}

	return nil
}
