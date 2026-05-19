package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/service/relay"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"gorm.io/gorm"
)

func setupRelayTokenRefreshCallback(
	db *gorm.DB,
	runnerConnMgr *runner.RunnerConnectionManager,
	tokenGenerator *relay.TokenGenerator,
	commandSender runner.RunnerCommandSender,
) {
	runnerConnMgr.SetRequestRelayTokenCallback(func(runnerID int64, data *runnerv1.RequestRelayTokenEvent) {
		slog.Info("Received relay token refresh request",
			"runner_id", runnerID,
			"pod_key", data.PodKey,
		)

		var pod struct {
			OrganizationID int64  `gorm:"column:organization_id"`
			RunnerID       int64  `gorm:"column:runner_id"`
			Status         string `gorm:"column:status"`
		}
		if err := db.Table("pods").Where("pod_key = ?", data.PodKey).First(&pod).Error; err != nil {
			slog.Error("failed to get pod for relay token refresh",
				"pod_key", data.PodKey,
				"error", err,
			)
			return
		}

		if pod.RunnerID != runnerID {
			slog.Warn("runner does not own pod for relay token refresh",
				"runner_id", runnerID,
				"pod_runner_id", pod.RunnerID,
				"pod_key", data.PodKey,
			)
			return
		}

		if pod.Status != "running" && pod.Status != "initializing" && pod.Status != "disconnected" {
			slog.Warn("pod is not active for relay token refresh",
				"pod_key", data.PodKey,
				"status", pod.Status,
			)
			return
		}

		newToken, err := tokenGenerator.GenerateToken(
			data.PodKey,
			runnerID,
			0, // userID=0 for runner token
			pod.OrganizationID,
			time.Hour,
		)
		if err != nil {
			slog.Error("failed to generate new relay token",
				"pod_key", data.PodKey,
				"error", err,
			)
			return
		}

		if err := commandSender.SendSubscribePod(
			context.Background(),
			runnerID,
			data.PodKey,
			data.RelayUrl,
			newToken,
			"",   // localToken: cloud-relay refresh doesn't touch local relay
			true, // include snapshot (runner will resend after reconnect)
			1000, // snapshot history lines
		); err != nil {
			slog.Error("failed to send subscribe pod with new token",
				"runner_id", runnerID,
				"pod_key", data.PodKey,
				"error", err,
			)
			return
		}

		slog.Info("Sent new relay token to runner",
			"runner_id", runnerID,
			"pod_key", data.PodKey,
		)
	})
}
