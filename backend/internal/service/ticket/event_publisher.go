package ticket

import (
	"context"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

// EventBusPublisher implements EventPublisher using EventBus
type EventBusPublisher struct {
	eventBus *eventbus.EventBus
	logger   *slog.Logger
}

// NewEventBusPublisher creates a new EventBusPublisher
func NewEventBusPublisher(eventBus *eventbus.EventBus, logger *slog.Logger) *EventBusPublisher {
	if logger == nil {
		logger = slog.Default()
	}
	return &EventBusPublisher{
		eventBus: eventBus,
		logger:   logger.With("component", "ticket_event_publisher"),
	}
}

// PublishTicketEvent publishes a ticket event
func (p *EventBusPublisher) PublishTicketEvent(ctx context.Context, eventType TicketEventType, orgID int64, identifier, status, previousStatus string) {
	if p.eventBus == nil {
		return
	}

	// Map TicketEventType to eventbus.EventType (compile-time type safety)
	var et eventbus.EventType
	switch eventType {
	case TicketEventCreated:
		et = eventbus.EventTicketCreated
	case TicketEventUpdated:
		et = eventbus.EventTicketUpdated
	case TicketEventStatusChanged:
		et = eventbus.EventTicketStatusChanged
	case TicketEventMoved:
		et = eventbus.EventTicketMoved
	case TicketEventDeleted:
		et = eventbus.EventTicketDeleted
	default:
		p.logger.Warn("unknown ticket event type", "type", eventType)
		return
	}

	data := &eventbus.TicketStatusChangedData{
		Identifier:     identifier,
		Status:         status,
		PreviousStatus: previousStatus,
	}

	event, err := eventbus.NewEntityEvent(et, orgID, "ticket", identifier, data)
	if err != nil {
		p.logger.Error("failed to create ticket event",
			"error", err,
			"type", eventType,
			"identifier", identifier,
		)
		return
	}

	if err := p.eventBus.Publish(ctx, event); err != nil {
		p.logger.Error("failed to publish ticket event",
			"error", err,
			"type", eventType,
			"identifier", identifier,
		)
	}
}
