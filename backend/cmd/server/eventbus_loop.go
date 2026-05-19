package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	"github.com/anthropics/agentsmesh/backend/internal/service/instance"
	loop "github.com/anthropics/agentsmesh/backend/internal/service/loop"
)

func setupLoopEventSubscriptions(eventBus *eventbus.EventBus, loopOrchestrator *loop.LoopOrchestrator) {
	eventBus.Subscribe(eventbus.EventPodTerminated, func(event *eventbus.Event) {
		var data eventbus.PodStatusChangedData
		if err := json.Unmarshal(event.Data, &data); err != nil {
			slog.Error("failed to unmarshal pod terminated event for loop", "error", err)
			return
		}

		var finishedAt *time.Time
		now := time.Now()
		finishedAt = &now

		loopOrchestrator.HandlePodTerminated(context.Background(), data.PodKey, data.Status, finishedAt)
	})

	eventBus.Subscribe(eventbus.EventPodStatusChanged, func(event *eventbus.Event) {
		var data eventbus.PodStatusChangedData
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return
		}

		switch data.Status {
		case agentpod.StatusCompleted, agentpod.StatusError:
			var finishedAt *time.Time
			now := time.Now()
			finishedAt = &now
			loopOrchestrator.HandlePodTerminated(context.Background(), data.PodKey, data.Status, finishedAt)
		}
	})

	// Autopilot status changed → detect terminal phases and handle completion.
	// This is the single path for Autopilot termination detection:
	//   Runner gRPC → PodCoordinator → onAutopilotStatusChange callback → EventAutopilotStatusChanged
	// Note: EventAutopilotTerminated is NOT used because it is never published by the callback chain.
	eventBus.Subscribe(eventbus.EventAutopilotStatusChanged, func(event *eventbus.Event) {
		var data eventbus.AutopilotStatusChangedData
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return
		}

		switch data.Phase {
		case agentpod.AutopilotPhaseCompleted, agentpod.AutopilotPhaseFailed, agentpod.AutopilotPhaseStopped:
			loopOrchestrator.HandleAutopilotTerminated(context.Background(), data.AutopilotControllerKey, data.Phase)
		}
	})

	slog.Info("Loop event subscriptions registered")
}

// setupOrgAwarenessRefresh subscribes to Runner online/offline events to
// trigger immediate refresh of the OrgAwarenessService cache.
// This ensures the local org set is updated as soon as Runners connect/disconnect,
// without waiting for the periodic 30s refresh.
func setupOrgAwarenessRefresh(eventBus *eventbus.EventBus, orgAwareness *instance.OrgAwarenessService) {
	eventBus.Subscribe(eventbus.EventRunnerOnline, func(event *eventbus.Event) {
		orgAwareness.Refresh()
	})

	eventBus.Subscribe(eventbus.EventRunnerOffline, func(event *eventbus.Event) {
		orgAwareness.Refresh()
	})

	slog.Info("OrgAwareness runner event subscriptions registered")
}
