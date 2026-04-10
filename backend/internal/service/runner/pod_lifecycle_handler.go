package runner

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// handlePodCreated handles pod creation event from runner (Proto type)
func (pc *PodCoordinator) handlePodCreated(runnerID int64, data *runnerv1.PodCreatedEvent) {
	ctx := context.Background()

	// Resolve any pending ACK (pod is alive, no need to wait further)
	pc.ackTracker.Resolve(data.PodKey)
	pc.clearInitReportCount(data.PodKey)

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

	if _, err := pc.podStore.UpdateByKey(ctx, data.PodKey, updates); err != nil {
		pc.logger.Error("failed to update pod on creation",
			"pod_key", data.PodKey,
			"error", err)
		return
	}

	// Register with pod router
	pc.podRouter.RegisterPod(data.PodKey, runnerID)

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

	// Status is decided by Runner and sent explicitly in data.Status.
	// Backend stores it directly — no interpretation of exit codes or error messages.
	status := data.Status
	if status == "" {
		// Backward compatibility: old runners without status field.
		// Fall back to ErrorMessage-based inference.
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

	// Only update if pod is still active — prevents overwriting a pod already
	// in terminal state (e.g., server-initiated TerminatePod pre-sets completed).
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
			// keep status as-is
		} else {
			pc.logger.Info("pod already in terminal state, skipping status update",
				"pod_key", data.PodKey)
			status = ""
		}
	}

	// Decrement runner pod count
	_ = pc.runnerRepo.DecrementPods(ctx, runnerID)

	// Unregister from pod router and clean up miss counter
	pc.podRouter.UnregisterPod(data.PodKey)
	pc.clearMissCount(data.PodKey)

	pc.logger.Info("pod terminated",
		"pod_key", data.PodKey,
		"runner_id", runnerID,
		"exit_code", data.ExitCode,
		"status", status)

	// Notify status change (skip if pod was already in terminal state)
	if pc.onStatusChange != nil && status != "" {
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

	// Resolve any pending ACK (error is also an acknowledgment)
	pc.ackTracker.Resolve(data.PodKey)

	ctx := context.Background()

	now := time.Now()

	// Handle errors during initialization (pod creation failed)
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
		// Initialization error -- decrement runner pod count
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

	// Handle errors during runtime (e.g., PTY read failure due to disk full).
	// Only store the error info; don't change status or finished_at here because
	// a pod_terminated event will follow shortly to finalize the pod lifecycle.
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
