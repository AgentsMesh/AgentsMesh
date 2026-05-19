package runner

import (
	"context"
	"encoding/json"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

func (pc *PodCoordinator) handleAutopilotIteration(runnerID int64, data *runnerv1.AutopilotIterationEvent) {
	ctx := context.Background()

	rp, err := pc.autopilotRepo.GetByKey(ctx, data.GetAutopilotKey())
	if err != nil {
		pc.logger.Error("failed to find autopilot pod for iteration",
			"autopilot_controller_key", data.GetAutopilotKey(),
			"error", err)
		return
	}

	var filesChangedJSON *string
	if len(data.GetFilesChanged()) > 0 {
		if jsonBytes, err := json.Marshal(data.GetFilesChanged()); err == nil {
			s := string(jsonBytes)
			filesChangedJSON = &s
		}
	}

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

	if err := pc.autopilotRepo.CreateIteration(ctx, iteration); err != nil {
		pc.logger.Error("failed to create autopilot iteration record",
			"autopilot_controller_key", data.GetAutopilotKey(),
			"iteration", data.GetIteration(),
			"error", err)
		return
	}

	now := time.Now()
	if err := pc.autopilotRepo.UpdateStatusByKey(ctx, data.GetAutopilotKey(), map[string]interface{}{
		"current_iteration": data.GetIteration(),
		"last_iteration_at": now,
		"updated_at":        now,
	}); err != nil {
		pc.logger.Error("failed to update autopilot pod iteration count",
			"autopilot_controller_key", data.GetAutopilotKey(),
			"error", err)
	}

	pc.logger.Debug("autopilot iteration recorded",
		"autopilot_controller_key", data.GetAutopilotKey(),
		"iteration", data.GetIteration(),
		"phase", data.GetPhase(),
		"summary", data.GetSummary())

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

func (pc *PodCoordinator) handleAutopilotThinking(runnerID int64, data *runnerv1.AutopilotThinkingEvent) {
	pc.logger.Debug("autopilot thinking received",
		"autopilot_controller_key", data.GetAutopilotKey(),
		"iteration", data.GetIteration(),
		"decision_type", data.GetDecisionType(),
		"reasoning", data.GetReasoning())

	if pc.onAutopilotThinkingChange != nil {
		pc.onAutopilotThinkingChange(runnerID, data)
	}
}
