package runner

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// handleAutopilotControllerStatus handles AutopilotController status events from runner
func (pc *PodCoordinator) handleAutopilotControllerStatus(runnerID int64, data *runnerv1.AutopilotStatusEvent) {
	ctx := context.Background()

	status := data.GetStatus()
	if status == nil {
		pc.logger.Warn("autopilot pod status event missing status",
			"autopilot_controller_key", data.GetAutopilotKey())
		return
	}

	now := time.Now()

	// Determine user_takeover from phase
	userTakeover := status.GetPhase() == agentpod.AutopilotPhaseUserTakeover

	updates := map[string]interface{}{
		"phase":                 status.GetPhase(),
		"current_iteration":     status.GetCurrentIteration(),
		"circuit_breaker_state": status.GetCircuitBreakerState(),
		"user_takeover":         userTakeover,
		"updated_at":            now,
	}

	// Update circuit breaker reason if present
	if reason := status.GetCircuitBreakerReason(); reason != "" {
		updates["circuit_breaker_reason"] = reason
	}

	// Update last_iteration_at if iteration changed
	if status.GetLastIterationAt() > 0 {
		updates["last_iteration_at"] = time.Unix(status.GetLastIterationAt(), 0)
	}

	// Update started_at if provided and not already set
	if status.GetStartedAt() > 0 {
		updates["started_at"] = time.Unix(status.GetStartedAt(), 0)
	}

	// Set completed_at for terminal states
	if status.GetPhase() == agentpod.AutopilotPhaseCompleted ||
		status.GetPhase() == agentpod.AutopilotPhaseFailed ||
		status.GetPhase() == agentpod.AutopilotPhaseStopped {
		updates["completed_at"] = now
	}

	// Set approval_request_at for waiting_approval state
	if status.GetPhase() == agentpod.AutopilotPhaseWaitingApproval {
		updates["approval_request_at"] = now
	}

	if err := pc.autopilotRepo.UpdateStatusByKey(ctx, data.GetAutopilotKey(), updates); err != nil {
		pc.logger.Error("failed to update autopilot pod status",
			"autopilot_controller_key", data.GetAutopilotKey(),
			"error", err)
		return
	}

	pc.logger.Info("autopilot pod status updated",
		"autopilot_controller_key", data.GetAutopilotKey(),
		"pod_key", data.GetPodKey(),
		"phase", status.GetPhase(),
		"iteration", status.GetCurrentIteration(),
		"circuit_breaker", status.GetCircuitBreakerState())

	// Notify via callback (to publish realtime event)
	if pc.onAutopilotStatusChange != nil {
		pc.onAutopilotStatusChange(
			data.GetAutopilotKey(),
			data.GetPodKey(),
			status.GetPhase(),
			status.GetCurrentIteration(),
			status.GetMaxIterations(),
			status.GetCircuitBreakerState(),
			status.GetCircuitBreakerReason(),
			userTakeover,
		)
	}
}

// handleAutopilotControllerCreated handles AutopilotController creation confirmation from runner
func (pc *PodCoordinator) handleAutopilotControllerCreated(runnerID int64, data *runnerv1.AutopilotCreatedEvent) {
	ctx := context.Background()

	now := time.Now()
	updates := map[string]interface{}{
		"phase":      agentpod.AutopilotPhaseRunning,
		"started_at": now,
		"updated_at": now,
	}

	if err := pc.autopilotRepo.UpdateStatusByKey(ctx, data.GetAutopilotKey(), updates); err != nil {
		pc.logger.Error("failed to update autopilot pod on creation",
			"autopilot_controller_key", data.GetAutopilotKey(),
			"error", err)
		return
	}

	pc.logger.Info("autopilot pod created",
		"autopilot_controller_key", data.GetAutopilotKey(),
		"pod_key", data.GetPodKey(),
		"runner_id", runnerID)

	// Notify via callback
	if pc.onAutopilotStatusChange != nil {
		// Fetch the full AutopilotController to get max_iterations
		rp, err := pc.autopilotRepo.GetByKey(ctx, data.GetAutopilotKey())
		if err == nil {
			pc.onAutopilotStatusChange(
				data.GetAutopilotKey(),
				data.GetPodKey(),
				agentpod.AutopilotPhaseRunning,
				0,
				rp.MaxIterations,
				agentpod.CircuitBreakerClosed,
				"",
				false,
			)
		}
	}
}

// handleAutopilotControllerTerminated handles AutopilotController termination event from runner
func (pc *PodCoordinator) handleAutopilotControllerTerminated(runnerID int64, data *runnerv1.AutopilotTerminatedEvent) {
	ctx := context.Background()

	now := time.Now()
	phase := agentpod.AutopilotPhaseStopped
	if reason := data.GetReason(); reason != "" {
		switch reason {
		case "completed":
			phase = agentpod.AutopilotPhaseCompleted
		case "failed":
			phase = agentpod.AutopilotPhaseFailed
		}
	}

	updates := map[string]interface{}{
		"phase":        phase,
		"completed_at": now,
		"updated_at":   now,
	}

	if err := pc.autopilotRepo.UpdateStatusByKey(ctx, data.GetAutopilotKey(), updates); err != nil {
		pc.logger.Error("failed to update autopilot pod on termination",
			"autopilot_controller_key", data.GetAutopilotKey(),
			"error", err)
		return
	}

	pc.logger.Info("autopilot pod terminated",
		"autopilot_controller_key", data.GetAutopilotKey(),
		"runner_id", runnerID,
		"reason", data.GetReason())

	// Notify via callback
	if pc.onAutopilotStatusChange != nil {
		// Fetch the full AutopilotController to get details
		rp, err := pc.autopilotRepo.GetByKey(ctx, data.GetAutopilotKey())
		if err == nil {
			pc.onAutopilotStatusChange(
				data.GetAutopilotKey(),
				rp.PodKey,
				phase,
				rp.CurrentIteration,
				rp.MaxIterations,
				rp.CircuitBreakerState,
				"",
				rp.UserTakeover,
			)
		}
	}
}
