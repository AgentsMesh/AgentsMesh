package ticketconnect

import (
	"time"

	domainticket "github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	ticketv1 "github.com/anthropics/agentsmesh/proto/gen/go/ticket/v1"
)

// toProtoTicket converts the GORM-backed domain model into the protobuf
// wire shape. Fields kept in lockstep with the .proto definition. Time
// pointers are RFC 3339 strings; optional pointer fields are passed
// through when set.
func toProtoTicket(t *domainticket.Ticket) *ticketv1.Ticket {
	if t == nil {
		return nil
	}
	out := &ticketv1.Ticket{
		Id:        t.ID,
		Number:    int32(t.Number),
		Slug:      t.Slug,
		Title:     t.Title,
		Status:    t.Status,
		Priority:  t.Priority,
		CreatedAt: t.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: t.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if t.Content != nil {
		out.Content = t.Content
	}
	if t.Severity != nil {
		out.Severity = t.Severity
	}
	if t.Estimate != nil {
		e := int32(*t.Estimate)
		out.Estimate = &e
	}
	if t.DueDate != nil {
		s := t.DueDate.UTC().Format(time.RFC3339)
		out.DueDate = &s
	}
	if t.StartedAt != nil {
		s := t.StartedAt.UTC().Format(time.RFC3339)
		out.StartedAt = &s
	}
	if t.CompletedAt != nil {
		s := t.CompletedAt.UTC().Format(time.RFC3339)
		out.CompletedAt = &s
	}
	if t.RepositoryID != nil {
		out.RepositoryId = t.RepositoryID
	}
	if t.ParentTicketID != nil {
		out.ParentTicketId = t.ParentTicketID
	}
	reporterID := t.ReporterID
	out.ReporterId = &reporterID
	return out
}

func toProtoTickets(tickets []*domainticket.Ticket) []*ticketv1.Ticket {
	out := make([]*ticketv1.Ticket, 0, len(tickets))
	for _, t := range tickets {
		out = append(out, toProtoTicket(t))
	}
	return out
}

func toProtoLabel(l *domainticket.Label) *ticketv1.Label {
	if l == nil {
		return nil
	}
	out := &ticketv1.Label{
		Id:             l.ID,
		OrganizationId: l.OrganizationID,
		Name:           l.Name,
		Color:          l.Color,
	}
	if l.RepositoryID != nil {
		out.RepositoryId = l.RepositoryID
	}
	return out
}

func toProtoLabels(labels []*domainticket.Label) []*ticketv1.Label {
	out := make([]*ticketv1.Label, 0, len(labels))
	for _, l := range labels {
		out = append(out, toProtoLabel(l))
	}
	return out
}

func toProtoBoardColumn(c domainticket.BoardColumn) *ticketv1.BoardColumn {
	tickets := make([]*ticketv1.Ticket, 0, len(c.Tickets))
	for i := range c.Tickets {
		tickets = append(tickets, toProtoTicket(&c.Tickets[i]))
	}
	return &ticketv1.BoardColumn{
		Status:     c.Status,
		Tickets:    tickets,
		TotalCount: int64(c.Count),
	}
}

func toProtoBoard(b *domainticket.Board) *ticketv1.Board {
	if b == nil {
		return &ticketv1.Board{}
	}
	cols := make([]*ticketv1.BoardColumn, 0, len(b.Columns))
	for _, col := range b.Columns {
		cols = append(cols, toProtoBoardColumn(col))
	}
	return &ticketv1.Board{Columns: cols}
}
