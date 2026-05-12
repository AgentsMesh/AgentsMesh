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

const defaultListLimit = 20

// ListTickets mirrors REST handler `ListTickets` (tickets.go:75). Returns
// the uniform {items, total, limit, offset} envelope per conventions §8.
func (s *Server) ListTickets(
	ctx context.Context, req *connect.Request[ticketv1.ListTicketsRequest],
) (*connect.Response[ticketv1.ListTicketsResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)

	limit := int(req.Msg.GetLimit())
	if limit == 0 {
		limit = defaultListLimit
	}
	offset := int(req.Msg.GetOffset())

	filter := &domainticket.TicketListFilter{
		OrganizationID: tenant.OrganizationID,
		Status:         req.Msg.GetStatus(),
		Priority:       req.Msg.GetPriority(),
		Query:          req.Msg.GetQuery(),
		UserRole:       tenant.UserRole,
		Limit:          limit,
		Offset:         offset,
	}
	if req.Msg.RepositoryId != nil {
		v := req.Msg.GetRepositoryId()
		filter.RepositoryID = &v
	}
	if req.Msg.AssigneeId != nil {
		v := req.Msg.GetAssigneeId()
		filter.AssigneeID = &v
	}

	tickets, total, err := s.ticketSvc.ListTickets(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&ticketv1.ListTicketsResponse{
		Items:  toProtoTickets(tickets),
		Total:  total,
		Limit:  int32(limit),
		Offset: int32(offset),
	}), nil
}

// GetTicket mirrors REST handler `GetTicket` (tickets.go:164).
func (s *Server) GetTicket(
	ctx context.Context, req *connect.Request[ticketv1.GetTicketRequest],
) (*connect.Response[ticketv1.Ticket], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	t, err := s.ticketSvc.GetTicketBySlug(ctx, tenant.OrganizationID, req.Msg.GetTicketSlug())
	if err != nil {
		return nil, mapServiceError(ticketservice.ErrTicketNotFound)
	}
	return connect.NewResponse(toProtoTicket(t)), nil
}

// CreateTicket mirrors REST handler `CreateTicket` (tickets.go:115).
func (s *Server) CreateTicket(
	ctx context.Context, req *connect.Request[ticketv1.CreateTicketRequest],
) (*connect.Response[ticketv1.Ticket], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)

	var content *string
	if v := req.Msg.GetContent(); v != "" {
		content = &v
	}

	var parentTicketID *int64
	if v := req.Msg.GetParentTicketSlug(); v != "" {
		parent, perr := s.ticketSvc.GetTicketByIDOrSlug(ctx, tenant.OrganizationID, v)
		if perr != nil {
			return nil, connect.NewError(connect.CodeNotFound, perr)
		}
		parentTicketID = &parent.ID
	}

	create := &ticketservice.CreateTicketRequest{
		OrganizationID: tenant.OrganizationID,
		ReporterID:     tenant.UserID,
		Title:          req.Msg.GetTitle(),
		Content:        content,
		Status:         req.Msg.GetStatus(),
		Priority:       req.Msg.GetPriority(),
		AssigneeIDs:    req.Msg.GetAssigneeIds(),
		Labels:         req.Msg.GetLabels(),
		ParentTicketID: parentTicketID,
	}
	if req.Msg.RepositoryId != nil {
		v := req.Msg.GetRepositoryId()
		create.RepositoryID = &v
	}

	t, err := s.ticketSvc.CreateTicket(ctx, create)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoTicket(t)), nil
}

// DeleteTicket mirrors REST handler `DeleteTicket` (tickets.go:245).
func (s *Server) DeleteTicket(
	ctx context.Context, req *connect.Request[ticketv1.DeleteTicketRequest],
) (*connect.Response[ticketv1.DeleteTicketResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	t, err := s.ticketSvc.GetTicketBySlug(ctx, tenant.OrganizationID, req.Msg.GetTicketSlug())
	if err != nil {
		return nil, mapServiceError(ticketservice.ErrTicketNotFound)
	}

	if err := s.ticketSvc.DeleteTicket(ctx, t.ID); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&ticketv1.DeleteTicketResponse{}), nil
}
