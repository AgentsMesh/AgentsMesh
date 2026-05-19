package runner

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	otelinit "github.com/anthropics/agentsmesh/backend/internal/infra/otel"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

func (pc *PodCoordinator) handlePodCreated(runnerID int64, data *runnerv1.PodCreatedEvent) {
	ctx := context.Background()

	pc.ackTracker.Resolve(data.PodKey)
	pc.clearInitReportCount(data.PodKey)

	now := time.Now()
	updates := map[string]interface{}{
		"pty_pid":       int(data.Pid),
		"status":        agentpod.StatusRunning,
		"started_at":    now,
		"last_activity": now,
	}

	if data.SandboxPath != "" {
		updates["sandbox_path"] = data.SandboxPath
	}
	if data.BranchName != "" {
		updates["branch_name"] = data.BranchName
	}

	if _, err := pc.podStore.UpdateByKey(ctx, data.PodKey, updates); err != nil {
		pc.logger.Error("failed to update pod on creation",
			"pod_key", data.PodKey,
			"error", err)
		return
	}

	pc.podRouter.RegisterPod(data.PodKey, runnerID)
	otelinit.PodActiveCount.Add(ctx, 1)

	pc.logger.Info("pod created",
		"pod_key", data.PodKey,
		"runner_id", runnerID,
		"pid", data.Pid,
		"sandbox_path", data.SandboxPath,
		"branch_name", data.BranchName)

	if pc.onStatusChange != nil {
		pc.onStatusChange(data.PodKey, agentpod.StatusRunning, "")
	}
}

func (pc *PodCoordinator) handlePodTerminated(runnerID int64, data *runnerv1.PodTerminatedEvent) {
	ctx := context.Background()

	now := time.Now()

	status := data.Status
	if status == "" {
		// Back-compat for old runners without status field — infer from ErrorMessage.
		if data.ErrorMessage != "" {
			status = agentpod.StatusError
		} else {
			status = agentpod.StatusCompleted
		}
	}

	updates := map[string]interface{}{
		"agent_status": agentpod.AgentStatusIdle,
		"finished_at":  now,
		"pty_pid":      nil,
		"status":       status,
	}
	if data.ErrorMessage != "" {
		updates["error_message"] = data.ErrorMessage
	}

	if status == agentpod.StatusError {
		rowsAffected, err := pc.podStore.UpdateTerminatedIfActive(ctx, data.PodKey, updates, "process_exit")
		if err != nil {
			pc.logger.Error("failed to update pod on termination",
				"pod_key", data.PodKey, "error", err)
			return
		}
		if rowsAffected == 0 {
			pc.logger.Info("pod already in terminal state, skipping status update",
				"pod_key", data.PodKey)
			status = ""
		}
	} else {
		rowsAffected, err := pc.podStore.UpdateByKeyAndActiveStatus(ctx, data.PodKey, updates)
		if err != nil {
			pc.logger.Error("failed to update pod on termination",
				"pod_key", data.PodKey, "error", err)
			return
		}
		if rowsAffected > 0 {
		} else {
			pc.logger.Info("pod already in terminal state, skipping status update",
				"pod_key", data.PodKey)
			status = ""
		}
	}

	_ = pc.runnerRepo.DecrementPods(ctx, runnerID)
	otelinit.PodActiveCount.Add(ctx, -1)

	pc.podRouter.UnregisterPod(data.PodKey)
	pc.clearMissCount(data.PodKey)

	pc.logger.Info("pod terminated",
		"pod_key", data.PodKey,
		"runner_id", runnerID,
		"exit_code", data.ExitCode,
		"status", status)

	if pc.onStatusChange != nil && status != "" {
		pc.onStatusChange(data.PodKey, status, "")
	}
}

func (pc *PodCoordinator) handlePodError(runnerID int64, data *runnerv1.ErrorEvent) {
	if data.PodKey == "" {
		pc.logger.Warn("received pod error without pod_key, ignoring",
			"runner_id", runnerID,
			"code", data.Code,
			"message", data.Message)
		return
	}

	pc.ackTracker.Resolve(data.PodKey)

	ctx := context.Background()

	now := time.Now()

	rowsAffected, err := pc.podStore.UpdateByKeyAndStatusCounted(ctx, data.PodKey, agentpod.StatusInitializing, map[string]interface{}{
		"status":        agentpod.StatusError,
		"error_code":    data.Code,
		"error_message": data.Message,
		"finished_at":   now,
	})
	if err != nil {
		pc.logger.Error("failed to update pod on error",
			"pod_key", data.PodKey,
			"error", err)
		return
	}

	if rowsAffected > 0 {
		_ = pc.runnerRepo.DecrementPods(ctx, runnerID)

		pc.logger.Error("pod creation failed",
			"pod_key", data.PodKey,
			"runner_id", runnerID,
			"error_code", data.Code,
			"error_message", data.Message)

		if pc.onStatusChange != nil {
			pc.onStatusChange(data.PodKey, agentpod.StatusError, "")
		}
		return
	}

	// Runtime errors (e.g. PTY read after disk full) — keep status/finished_at unchanged
	// because pod_terminated will arrive shortly to finalize the lifecycle.
	rowsAffected, err = pc.podStore.UpdateByKeyAndStatusCounted(ctx, data.PodKey, agentpod.StatusRunning, map[string]interface{}{
		"error_code":    data.Code,
		"error_message": data.Message,
	})
	if err != nil {
		pc.logger.Error("failed to store runtime error on pod",
			"pod_key", data.PodKey,
			"error", err)
		return
	}

	if rowsAffected > 0 {
		pc.logger.Error("pod runtime error recorded",
			"pod_key", data.PodKey,
			"runner_id", runnerID,
			"error_code", data.Code,
			"error_message", data.Message)
		return
	}

	pc.logger.Warn("pod error ignored: pod not in initializing or running state",
		"pod_key", data.PodKey,
		"runner_id", runnerID)
}
