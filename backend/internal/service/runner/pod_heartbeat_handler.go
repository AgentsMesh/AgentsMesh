package runner

import (
	"context"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

func (pc *PodCoordinator) handleHeartbeat(runnerID int64, data *runnerv1.HeartbeatData) {
	ctx := context.Background()

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

	reportedPodKeys := make(map[string]bool)
	for _, p := range data.Pods {
		reportedPodKeys[p.PodKey] = true
		if p.AgentStatus != "" {
			_ = pc.podStore.UpdateField(ctx, p.PodKey, "agent_status", p.AgentStatus)
		}
	}

	pc.reconcilePods(ctx, runnerID, reportedPodKeys)
}
