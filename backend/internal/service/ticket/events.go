package ticket

import "context"

type TicketEventType int

const (
	TicketEventCreated TicketEventType = iota
	TicketEventUpdated
	TicketEventStatusChanged
	TicketEventMoved
	TicketEventDeleted
)

type EventPublisher interface {
	PublishTicketEvent(ctx context.Context, eventType TicketEventType, orgID int64, slug, status, previousStatus string)
}
