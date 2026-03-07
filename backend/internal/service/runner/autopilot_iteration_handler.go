package runner

import (
	"context"
	"encoding/json"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// handleAutopilotIteration handles AutopilotController iteration events from runner
func (pc *PodCoordinator) handleAutopilotIteration(runnerID int64, data *runnerv1.AutopilotIterationEvent) {
	ctx := context.Background()

	// Get AutopilotController ID
	var rp agentpod.AutopilotController
	if err := pc.db.WithContext(ctx).
		Where("autopilot_controller_key = ?", data.GetAutopilotKey()).
		First(&rp).Error; err != nil {
		pc.logger.Error("failed to find autopilot pod for iteration",
			"autopilot_controller_key", data.GetAutopilotKey(),
			"error", err)
		return
	}

	// Serialize files changed as JSON
	var filesChangedJSON *string
	if len(data.GetFilesChanged()) > 0 {
		if jsonBytes, err := json.Marshal(data.GetFilesChanged()); err == nil {
			s := string(jsonBytes)
			filesChangedJSON = &s
		}
	}

	// Create iteration record
	var summary *string
	if s := data.GetSummary(); s != "" {
		summary = &s
	}

	iteration := &agentpod.AutopilotIteration{
		AutopilotControllerID: rp.ID,
		Iteration:             data.GetIteration(),
		Phase:                 data.GetPhase(),
		Summary:               summary,
		FilesChanged:          filesChangedJSON,
		DurationMs:            data.GetDurationMs(),
	}

	if err := pc.db.WithContext(ctx).Create(iteration).Error; err != nil {
		pc.logger.Error("failed to create autopilot iteration record",
			"autopilot_controller_key", data.GetAutopilotKey(),
			"iteration", data.GetIteration(),
			"error", err)
		return
	}

	// Update AutopilotController current_iteration and last_iteration_at
	now := time.Now()
	if err := pc.db.WithContext(ctx).
		Model(&agentpod.AutopilotController{}).
		Where("autopilot_controller_key = ?", data.GetAutopilotKey()).
		Updates(map[string]interface{}{
			"current_iteration": data.GetIteration(),
			"last_iteration_at": now,
			"updated_at":        now,
		}).Error; err != nil {
		pc.logger.Error("failed to update autopilot pod iteration count",
			"autopilot_controller_key", data.GetAutopilotKey(),
			"error", err)
	}

	pc.logger.Debug("autopilot iteration recorded",
		"autopilot_controller_key", data.GetAutopilotKey(),
		"iteration", data.GetIteration(),
		"phase", data.GetPhase(),
		"summary", data.GetSummary())

	// Notify via callback (to publish realtime event)
	if pc.onAutopilotIterationChange != nil {
		pc.onAutopilotIterationChange(
			data.GetAutopilotKey(),
			data.GetIteration(),
			data.GetPhase(),
			data.GetSummary(),
			data.GetFilesChanged(),
			data.GetDurationMs(),
		)
	}

	// Also send status_changed event to update current_iteration in frontend
	// The frontend relies on autopilot:status_changed to update the iteration counter display
	if pc.onAutopilotStatusChange != nil {
		pc.onAutopilotStatusChange(
			data.GetAutopilotKey(),
			rp.PodKey,
			rp.Phase, // Keep current phase
			data.GetIteration(),
			rp.MaxIterations,
			rp.CircuitBreakerState,
			"",
			rp.UserTakeover,
		)
	}
}

// handleAutopilotThinking handles AutopilotController thinking events from runner
// This event exposes the Control Agent's decision-making process to the user
func (pc *PodCoordinator) handleAutopilotThinking(runnerID int64, data *runnerv1.AutopilotThinkingEvent) {
	pc.logger.Debug("autopilot thinking received",
		"autopilot_controller_key", data.GetAutopilotKey(),
		"iteration", data.GetIteration(),
		"decision_type", data.GetDecisionType(),
		"reasoning", data.GetReasoning())

	// Notify via callback (to publish realtime event)
	if pc.onAutopilotThinkingChange != nil {
		pc.onAutopilotThinkingChange(runnerID, data)
	}
}
