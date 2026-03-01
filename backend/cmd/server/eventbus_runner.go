package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/infra/eventbus"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/runner"
	runnerv1 "github.com/AgentsMesh/AgentsMesh/proto/gen/go/runner/v1"
	"gorm.io/gorm"
)

// setupRunnerEventCallbacks sets up runner connection manager callbacks to publish events
func setupRunnerEventCallbacks(db *gorm.DB, runnerConnMgr *runner.RunnerConnectionManager, eventBus *eventbus.EventBus) {
	// Wrap heartbeat callback to detect runner coming online (using Proto type)
	originalHeartbeatCallback := runnerConnMgr.GetHeartbeatCallback()
	runnerConnMgr.SetHeartbeatCallback(func(runnerID int64, data *runnerv1.HeartbeatData) {
		// Call original callback first
		if originalHeartbeatCallback != nil {
			originalHeartbeatCallback(runnerID, data)
		}

		// Publish runner online event (heartbeat indicates runner is online)
		// Only publish if this is likely a new connection or status change
		var r struct {
			OrganizationID int64  `gorm:"column:organization_id"`
			NodeID         string `gorm:"column:node_id"`
			Status         string `gorm:"column:status"`
		}
		if err := db.Table("runners").Where("id = ?", runnerID).First(&r).Error; err != nil {
			return // Silently ignore - runner might not exist yet
		}

		// Only publish event if status was offline (changed to online)
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
	})

	// Wrap disconnect callback to publish runner offline events
	originalDisconnectCallback := runnerConnMgr.GetDisconnectCallback()
	runnerConnMgr.SetDisconnectCallback(func(runnerID int64) {
		// Query runner first before status changes
		var r struct {
			OrganizationID int64  `gorm:"column:organization_id"`
			NodeID         string `gorm:"column:node_id"`
		}
		if err := db.Table("runners").Where("id = ?", runnerID).First(&r).Error; err == nil {
			// Publish runner offline event
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

		// Call original callback
		if originalDisconnectCallback != nil {
			originalDisconnectCallback(runnerID)
		}
	})
}
