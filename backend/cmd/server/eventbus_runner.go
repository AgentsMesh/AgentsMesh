package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"gorm.io/gorm"
)

func setupRunnerEventCallbacks(db *gorm.DB, runnerConnMgr *runner.RunnerConnectionManager, eventBus *eventbus.EventBus) {
	originalHeartbeatCallback := runnerConnMgr.GetHeartbeatCallback()
	runnerConnMgr.SetHeartbeatCallback(func(runnerID int64, data *runnerv1.HeartbeatData) {
		if originalHeartbeatCallback != nil {
			originalHeartbeatCallback(runnerID, data)
		}

		conn := runnerConnMgr.GetConnection(runnerID)
		if conn == nil || conn.IsOnlineEventSent() {
			return
		}

		var r struct {
			OrganizationID int64  `gorm:"column:organization_id"`
			NodeID         string `gorm:"column:node_id"`
			Status         string `gorm:"column:status"`
		}
		if err := db.Table("runners").Where("id = ?", runnerID).First(&r).Error; err != nil {
			return // Silently ignore - runner might not exist yet
		}

		if r.Status != "online" {
			eventData := &eventbus.RunnerStatusData{
				RunnerID:    runnerID,
				NodeID:      r.NodeID,
				Status:      "online",
				CurrentPods: len(data.Pods),
			}
			event, err := eventbus.NewEntityEvent(eventbus.EventRunnerOnline, r.OrganizationID, "runner", fmt.Sprintf("%d", runnerID), eventData)
			if err != nil {
				slog.Error("failed to create runner online event", "error", err)
			} else if err := eventBus.Publish(context.Background(), event); err != nil {
				slog.Error("failed to publish runner online event", "error", err)
			}
		}

		conn.MarkOnlineEventSent()
	})

	originalDisconnectCallback := runnerConnMgr.GetDisconnectCallback()
	runnerConnMgr.SetDisconnectCallback(func(runnerID int64) {
		var r struct {
			OrganizationID int64  `gorm:"column:organization_id"`
			NodeID         string `gorm:"column:node_id"`
		}
		if err := db.Table("runners").Where("id = ?", runnerID).First(&r).Error; err == nil {
			eventData := &eventbus.RunnerStatusData{
				RunnerID: runnerID,
				NodeID:   r.NodeID,
				Status:   "offline",
			}
			event, err := eventbus.NewEntityEvent(eventbus.EventRunnerOffline, r.OrganizationID, "runner", fmt.Sprintf("%d", runnerID), eventData)
			if err != nil {
				slog.Error("failed to create runner offline event", "error", err)
			} else if err := eventBus.Publish(context.Background(), event); err != nil {
				slog.Error("failed to publish runner offline event", "error", err)
			}
		}

		if originalDisconnectCallback != nil {
			originalDisconnectCallback(runnerID)
		}
	})
}
