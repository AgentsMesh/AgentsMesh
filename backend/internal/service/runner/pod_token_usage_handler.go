package runner

import (
	"context"
	"time"

	tokenusagesvc "github.com/anthropics/agentsmesh/backend/internal/service/tokenusage"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// SetTokenUsageService wires the token usage callback.
// The PodCoordinator looks up pod info (org, user, agent type) and delegates
// recording to the TokenUsageService.
func (pc *PodCoordinator) SetTokenUsageService(svc *tokenusagesvc.Service) {
	pc.connectionManager.SetTokenUsageCallback(func(runnerID int64, data *runnerv1.TokenUsageReport) {
		pc.handleTokenUsage(runnerID, data, svc)
	})
}

func (pc *PodCoordinator) handleTokenUsage(runnerID int64, data *runnerv1.TokenUsageReport, svc *tokenusagesvc.Service) {
	if len(data.Models) == 0 || data.PodKey == "" {
		return
	}

	// Cap the number of model entries to prevent abuse.
	// Use a local slice to avoid mutating the caller's proto message.
	const maxModels = 50
	models := data.Models
	if len(models) > maxModels {
		pc.logger.Warn("token usage report truncated: too many models",
			"pod_key", data.PodKey,
			"runner_id", runnerID,
			"count", len(models),
		)
		models = models[:maxModels]
	}

	// Use a bounded context so DB issues don't block indefinitely.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Look up the pod to retrieve org, user, and agent type info.
	pod, err := pc.podRepo.GetByKey(ctx, data.PodKey)
	if err != nil {
		pc.logger.Error("failed to look up pod for token usage",
			"pod_key", data.PodKey,
			"runner_id", runnerID,
			"error", err,
		)
		return
	}

	// Verify that the reporting runner actually owns this pod.
	if pod.RunnerID != runnerID {
		pc.logger.Warn("token usage rejected: runner does not own pod",
			"pod_key", data.PodKey,
			"runner_id", runnerID,
			"pod_runner_id", pod.RunnerID,
		)
		return
	}

	// Determine agent slug from the pod.
	agentSlug := pod.AgentSlug
	if agentSlug == "" {
		agentSlug = "unknown"
	}
	// Enforce DB column length limit (VARCHAR(50)).
	const maxSlugLen = 50
	if len(agentSlug) > maxSlugLen {
		agentSlug = agentSlug[:maxSlugLen]
	}

	// Pass a report with the (possibly truncated) model list.
	report := &runnerv1.TokenUsageReport{
		PodKey: data.PodKey,
		Models: models,
	}
	svc.RecordUsage(
		ctx,
		pod.OrganizationID,
		&pod.ID,
		data.PodKey,
		pod.CreatedByID,
		runnerID,
		agentSlug,
		report,
	)
}
