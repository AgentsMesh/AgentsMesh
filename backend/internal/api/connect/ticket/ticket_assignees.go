package ticketconnect

import (
	"context"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	ticketservice "github.com/anthropics/agentsmesh/backend/internal/service/ticket"
	ticketv1 "github.com/anthropics/agentsmesh/proto/gen/go/ticket/v1"
)

// AddAssignee mirrors REST handler `AddAssignee` (ticket_assignees.go:21).
func (s *Server) AddAssignee(
	ctx context.Context, req *connect.Request[ticketv1.AddAssigneeRequest],
) (*connect.Response[ticketv1.AddAssigneeResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	t, err := s.ticketSvc.GetTicketBySlug(ctx, tenant.OrganizationID, req.Msg.GetTicketSlug())
	if err != nil {
		return nil, mapServiceError(ticketservice.ErrTicketNotFound)
	}

	if err := s.ticketSvc.AddAssignee(ctx, t.ID, req.Msg.GetUserId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&ticketv1.AddAssigneeResponse{}), nil
}

// RemoveAssignee mirrors REST handler `RemoveAssignee` (ticket_assignees.go:48).
func (s *Server) RemoveAssignee(
	ctx context.Context, req *connect.Request[ticketv1.RemoveAssigneeRequest],
) (*connect.Response[ticketv1.RemoveAssigneeResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	t, err := s.ticketSvc.GetTicketBySlug(ctx, tenant.OrganizationID, req.Msg.GetTicketSlug())
	if err != nil {
		return nil, mapServiceError(ticketservice.ErrTicketNotFound)
	}

	if err := s.ticketSvc.RemoveAssignee(ctx, t.ID, req.Msg.GetUserId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&ticketv1.RemoveAssigneeResponse{}), nil
}
