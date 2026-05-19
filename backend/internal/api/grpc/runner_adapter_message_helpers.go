package grpc

import (
	"context"
	"time"

	runnerDomain "github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

func (a *GRPCRunnerAdapter) handleInitialize(ctx context.Context, runnerID int64, conn *runner.GRPCConnection, req *runnerv1.InitializeRequest) {
	a.logger.Debug("received initialize request",
		"runner_id", runnerID,
		"protocol_version", req.ProtocolVersion,
	)

	var agents []*runnerv1.AgentInfo
	if a.agentsProvider != nil {
		types := a.agentsProvider.GetAgentsForRunner()
		agents = make([]*runnerv1.AgentInfo, len(types))
		for i, t := range types {
			agents[i] = &runnerv1.AgentInfo{
				Slug:    t.Slug,
				Name:    t.Name,
				Command: t.Executable,
			}
		}
		a.logger.Debug("sending agents to runner",
			"runner_id", runnerID,
			"agent_count", len(agents),
		)
	}

	if req.RunnerInfo != nil && a.runnerService != nil {
		hostInfo := map[string]interface{}{
			"os":       req.RunnerInfo.GetOs(),
			"arch":     req.RunnerInfo.GetArch(),
			"hostname": req.RunnerInfo.GetHostname(),
		}
		if err := a.runnerService.UpdateRunnerVersionAndHostInfo(ctx, runnerID, req.RunnerInfo.GetVersion(), hostInfo); err != nil {
			a.logger.Error("Failed to update runner version and host info", "runner_id", runnerID, "error", err)
		}
	}

	result := &runnerv1.InitializeResult{
		ProtocolVersion: 2,
		ServerInfo: &runnerv1.ServerInfo{
			Version: "1.0.0",
		},
		Agents: agents,
		Features: []string{
			"files_to_create",
			"work_dir_config",
			"prompt",
		},
	}

	response := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_InitializeResult{
			InitializeResult: result,
		},
		Timestamp: time.Now().UnixMilli(),
	}

	if err := conn.SendMessage(response); err != nil {
		a.logger.Warn("failed to send initialize result", "runner_id", runnerID, "error", err)
	}
}

func (a *GRPCRunnerAdapter) handleInitialized(ctx context.Context, runnerID int64, conn *runner.GRPCConnection, msg *runnerv1.InitializedConfirm) {
	a.logger.Info("Runner initialized",
		"runner_id", runnerID,
		"available_agents", msg.AvailableAgents,
		"agent_versions", len(msg.AgentVersions),
	)

	a.connManager.HandleInitialized(runnerID, msg.AvailableAgents)

	if a.runnerService != nil {
		_ = a.runnerService.UpdateLastSeen(ctx, runnerID)
		if err := a.runnerService.UpdateAvailableAgents(ctx, runnerID, msg.AvailableAgents); err != nil {
			a.logger.Error("failed to update available agents",
				"runner_id", runnerID,
				"error", err,
			)
		}

		// Old Runners won't send AgentVersions.
		if len(msg.AgentVersions) > 0 {
			a.persistAgentVersions(ctx, runnerID, msg.AgentVersions, false)
		}
	}
}

func (a *GRPCRunnerAdapter) handleHeartbeatAgentVersions(ctx context.Context, runnerID int64, versions []*runnerv1.AgentVersionInfo) {
	a.logger.Info("Agent version change detected via heartbeat",
		"runner_id", runnerID,
		"changes", len(versions),
	)
	a.persistAgentVersions(ctx, runnerID, versions, true)
}

func (a *GRPCRunnerAdapter) handlePong(runnerID int64, conn *runner.GRPCConnection, pong *runnerv1.PongEvent) {
	conn.UpdateLastPong()
	rtt := time.Now().UnixMilli() - pong.PingTimestamp
	a.logger.Debug("downstream pong received",
		"runner_id", runnerID,
		"rtt_ms", rtt,
	)
}

func (a *GRPCRunnerAdapter) persistAgentVersions(ctx context.Context, runnerID int64, versions []*runnerv1.AgentVersionInfo, isDelta bool) {
	if a.runnerService == nil {
		return
	}

	var finalVersions []runnerDomain.AgentVersion

	if isDelta {
		incoming := make(map[string]runnerDomain.AgentVersion, len(versions))
		for _, v := range versions {
			incoming[v.Slug] = runnerDomain.AgentVersion{
				Slug:    v.Slug,
				Version: v.Version,
				Path:    v.Path,
			}
			a.logger.Info("Agent version updated",
				"runner_id", runnerID,
				"agent", v.Slug,
				"version", v.Version,
				"path", v.Path,
			)
		}

		if err := a.runnerService.MergeAgentVersions(ctx, runnerID, incoming); err != nil {
			a.logger.Error("failed to merge agent versions",
				"runner_id", runnerID,
				"error", err,
			)
		}
		return
	}

	finalVersions = make([]runnerDomain.AgentVersion, 0, len(versions))
	for _, v := range versions {
		finalVersions = append(finalVersions, runnerDomain.AgentVersion{
			Slug:    v.Slug,
			Version: v.Version,
			Path:    v.Path,
		})
		a.logger.Info("Agent version detected",
			"runner_id", runnerID,
			"agent", v.Slug,
			"version", v.Version,
			"path", v.Path,
		)
	}
	if err := a.runnerService.UpdateAgentVersions(ctx, runnerID, finalVersions); err != nil {
		a.logger.Error("failed to update agent versions",
			"runner_id", runnerID,
			"error", err,
		)
	}
}
