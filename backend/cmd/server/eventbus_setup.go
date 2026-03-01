package main

import (
	"encoding/json"
	"log/slog"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/infra/eventbus"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/infra/websocket"
	runnerv1 "github.com/AgentsMesh/AgentsMesh/proto/gen/go/runner/v1"
)

// protoToEventbusThinking converts Proto AutopilotThinkingEvent to eventbus data
func protoToEventbusThinking(data *runnerv1.AutopilotThinkingEvent) *eventbus.AutopilotThinkingData {
	result := &eventbus.AutopilotThinkingData{
		AutopilotControllerKey: data.GetAutopilotKey(),
		Iteration:              data.GetIteration(),
		DecisionType:           data.GetDecisionType(),
		Reasoning:              data.GetReasoning(),
		Confidence:             data.GetConfidence(),
	}

	// Convert action
	if action := data.GetAction(); action != nil {
		result.Action = &eventbus.AutopilotActionData{
			Type:    action.GetType(),
			Content: action.GetContent(),
			Reason:  action.GetReason(),
		}
	}

	// Convert progress
	if progress := data.GetProgress(); progress != nil {
		result.Progress = &eventbus.AutopilotProgressData{
			Summary:        progress.GetSummary(),
			CompletedSteps: progress.GetCompletedSteps(),
			RemainingSteps: progress.GetRemainingSteps(),
			Percent:        progress.GetPercent(),
		}
	}

	// Convert help request
	if helpReq := data.GetHelpRequest(); helpReq != nil {
		result.HelpRequest = &eventbus.AutopilotHelpRequestData{
			Reason:          helpReq.GetReason(),
			Context:         helpReq.GetContext(),
			TerminalExcerpt: helpReq.GetTerminalExcerpt(),
		}
		for _, s := range helpReq.GetSuggestions() {
			result.HelpRequest.Suggestions = append(result.HelpRequest.Suggestions, eventbus.AutopilotHelpSuggestionData{
				Action: s.GetAction(),
				Label:  s.GetLabel(),
			})
		}
	}

	return result
}

// setupEventBusHub sets up the integration between EventBus and WebSocket Hub
// EventBus publishes events, Hub routes them to appropriate WebSocket clients
func setupEventBusHub(eb *eventbus.EventBus, hub *websocket.Hub) {
	// Subscribe to entity events (broadcast to organization)
	eb.SubscribeCategory(eventbus.CategoryEntity, func(event *eventbus.Event) {
		data, err := json.Marshal(event)
		if err != nil {
			slog.Error("failed to marshal event for hub", "error", err, "type", event.Type)
			return
		}
		hub.BroadcastToOrg(event.OrganizationID, data)
	})

	// Subscribe to notification events (targeted to specific users)
	eb.SubscribeCategory(eventbus.CategoryNotification, func(event *eventbus.Event) {
		slog.Info("notification event received",
			"type", event.Type,
			"target_user_id", event.TargetUserID,
			"org_id", event.OrganizationID,
		)

		data, err := json.Marshal(event)
		if err != nil {
			slog.Error("failed to marshal notification event", "error", err, "type", event.Type)
			return
		}

		// Send to specific target user
		if event.TargetUserID != nil {
			slog.Info("sending notification to user",
				"user_id", *event.TargetUserID,
				"type", event.Type,
				"user_client_count", hub.GetUserClientCount(*event.TargetUserID),
			)
			hub.SendToUser(*event.TargetUserID, data)
		}

		// Send to multiple target users
		for _, userID := range event.TargetUserIDs {
			hub.SendToUser(userID, data)
		}
	})

	// Subscribe to system events (broadcast to organization)
	eb.SubscribeCategory(eventbus.CategorySystem, func(event *eventbus.Event) {
		data, err := json.Marshal(event)
		if err != nil {
			slog.Error("failed to marshal system event", "error", err, "type", event.Type)
			return
		}
		hub.BroadcastToOrg(event.OrganizationID, data)
	})
}
