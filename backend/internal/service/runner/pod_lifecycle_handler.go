package runner

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// handlePodCreated handles pod creation event from runner (Proto type)
func (pc *PodCoordinator) handlePodCreated(runnerID int64, data *runnerv1.PodCreatedEvent) {
	ctx := context.Background()

	now := time.Now()
	updates := map[string]interface{}{
		"pty_pid":       int(data.Pid),
		"status":        agentpod.StatusRunning,
		"started_at":    now,
		"last_activity": now,
	}

	// Store sandbox_path and branch_name for Resume functionality
	if data.SandboxPath != "" {
		updates["sandbox_path"] = data.SandboxPath
	}
	if data.BranchName != "" {
		updates["branch_name"] = data.BranchName
	}

	if err := pc.db.WithContext(ctx).
		Model(&agentpod.Pod{}).
		Where("pod_key = ?", data.PodKey).
		Updates(updates).Error; err != nil {
		pc.logger.Error("failed to update pod on creation",
			"pod_key", data.PodKey,
			"error", err)
		return
	}

	// Register with terminal router
	pc.terminalRouter.RegisterPod(data.PodKey, runnerID)

	pc.logger.Info("pod created",
		"pod_key", data.PodKey,
		"runner_id", runnerID,
		"pid", data.Pid,
		"sandbox_path", data.SandboxPath,
		"branch_name", data.BranchName)

	// Notify status change
	if pc.onStatusChange != nil {
		pc.onStatusChange(data.PodKey, agentpod.StatusRunning, "")
	}
}

// handlePodTerminated handles pod termination event from runner (Proto type)
func (pc *PodCoordinator) handlePodTerminated(runnerID int64, data *runnerv1.PodTerminatedEvent) {
	ctx := context.Background()

	now := time.Now()

	// Determine status: if the process exited with an error and provided early output,
	// mark as error so the user can see why the process failed.
	status := agentpod.StatusCompleted
	updates := map[string]interface{}{
		"agent_status": agentpod.AgentStatusIdle,
		"finished_at":  now,
		"pty_pid":      nil,
	}

	if data.ErrorMessage != "" {
		// Process exited with early output (e.g., invalid CLI arguments, PTY error).
		// Store the error message so the frontend can display why the pod failed.
		status = agentpod.StatusError
		updates["error_message"] = data.ErrorMessage
		// Preserve existing error_code if already set by a prior error event
		// (e.g., PTY_READ_ERROR from handlePodError). Only set the default
		// "process_exit" code when no specific error code exists yet.
		updates["error_code"] = gorm.Expr("COALESCE(NULLIF(error_code, ''), ?)", "process_exit")
	}
	updates["status"] = status

	if err := pc.db.WithContext(ctx).
		Model(&agentpod.Pod{}).
		Where("pod_key = ?", data.PodKey).
		Updates(updates).Error; err != nil {
		pc.logger.Error("failed to update pod on termination",
			"pod_key", data.PodKey,
			"error", err)
		return
	}

	// Decrement runner pod count
	pc.db.WithContext(ctx).Exec(
		"UPDATE runners SET current_pods = GREATEST(current_pods - 1, 0) WHERE id = ?",
		runnerID,
	)

	// Unregister from terminal router
	pc.terminalRouter.UnregisterPod(data.PodKey)

	pc.logger.Info("pod terminated",
		"pod_key", data.PodKey,
		"runner_id", runnerID,
		"exit_code", data.ExitCode,
		"has_early_output", data.ErrorMessage != "")

	// Notify status change
	if pc.onStatusChange != nil {
		pc.onStatusChange(data.PodKey, status, "")
	}
}

// handlePodError handles pod creation error event from runner (Proto type)
func (pc *PodCoordinator) handlePodError(runnerID int64, data *runnerv1.ErrorEvent) {
	if data.PodKey == "" {
		pc.logger.Warn("received pod error without pod_key, ignoring",
			"runner_id", runnerID,
			"code", data.Code,
			"message", data.Message)
		return
	}

	ctx := context.Background()

	now := time.Now()

	// Handle errors during initialization (pod creation failed)
	result := pc.db.WithContext(ctx).
		Model(&agentpod.Pod{}).
		Where("pod_key = ? AND status = ?", data.PodKey, agentpod.StatusInitializing).
		Updates(map[string]interface{}{
			"status":        agentpod.StatusError,
			"error_code":    data.Code,
			"error_message": data.Message,
			"finished_at":   now,
		})
	if result.Error != nil {
		pc.logger.Error("failed to update pod on error",
			"pod_key", data.PodKey,
			"error", result.Error)
		return
	}

	if result.RowsAffected > 0 {
		// Initialization error — decrement runner pod count
		pc.db.WithContext(ctx).Exec(
			"UPDATE runners SET current_pods = GREATEST(current_pods - 1, 0) WHERE id = ?",
			runnerID,
		)

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

	// Handle errors during runtime (e.g., PTY read failure due to disk full).
	// Only store the error info; don't change status or finished_at here because
	// a pod_terminated event will follow shortly to finalize the pod lifecycle.
	result = pc.db.WithContext(ctx).
		Model(&agentpod.Pod{}).
		Where("pod_key = ? AND status = ?", data.PodKey, agentpod.StatusRunning).
		Updates(map[string]interface{}{
			"error_code":    data.Code,
			"error_message": data.Message,
		})
	if result.Error != nil {
		pc.logger.Error("failed to store runtime error on pod",
			"pod_key", data.PodKey,
			"error", result.Error)
		return
	}

	if result.RowsAffected > 0 {
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

// handleRunnerDisconnect handles runner disconnection
func (pc *PodCoordinator) handleRunnerDisconnect(runnerID int64) {
	ctx := context.Background()

	// Mark runner as offline, but don't immediately orphan pods
	// Pods will be orphaned by reconcilePods if runner doesn't reconnect
	// and report them in heartbeat
	if err := pc.db.WithContext(ctx).
		Table("runners").
		Where("id = ?", runnerID).
		Update("status", "offline").Error; err != nil {
		pc.logger.Error("failed to mark runner as offline",
			"runner_id", runnerID,
			"error", err)
	}

	// Clear relay connection cache for this runner
	pc.relayConnectionCache.Delete(runnerID)

	pc.logger.Info("runner disconnected, pods will be reconciled on reconnect",
		"runner_id", runnerID)

	// Note: We intentionally don't mark pods as orphaned here
	// The runner might reconnect quickly (network glitch) and pods are still running
	// Pods will be properly reconciled when:
	// 1. Runner reconnects and sends heartbeat - reconcilePods will handle it
	// 2. Pod cleanup task runs and finds stale pods
}
