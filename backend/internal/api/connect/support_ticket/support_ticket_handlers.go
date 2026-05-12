package supportticketconnect

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"

	supportticketsvc "github.com/anthropics/agentsmesh/backend/internal/service/supportticket"
	supportticketv1 "github.com/anthropics/agentsmesh/proto/gen/go/support_ticket/v1"
)

// ListSupportTickets returns the authenticated user's tickets, mirroring
// REST GET /api/v1/support-tickets (support_tickets.go:108). The wire
// envelope unifies to {items, total, limit, offset} (conventions §8);
// REST's `{data, total, page, page_size, total_pages}` shape lives only
// in the TS adapter from here on.
func (s *Server) ListSupportTickets(
	ctx context.Context, req *connect.Request[supportticketv1.ListSupportTicketsRequest],
) (*connect.Response[supportticketv1.ListSupportTicketsResponse], error) {
	userID, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	offset, limit := normalizeListArgs(req.Msg.GetOffset(), req.Msg.GetLimit())
	page := (offset / limit) + 1

	result, err := s.svc.ListByUser(ctx, userID, &supportticketsvc.ListQuery{
		Status:   req.Msg.GetStatus(),
		Page:     int(page),
		PageSize: int(limit),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	items := make([]*supportticketv1.SupportTicket, 0, len(result.Data))
	for i := range result.Data {
		items = append(items, toProtoTicket(&result.Data[i]))
	}
	return connect.NewResponse(&supportticketv1.ListSupportTicketsResponse{
		Items:  items,
		Total:  result.Total,
		Limit:  limit,
		Offset: offset,
	}), nil
}

// GetSupportTicket returns a ticket with its messages, mirroring REST GET
// /api/v1/support-tickets/:id (support_tickets.go:139). Ownership is
// enforced by the service layer (GetByID returns ErrTicketNotFound for
// non-owner reads). Messages load best-effort — a load error is logged
// and an empty list is returned, matching REST behavior.
func (s *Server) GetSupportTicket(
	ctx context.Context, req *connect.Request[supportticketv1.GetSupportTicketRequest],
) (*connect.Response[supportticketv1.SupportTicketDetail], error) {
	userID, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if req.Msg.GetId() == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("ticket id is required"))
	}

	ticket, err := s.svc.GetByID(ctx, req.Msg.GetId(), userID)
	if err != nil {
		return nil, mapSupportTicketError(err)
	}

	messages, err := s.svc.ListMessages(ctx, ticket.ID, userID)
	if err != nil {
		slog.WarnContext(ctx, "failed to load messages for ticket",
			"ticket_id", ticket.ID, "error", err)
		messages = nil
	}

	protoMessages := make([]*supportticketv1.SupportTicketMessage, 0, len(messages))
	for i := range messages {
		protoMessages = append(protoMessages, toProtoMessage(&messages[i]))
	}
	return connect.NewResponse(&supportticketv1.SupportTicketDetail{
		Ticket:   toProtoTicket(ticket),
		Messages: protoMessages,
	}), nil
}

// GetAttachmentURL returns a presigned download URL, mirroring REST GET
// /api/v1/support-tickets/attachments/:attachmentId/url
// (support_tickets.go:267). Ownership is enforced by the service layer
// (ticket.UserID == caller).
func (s *Server) GetAttachmentURL(
	ctx context.Context, req *connect.Request[supportticketv1.GetAttachmentUrlRequest],
) (*connect.Response[supportticketv1.GetAttachmentUrlResponse], error) {
	userID, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if req.Msg.GetAttachmentId() == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("attachment_id is required"))
	}

	url, err := s.svc.GetAttachmentURL(ctx, req.Msg.GetAttachmentId(), userID)
	if err != nil {
		return nil, mapSupportTicketError(err)
	}
	return connect.NewResponse(&supportticketv1.GetAttachmentUrlResponse{
		Url: url,
	}), nil
}
