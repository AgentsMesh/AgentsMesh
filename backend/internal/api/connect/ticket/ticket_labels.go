package ticketconnect

import (
	"context"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	ticketservice "github.com/anthropics/agentsmesh/backend/internal/service/ticket"
	ticketv1 "github.com/anthropics/agentsmesh/proto/gen/go/ticket/v1"
)

// ListLabels mirrors REST handler `ListLabels` (ticket_labels.go:23). Returns
// the uniform {items, total, limit, offset} envelope per conventions §8.
// Labels currently have no pagination on the service side, so limit/offset
// echo back the input (or 0/total respectively).
func (s *Server) ListLabels(
	ctx context.Context, req *connect.Request[ticketv1.ListLabelsRequest],
) (*connect.Response[ticketv1.ListLabelsResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	var repoID *int64
	if req.Msg.RepositoryId != nil {
		v := req.Msg.GetRepositoryId()
		repoID = &v
	}

	labels, err := s.ticketSvc.ListLabels(ctx, tenant.OrganizationID, repoID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := toProtoLabels(labels)
	return connect.NewResponse(&ticketv1.ListLabelsResponse{
		Items:  items,
		Total:  int64(len(items)),
		Limit:  int32(len(items)),
		Offset: 0,
	}), nil
}

// CreateLabel mirrors REST handler `CreateLabel` (ticket_labels.go:47).
func (s *Server) CreateLabel(
	ctx context.Context, req *connect.Request[ticketv1.CreateLabelRequest],
) (*connect.Response[ticketv1.Label], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	var repoID *int64
	if req.Msg.RepositoryId != nil {
		v := req.Msg.GetRepositoryId()
		repoID = &v
	}

	label, err := s.ticketSvc.CreateLabel(
		ctx, tenant.OrganizationID, repoID,
		req.Msg.GetName(), req.Msg.GetColor(),
	)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoLabel(label)), nil
}

// UpdateLabel mirrors REST handler `UpdateLabel` (ticket_labels.go:73).
func (s *Server) UpdateLabel(
	ctx context.Context, req *connect.Request[ticketv1.UpdateLabelRequest],
) (*connect.Response[ticketv1.Label], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	updates := make(map[string]interface{})
	if v := req.Msg.GetName(); v != "" {
		updates["name"] = v
	}
	if v := req.Msg.GetColor(); v != "" {
		updates["color"] = v
	}

	label, err := s.ticketSvc.UpdateLabel(
		ctx, tenant.OrganizationID, req.Msg.GetId(), updates,
	)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoLabel(label)), nil
}

// DeleteLabel mirrors REST handler `DeleteLabel` (ticket_labels.go:107).
func (s *Server) DeleteLabel(
	ctx context.Context, req *connect.Request[ticketv1.DeleteLabelRequest],
) (*connect.Response[ticketv1.DeleteLabelResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	if err := s.ticketSvc.DeleteLabel(ctx, tenant.OrganizationID, req.Msg.GetId()); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&ticketv1.DeleteLabelResponse{}), nil
}

// AddLabel mirrors REST handler `AddLabel` (ticket_labels.go:131).
func (s *Server) AddLabel(
	ctx context.Context, req *connect.Request[ticketv1.AddLabelRequest],
) (*connect.Response[ticketv1.AddLabelResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	t, err := s.ticketSvc.GetTicketBySlug(ctx, tenant.OrganizationID, req.Msg.GetTicketSlug())
	if err != nil {
		return nil, mapServiceError(ticketservice.ErrTicketNotFound)
	}

	if err := s.ticketSvc.AddLabel(ctx, t.ID, req.Msg.GetLabelId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&ticketv1.AddLabelResponse{}), nil
}

// RemoveLabel mirrors REST handler `RemoveLabel` (ticket_labels.go:158).
func (s *Server) RemoveLabel(
	ctx context.Context, req *connect.Request[ticketv1.RemoveLabelRequest],
) (*connect.Response[ticketv1.RemoveLabelResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	t, err := s.ticketSvc.GetTicketBySlug(ctx, tenant.OrganizationID, req.Msg.GetTicketSlug())
	if err != nil {
		return nil, mapServiceError(ticketservice.ErrTicketNotFound)
	}

	if err := s.ticketSvc.RemoveLabel(ctx, t.ID, req.Msg.GetLabelId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&ticketv1.RemoveLabelResponse{}), nil
}
