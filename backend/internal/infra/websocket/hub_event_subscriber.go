package websocket

import (
	"encoding/json"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

// HubEventSubscriber routes EventBus events to WebSocket clients.
// Entity/System events → BroadcastToOrg. Notifications bypass EventBus
// and are delivered directly via NotificationRelay (see notification_relay.go).
type HubEventSubscriber struct {
	hub    *Hub
	logger *slog.Logger
}

func NewHubEventSubscriber(hub *Hub, logger *slog.Logger) *HubEventSubscriber {
	if logger == nil {
		logger = slog.Default()
	}
	return &HubEventSubscriber{
		hub:    hub,
		logger: logger.With("component", "hub_event_subscriber"),
	}
}

func (s *HubEventSubscriber) Subscribe(eb *eventbus.EventBus) {
	eb.SubscribeCategory(eventbus.CategoryEntity, s.handleEntityEvent)
	eb.SubscribeCategory(eventbus.CategorySystem, s.handleSystemEvent)
	s.logger.Info("subscribed to EventBus categories (entity, system)")
}

func (s *HubEventSubscriber) handleEntityEvent(event *eventbus.Event) {
	data, err := json.Marshal(event)
	if err != nil {
		s.logger.Error("failed to marshal entity event", "error", err, "type", event.Type)
		return
	}
	s.hub.BroadcastToOrg(event.OrganizationID, data)
}

func (s *HubEventSubscriber) handleSystemEvent(event *eventbus.Event) {
	data, err := json.Marshal(event)
	if err != nil {
		s.logger.Error("failed to marshal system event", "error", err, "type", event.Type)
		return
	}
	s.hub.BroadcastToOrg(event.OrganizationID, data)
}
