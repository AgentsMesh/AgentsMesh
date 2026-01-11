package ticket

import "context"

// TicketEventType defines the type of ticket event (type-safe)
type TicketEventType int

const (
	TicketEventCreated TicketEventType = iota
	TicketEventUpdated
	TicketEventStatusChanged
	TicketEventMoved
	TicketEventDeleted
)

// EventPublisher defines the interface for publishing events (dependency inversion)
type EventPublisher interface {
	PublishTicketEvent(ctx context.Context, eventType TicketEventType, orgID int64, identifier, status, previousStatus string)
}
