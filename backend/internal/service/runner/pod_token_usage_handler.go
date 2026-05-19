package runner

import (
	"context"
	"time"

	tokenusagesvc "github.com/anthropics/agentsmesh/backend/internal/service/tokenusage"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

func (pc *PodCoordinator) SetTokenUsageService(svc *tokenusagesvc.Service) {
	pc.connectionManager.SetTokenUsageCallback(func(runnerID int64, data *runnerv1.TokenUsageReport) {
		pc.handleTokenUsage(runnerID, data, svc)
	})
}

func (pc *PodCoordinator) handleTokenUsage(runnerID int64, data *runnerv1.TokenUsageReport, svc *tokenusagesvc.Service) {
	if len(data.Models) == 0 || data.PodKey == "" {
		return
	}

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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pod, err := pc.podStore.GetByKey(ctx, data.PodKey)
	if err != nil {
		pc.logger.Error("failed to look up pod for token usage",
			"pod_key", data.PodKey,
			"runner_id", runnerID,
			"error", err,
		)
		return
	}

	if pod.RunnerID != runnerID {
		pc.logger.Warn("token usage rejected: runner does not own pod",
			"pod_key", data.PodKey,
			"runner_id", runnerID,
			"pod_runner_id", pod.RunnerID,
		)
		return
	}

	agentSlug := pod.AgentSlug
	if agentSlug == "" {
		agentSlug = "unknown"
	}
	const maxSlugLen = 50
	if len(agentSlug) > maxSlugLen {
		agentSlug = agentSlug[:maxSlugLen]
	}

	report := &runnerv1.TokenUsageReport{
		PodKey:                  data.PodKey,
		Models:                  models,
		PodStartedAtUnixSeconds: data.PodStartedAtUnixSeconds,
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
