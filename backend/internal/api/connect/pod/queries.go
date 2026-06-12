package podconnect

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	poddom "github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/grant"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/policy"
	podv1 "github.com/anthropics/agentsmesh/proto/gen/go/pod/v1"
)

// ListPods — REST analogue: GET /api/v1/organizations/:slug/pods.
// Members can only list their own pods; admin/owner see everything in the org.
// Mirrors PodPolicy.ListFilter semantics from v1/pod_query.go.
func (s *Server) ListPods(
	ctx context.Context, req *connect.Request[podv1.ListPodsRequest],
) (*connect.Response[podv1.ListPodsResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)

	limit := int(req.Msg.GetLimit())
	if limit == 0 {
		limit = 20
	}
	offset := int(req.Msg.GetOffset())

	var statuses []string
	if status := req.Msg.GetStatus(); status != "" {
		statuses = strings.Split(status, ",")
	}

	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)
	filter := policy.PodPolicy.ListFilter(sub)
	createdByID := req.Msg.GetCreatedById()
	if filter.OwnerOnly > 0 {
		createdByID = filter.OwnerOnly
	}

	pods, total, err := s.podSvc.ListPods(ctx, tenant.OrganizationID, poddom.PodListQuery{
		Statuses:      statuses,
		CreatedByID:   createdByID,
		GrantedUserID: filter.GrantUserID,
		RunnerID:      req.Msg.GetRunnerId(),
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		return nil, mapServiceError(err)
	}

	items := toProtoPods(pods)
	s.applyResumedBy(ctx, items)

	return connect.NewResponse(&podv1.ListPodsResponse{
		Items:  items,
		Total:  total,
		Limit:  int32(limit),
		Offset: int32(offset),
	}), nil
}

// applyResumedBy fills the query-derived resumed_by_pod_key on terminated /
// completed / orphaned items — the only statuses resume accepts as a source.
// Failure is non-fatal: the field is an affordance hint, not list data.
func (s *Server) applyResumedBy(ctx context.Context, items []*podv1.Pod) {
	keys := make([]string, 0, len(items))
	for _, p := range items {
		switch p.GetStatus() {
		case poddom.StatusTerminated, poddom.StatusCompleted, poddom.StatusOrphaned:
			keys = append(keys, p.GetPodKey())
		}
	}
	if len(keys) == 0 {
		return
	}
	resumedBy, err := s.podSvc.ListActiveResumedBy(ctx, keys)
	if err != nil {
		slog.WarnContext(ctx, "resumed-by lookup failed", "error", err)
		return
	}
	for _, p := range items {
		if key, ok := resumedBy[p.GetPodKey()]; ok {
			p.ResumedByPodKey = &key
		}
	}
}

// GetPod — REST analogue: GET /api/v1/organizations/:slug/pods/:key.
func (s *Server) GetPod(
	ctx context.Context, req *connect.Request[podv1.GetPodRequest],
) (*connect.Response[podv1.Pod], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	podKey := req.Msg.GetPodKey()

	pod, err := s.podSvc.GetPod(ctx, podKey)
	if err != nil {
		return nil, mapServiceError(err)
	}

	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)
	if !policy.PodPolicy.AllowRead(sub, s.podResourceWithGrants(ctx, podKey, pod.OrganizationID, pod.CreatedByID)) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("forbidden"))
	}

	return connect.NewResponse(ToProtoPod(pod)), nil
}

// ListPodsByTicket — REST analogue: GET /api/v1/organizations/:slug/tickets/:id/pods.
// Filters by pod visibility, respecting per-resource grants.
func (s *Server) ListPodsByTicket(
	ctx context.Context, req *connect.Request[podv1.ListPodsByTicketRequest],
) (*connect.Response[podv1.ListPodsByTicketResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	ticketID := req.Msg.GetTicketId()

	pods, err := s.podSvc.GetPodsByTicket(ctx, ticketID)
	if err != nil {
		return nil, mapServiceError(err)
	}

	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)
	filter := policy.PodPolicy.ListFilter(sub)
	if filter.OwnerOnly > 0 {
		var grantedKeys map[string]bool
		if s.grantSvc != nil && filter.GrantUserID > 0 {
			if ids, err := s.grantSvc.GetGrantedResourceIDs(ctx, grant.TypePod, filter.GrantUserID, tenant.OrganizationID); err == nil && len(ids) > 0 {
				grantedKeys = make(map[string]bool, len(ids))
				for _, id := range ids {
					grantedKeys[id] = true
				}
			}
		}
		filtered := make([]*poddom.Pod, 0, len(pods))
		for _, p := range pods {
			if p.CreatedByID == filter.OwnerOnly || grantedKeys[p.PodKey] {
				filtered = append(filtered, p)
			}
		}
		pods = filtered
	}

	return connect.NewResponse(&podv1.ListPodsByTicketResponse{
		Items: toProtoPods(pods),
		Total: int64(len(pods)),
	}), nil
}
