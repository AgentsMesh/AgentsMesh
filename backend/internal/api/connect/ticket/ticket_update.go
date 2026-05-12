package ticketconnect

import (
	"context"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	ticketservice "github.com/anthropics/agentsmesh/backend/internal/service/ticket"
	ticketv1 "github.com/anthropics/agentsmesh/proto/gen/go/ticket/v1"
)

// UpdateTicket mirrors REST handler `UpdateTicket` (tickets.go:179).
// proto3 optional semantics: only fields whose Has*() returns true land
// in the update map.
func (s *Server) UpdateTicket(
	ctx context.Context, req *connect.Request[ticketv1.UpdateTicketRequest],
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

	updates := buildUpdateMap(req.Msg)
	t, err = s.ticketSvc.UpdateTicket(ctx, t.ID, updates)
	if err != nil {
		return nil, mapServiceError(err)
	}

	if req.Msg.AssigneeIds != nil {
		if err := s.ticketSvc.UpdateAssignees(ctx, t.ID, req.Msg.GetAssigneeIds()); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	return connect.NewResponse(toProtoTicket(t)), nil
}

// buildUpdateMap honors PATCH semantics: optional fields whose presence
// is set translate to map entries. repository_id=0 explicitly clears the
// association (tickets.go:209). due_date="" explicitly clears (tickets.go:217).
func buildUpdateMap(req *ticketv1.UpdateTicketRequest) map[string]interface{} {
	updates := make(map[string]interface{})
	if v := req.GetTitle(); v != "" {
		updates["title"] = v
	}
	if req.Content != nil {
		updates["content"] = req.GetContent()
	}
	if v := req.GetStatus(); v != "" {
		updates["status"] = v
	}
	if v := req.GetPriority(); v != "" {
		updates["priority"] = v
	}
	if req.RepositoryId != nil {
		if v := req.GetRepositoryId(); v == 0 {
			updates["repository_id"] = nil
		} else {
			updates["repository_id"] = v
		}
	}
	if req.DueDate != nil {
		if v := req.GetDueDate(); v == "" {
			updates["due_date"] = nil
		} else {
			updates["due_date"] = v
		}
	}
	return updates
}

// UpdateTicketStatus mirrors REST handler `UpdateTicketStatus` (tickets.go:267).
func (s *Server) UpdateTicketStatus(
	ctx context.Context, req *connect.Request[ticketv1.UpdateTicketStatusRequest],
) (*connect.Response[ticketv1.UpdateTicketStatusResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	t, err := s.ticketSvc.GetTicketBySlug(ctx, tenant.OrganizationID, req.Msg.GetTicketSlug())
	if err != nil {
		return nil, mapServiceError(ticketservice.ErrTicketNotFound)
	}

	if err := s.ticketSvc.UpdateStatus(ctx, t.ID, req.Msg.GetStatus()); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&ticketv1.UpdateTicketStatusResponse{}), nil
}
