package subscriptionadminconnect

import (
	"context"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	billingv1 "github.com/anthropics/agentsmesh/proto/gen/go/billing/v1"
)

func (s *Server) GetSubscription(
	ctx context.Context, req *connect.Request[billingv1.GetAdminSubscriptionRequest],
) (*connect.Response[billingv1.AdminSubscription], error) {
	ctx, adminUser, err := interceptors.ResolveSystemAdmin(ctx, s.db)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrgId()
	sub, err := s.billingSvc.GetSubscription(ctx, orgID)
	if err != nil {
		return nil, mapServiceError(err)
	}

	seatUsage, _ := s.billingSvc.GetSeatUsage(ctx, orgID)

	logAdminAction(ctx, s.adminSvc, adminUser.ID,
		admin.AuditActionSubView, admin.TargetTypeSubscription, orgID,
		nil, nil, req.Peer().Addr, req.Header().Get("User-Agent"))

	return connect.NewResponse(toProtoAdminSubscription(sub, seatUsage)), nil
}

func (s *Server) ListPlans(
	ctx context.Context, req *connect.Request[billingv1.ListAdminPlansRequest],
) (*connect.Response[billingv1.ListAdminPlansResponse], error) {
	ctx, _, err := interceptors.ResolveSystemAdmin(ctx, s.db)
	if err != nil {
		return nil, err
	}

	plans, err := s.billingSvc.ListPlans(ctx)
	if err != nil {
		return nil, mapServiceError(err)
	}

	items := make([]*billingv1.AdminSubscriptionPlan, 0, len(plans))
	for _, p := range plans {
		items = append(items, toProtoPlan(p))
	}
	return connect.NewResponse(&billingv1.ListAdminPlansResponse{Data: items}), nil
}
