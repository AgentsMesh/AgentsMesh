package ticketconnect

import (
	"context"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	domainticket "github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	ticketservice "github.com/anthropics/agentsmesh/backend/internal/service/ticket"
	ticketv1 "github.com/anthropics/agentsmesh/proto/gen/go/ticket/v1"
)

const (
	defaultActiveLimit = 20
	defaultBoardLimit  = 50
	maxBoardLimit      = 200
)

// GetActiveTickets mirrors REST handler `GetActiveTickets` (ticket_board.go:17).
func (s *Server) GetActiveTickets(
	ctx context.Context, req *connect.Request[ticketv1.GetActiveTicketsRequest],
) (*connect.Response[ticketv1.ListTicketsResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = defaultActiveLimit
	}

	var repoID *int64
	if req.Msg.RepositoryId != nil {
		v := req.Msg.GetRepositoryId()
		repoID = &v
	}

	tickets, err := s.ticketSvc.GetActiveTickets(ctx, tenant.OrganizationID, repoID, limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&ticketv1.ListTicketsResponse{
		Items:  toProtoTickets(tickets),
		Total:  int64(len(tickets)),
		Limit:  int32(limit),
		Offset: 0,
	}), nil
}

// GetBoard mirrors REST handler `GetBoard` (ticket_board.go:44).
func (s *Server) GetBoard(
	ctx context.Context, req *connect.Request[ticketv1.GetBoardRequest],
) (*connect.Response[ticketv1.Board], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	filter := &domainticket.TicketListFilter{
		OrganizationID: tenant.OrganizationID,
		UserRole:       tenant.UserRole,
		Limit:          defaultBoardLimit,
	}
	if req.Msg.RepositoryId != nil {
		v := req.Msg.GetRepositoryId()
		filter.RepositoryID = &v
	}
	if l := int(req.Msg.GetLimit()); l > 0 {
		filter.Limit = l
		if filter.Limit > maxBoardLimit {
			filter.Limit = maxBoardLimit
		}
	}
	if v := req.Msg.GetPriority(); v != "" {
		filter.Priority = v
	}
	if req.Msg.AssigneeId != nil {
		v := req.Msg.GetAssigneeId()
		filter.AssigneeID = &v
	}
	if v := req.Msg.GetQuery(); v != "" {
		filter.Query = v
	}

	board, err := s.ticketSvc.GetBoard(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(toProtoBoard(board)), nil
}

// GetSubTickets mirrors REST handler `GetSubTickets` (ticket_board.go:90).
func (s *Server) GetSubTickets(
	ctx context.Context, req *connect.Request[ticketv1.GetSubTicketsRequest],
) (*connect.Response[ticketv1.ListTicketsResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	t, err := s.ticketSvc.GetTicketBySlug(ctx, tenant.OrganizationID, req.Msg.GetTicketSlug())
	if err != nil {
		return nil, mapServiceError(ticketservice.ErrTicketNotFound)
	}

	subs, err := s.ticketSvc.GetChildTickets(ctx, t.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&ticketv1.ListTicketsResponse{
		Items:  toProtoTickets(subs),
		Total:  int64(len(subs)),
		Limit:  int32(len(subs)),
		Offset: 0,
	}), nil
}
