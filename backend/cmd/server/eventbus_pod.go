package main

import (
	"context"
	"fmt"
	"log/slog"

	notifDomain "github.com/anthropics/agentsmesh/backend/internal/domain/notification"
	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	notifService "github.com/anthropics/agentsmesh/backend/internal/service/notification"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"gorm.io/gorm"
)

// setupPodEventCallbacks sets up pod coordinator callbacks to publish events
func setupPodEventCallbacks(db *gorm.DB, podCoordinator *runner.PodCoordinator, eventBus *eventbus.EventBus, notifDispatcher *notifService.Dispatcher) {
	podCoordinator.SetStatusChangeCallback(func(podKey string, status string, agentStatus string) {
		// Query pod to get organization ID and other metadata
		var pod struct {
			OrganizationID int64   `gorm:"column:organization_id"`
			CreatedByID    int64   `gorm:"column:created_by_id"`
			Status         string  `gorm:"column:status"`
			AgentStatus    string  `gorm:"column:agent_status"`
			ErrorCode      *string `gorm:"column:error_code"`
			ErrorMessage   *string `gorm:"column:error_message"`
		}
		if err := db.Table("pods").Where("pod_key = ?", podKey).First(&pod).Error; err != nil {
			slog.Error("failed to get pod for event", "pod_key", podKey, "error", err)
			return
		}

		// Determine event type
		// Note: pod.Status is the previous status (before this callback updates it)
		var eventType eventbus.EventType
		if agentStatus != "" {
			eventType = eventbus.EventPodAgentChanged
		} else if status == agentpod.StatusCompleted || status == agentpod.StatusTerminated {
			eventType = eventbus.EventPodTerminated
		} else if pod.Status == agentpod.StatusInitializing && status == agentpod.StatusRunning {
			// Pod transitioned from initializing to running - this is a newly created pod starting
			eventType = eventbus.EventPodCreated
		} else {
			eventType = eventbus.EventPodStatusChanged
		}

		// Create and publish event
		data := &eventbus.PodStatusChangedData{
			PodKey:         podKey,
			Status:         status,
			PreviousStatus: "",
			AgentStatus:    agentStatus,
		}
		// Include error details if present
		if pod.ErrorCode != nil {
			data.ErrorCode = *pod.ErrorCode
		}
		if pod.ErrorMessage != nil {
			data.ErrorMessage = *pod.ErrorMessage
		}
		event, err := eventbus.NewEntityEvent(eventType, pod.OrganizationID, "pod", podKey, data)
		if err != nil {
			slog.Error("failed to create pod event", "error", err)
			return
		}
		if err := eventBus.Publish(context.Background(), event); err != nil {
			slog.Error("failed to publish pod event", "error", err)
		}

		// Skip task:completed if OSC 777 already notified for this pod (dedup window: 30s)
		if status == agentpod.StatusCompleted || status == agentpod.StatusTerminated || status == agentpod.StatusError {
			if wasOSCNotifRecent(podKey) {
				slog.Debug("skipping task:completed notification, recent OSC notification exists", "pod_key", podKey)
			} else if err := notifDispatcher.Dispatch(context.Background(), &notifDomain.NotificationRequest{
				OrganizationID:    pod.OrganizationID,
				Source:            notifDomain.SourceTaskCompleted,
				SourceEntityID:    podKey,
				RecipientResolver: fmt.Sprintf("pod_creator:%s", podKey),
				Title:             "Task Completed",
				Body:              fmt.Sprintf("Pod %s finished with status: %s", podKey, status),
				Link:              fmt.Sprintf("/workspace?pod=%s", podKey),
				Priority:          notifDomain.PriorityNormal,
			}); err != nil {
				slog.Error("failed to dispatch task completed notification", "pod_key", podKey, "error", err)
			}
		}
	})

	// Set up init progress callback
	podCoordinator.SetInitProgressCallback(func(podKey string, phase string, progress int, message string) {
		// Query pod to get organization ID
		var pod struct {
			OrganizationID int64 `gorm:"column:organization_id"`
		}
		if err := db.Table("pods").Where("pod_key = ?", podKey).First(&pod).Error; err != nil {
			slog.Error("failed to get pod for init progress event", "pod_key", podKey, "error", err)
			return
		}

		// Create and publish init progress event
		data := &eventbus.PodInitProgressData{
			PodKey:   podKey,
			Phase:    phase,
			Progress: progress,
			Message:  message,
		}
		event, err := eventbus.NewEntityEvent(eventbus.EventPodInitProgress, pod.OrganizationID, "pod", podKey, data)
		if err != nil {
			slog.Error("failed to create pod init progress event", "error", err)
			return
		}
		if err := eventBus.Publish(context.Background(), event); err != nil {
			slog.Error("failed to publish pod init progress event", "error", err)
		}
	})

	// Set up AutopilotController status change callback
	podCoordinator.SetAutopilotStatusChangeCallback(func(
		autopilotKey string,
		podKey string,
		phase string,
		iteration int32,
		maxIterations int32,
		circuitBreakerState string,
		circuitBreakerReason string,
		userTakeover bool,
	) {
		// Query AutopilotController to get organization ID
		var autopilot struct {
			OrganizationID int64 `gorm:"column:organization_id"`
		}
		if err := db.Table("autopilot_controllers").Where("autopilot_controller_key = ?", autopilotKey).First(&autopilot).Error; err != nil {
			slog.Error("failed to get autopilot for status event", "autopilot_key", autopilotKey, "error", err)
			return
		}

		// Create and publish AutopilotController status event
		data := &eventbus.AutopilotStatusChangedData{
			AutopilotControllerKey: autopilotKey,
			PodKey:                 podKey,
			Phase:                  phase,
			CurrentIteration:       iteration,
			MaxIterations:          maxIterations,
			CircuitBreakerState:    circuitBreakerState,
			CircuitBreakerReason:   circuitBreakerReason,
			UserTakeover:           userTakeover,
		}
		event, err := eventbus.NewEntityEvent(eventbus.EventAutopilotStatusChanged, autopilot.OrganizationID, "autopilot_controller", autopilotKey, data)
		if err != nil {
			slog.Error("failed to create autopilot status event", "error", err)
			return
		}
		if err := eventBus.Publish(context.Background(), event); err != nil {
			slog.Error("failed to publish autopilot status event", "error", err)
		}
	})

	// Set up AutopilotController iteration callback
	podCoordinator.SetAutopilotIterationChangeCallback(func(
		autopilotKey string,
		iteration int32,
		phase string,
		summary string,
		filesChanged []string,
		durationMs int64,
	) {
		// Query AutopilotController to get organization ID
		var autopilot struct {
			OrganizationID int64 `gorm:"column:organization_id"`
		}
		if err := db.Table("autopilot_controllers").Where("autopilot_controller_key = ?", autopilotKey).First(&autopilot).Error; err != nil {
			slog.Error("failed to get autopilot for iteration event", "autopilot_key", autopilotKey, "error", err)
			return
		}

		// Create and publish AutopilotController iteration event
		data := &eventbus.AutopilotIterationData{
			AutopilotControllerKey: autopilotKey,
			Iteration:              iteration,
			Phase:                  phase,
			Summary:                summary,
			FilesChanged:           filesChanged,
			DurationMs:             durationMs,
		}
		event, err := eventbus.NewEntityEvent(eventbus.EventAutopilotIteration, autopilot.OrganizationID, "autopilot_controller", autopilotKey, data)
		if err != nil {
			slog.Error("failed to create autopilot iteration event", "error", err)
			return
		}
		if err := eventBus.Publish(context.Background(), event); err != nil {
			slog.Error("failed to publish autopilot iteration event", "error", err)
		}
	})

	// Set up AutopilotController thinking callback
	podCoordinator.SetAutopilotThinkingCallback(func(runnerID int64, protoData *runnerv1.AutopilotThinkingEvent) {
		autopilotKey := protoData.GetAutopilotKey()

		// Query AutopilotController to get organization ID
		var autopilot struct {
			OrganizationID int64 `gorm:"column:organization_id"`
		}
		if err := db.Table("autopilot_controllers").Where("autopilot_controller_key = ?", autopilotKey).First(&autopilot).Error; err != nil {
			slog.Error("failed to get autopilot for thinking event", "autopilot_key", autopilotKey, "error", err)
			return
		}

		// Convert Proto to eventbus data
		data := protoToEventbusThinking(protoData)

		// Create and publish AutopilotController thinking event
		event, err := eventbus.NewEntityEvent(eventbus.EventAutopilotThinking, autopilot.OrganizationID, "autopilot_controller", autopilotKey, data)
		if err != nil {
			slog.Error("failed to create autopilot thinking event", "error", err)
			return
		}
		if err := eventBus.Publish(context.Background(), event); err != nil {
			slog.Error("failed to publish autopilot thinking event", "error", err)
		}

		slog.Debug("published autopilot thinking event",
			"autopilot_key", autopilotKey,
			"decision_type", data.DecisionType,
			"iteration", data.Iteration)
	})
}
