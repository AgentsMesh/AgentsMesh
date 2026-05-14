package subscriptionadminconnect

import (
	"context"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	billingv1 "github.com/anthropics/agentsmesh/proto/gen/go/billing/v1"
)

func (s *Server) CreateSubscription(
	ctx context.Context, req *connect.Request[billingv1.CreateAdminSubscriptionRequest],
) (*connect.Response[billingv1.AdminSubscription], error) {
	ctx, adminUser, err := interceptors.ResolveSystemAdmin(ctx, s.db)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrgId()
	months := int(req.Msg.GetMonths())
	if months <= 0 {
		months = 1
	}

	newSub, err := s.billingSvc.AdminCreateSubscription(ctx, orgID, req.Msg.GetPlanName(), months)
	if err != nil {
		return nil, mapServiceError(err)
	}

	seatUsage, _ := s.billingSvc.GetSeatUsage(ctx, orgID)

	logAdminAction(ctx, s.adminSvc, adminUser.ID,
		admin.AuditActionSubUpdate, admin.TargetTypeSubscription, orgID,
		nil, newSub, req.Peer().Addr, req.Header().Get("User-Agent"))

	return connect.NewResponse(toProtoAdminSubscription(newSub, seatUsage)), nil
}

func (s *Server) UpdatePlan(
	ctx context.Context, req *connect.Request[billingv1.UpdateAdminPlanRequest],
) (*connect.Response[billingv1.AdminSubscription], error) {
	ctx, adminUser, err := interceptors.ResolveSystemAdmin(ctx, s.db)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrgId()
	oldSub, _ := s.billingSvc.GetSubscription(ctx, orgID)

	newSub, err := s.billingSvc.AdminUpdatePlan(ctx, orgID, req.Msg.GetPlanName())
	if err != nil {
		return nil, mapServiceError(err)
	}

	seatUsage, _ := s.billingSvc.GetSeatUsage(ctx, orgID)

	logAdminAction(ctx, s.adminSvc, adminUser.ID,
		admin.AuditActionSubUpdate, admin.TargetTypeSubscription, orgID,
		oldSub, newSub, req.Peer().Addr, req.Header().Get("User-Agent"))

	return connect.NewResponse(toProtoAdminSubscription(newSub, seatUsage)), nil
}

func (s *Server) UpdateSeats(
	ctx context.Context, req *connect.Request[billingv1.UpdateAdminSeatsRequest],
) (*connect.Response[billingv1.AdminSubscription], error) {
	ctx, adminUser, err := interceptors.ResolveSystemAdmin(ctx, s.db)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrgId()
	oldSub, _ := s.billingSvc.GetSubscription(ctx, orgID)

	if err := s.billingSvc.AdminSetSeatCount(ctx, orgID, int(req.Msg.GetSeatCount())); err != nil {
		return nil, mapServiceError(err)
	}

	newSub, _ := s.billingSvc.GetSubscription(ctx, orgID)
	seatUsage, _ := s.billingSvc.GetSeatUsage(ctx, orgID)

	logAdminAction(ctx, s.adminSvc, adminUser.ID,
		admin.AuditActionSubUpdate, admin.TargetTypeSubscription, orgID,
		oldSub, newSub, req.Peer().Addr, req.Header().Get("User-Agent"))

	return connect.NewResponse(toProtoAdminSubscription(newSub, seatUsage)), nil
}

func (s *Server) UpdateCycle(
	ctx context.Context, req *connect.Request[billingv1.UpdateAdminCycleRequest],
) (*connect.Response[billingv1.AdminSubscription], error) {
	ctx, adminUser, err := interceptors.ResolveSystemAdmin(ctx, s.db)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrgId()
	oldSub, _ := s.billingSvc.GetSubscription(ctx, orgID)

	if err := s.billingSvc.SetNextBillingCycle(ctx, orgID, req.Msg.GetBillingCycle()); err != nil {
		return nil, mapServiceError(err)
	}

	newSub, _ := s.billingSvc.GetSubscription(ctx, orgID)
	seatUsage, _ := s.billingSvc.GetSeatUsage(ctx, orgID)

	logAdminAction(ctx, s.adminSvc, adminUser.ID,
		admin.AuditActionSubUpdate, admin.TargetTypeSubscription, orgID,
		oldSub, newSub, req.Peer().Addr, req.Header().Get("User-Agent"))

	return connect.NewResponse(toProtoAdminSubscription(newSub, seatUsage)), nil
}
