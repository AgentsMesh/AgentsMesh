package runner

import (
	"context"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// handleHeartbeat handles heartbeat from a runner (Proto type)
// Heartbeats are batched via HeartbeatBatcher for high-scale performance
func (pc *PodCoordinator) handleHeartbeat(runnerID int64, data *runnerv1.HeartbeatData) {
	ctx := context.Background()

	// Record heartbeat via batcher (batched DB writes + immediate Redis update)
	// Note: RunnerVersion not available in Proto, using empty string
	if err := pc.heartbeatBatcher.RecordHeartbeat(
		ctx,
		runnerID,
		len(data.Pods),
		"online",
		"", // RunnerVersion not in Proto HeartbeatData
	); err != nil {
		pc.logger.Error("failed to record heartbeat",
			"runner_id", runnerID,
			"error", err)
	}

	// Update relay connection cache
	if data.RelayConnections != nil {
		connections := make([]RelayConnectionInfo, 0, len(data.RelayConnections))
		for _, rc := range data.RelayConnections {
			connections = append(connections, RelayConnectionInfo{
				PodKey:      rc.PodKey,
				RelayURL:    rc.RelayUrl,
				SessionID:   rc.SessionId,
				Connected:   rc.Connected,
				ConnectedAt: time.UnixMilli(rc.ConnectedAt),
			})
		}
		pc.relayConnectionCache.Update(runnerID, connections)
	}

	// Reconcile pods
	reportedPodKeys := make(map[string]bool)
	for _, p := range data.Pods {
		reportedPodKeys[p.PodKey] = true
		// Silently sync agent_status from heartbeat (no WebSocket event)
		if p.AgentStatus != "" {
			_ = pc.podRepo.UpdateField(ctx, p.PodKey, "agent_status", p.AgentStatus)
		}
	}

	pc.reconcilePods(ctx, runnerID, reportedPodKeys)
}
