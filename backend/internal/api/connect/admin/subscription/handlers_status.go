package subscriptionadminconnect

import (
	"context"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	billingv1 "github.com/anthropics/agentsmesh/proto/gen/go/billing/v1"
)

func (s *Server) Freeze(
	ctx context.Context, req *connect.Request[billingv1.FreezeAdminSubscriptionRequest],
) (*connect.Response[billingv1.AdminSubscription], error) {
	ctx, adminUser, err := interceptors.ResolveSystemAdmin(ctx, s.db)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrgId()
	oldSub, _ := s.billingSvc.GetSubscription(ctx, orgID)

	if err := s.billingSvc.FreezeSubscription(ctx, orgID); err != nil {
		return nil, mapServiceError(err)
	}

	_ = s.adminSvc.UpdateOrganizationSubscriptionStatus(ctx, orgID, billing.SubscriptionStatusFrozen)

	newSub, _ := s.billingSvc.GetSubscription(ctx, orgID)
	seatUsage, _ := s.billingSvc.GetSeatUsage(ctx, orgID)

	logAdminAction(ctx, s.adminSvc, adminUser.ID,
		admin.AuditActionSubFreeze, admin.TargetTypeSubscription, orgID,
		oldSub, newSub, req.Peer().Addr, req.Header().Get("User-Agent"))

	return connect.NewResponse(toProtoAdminSubscription(newSub, seatUsage)), nil
}

func (s *Server) Unfreeze(
	ctx context.Context, req *connect.Request[billingv1.UnfreezeAdminSubscriptionRequest],
) (*connect.Response[billingv1.AdminSubscription], error) {
	ctx, adminUser, err := interceptors.ResolveSystemAdmin(ctx, s.db)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrgId()
	oldSub, _ := s.billingSvc.GetSubscription(ctx, orgID)

	cycle := billing.BillingCycleMonthly
	if oldSub != nil && oldSub.BillingCycle != "" {
		cycle = oldSub.BillingCycle
	}

	if err := s.billingSvc.UnfreezeSubscription(ctx, orgID, cycle); err != nil {
		return nil, mapServiceError(err)
	}

	_ = s.adminSvc.UpdateOrganizationSubscriptionStatus(ctx, orgID, billing.SubscriptionStatusActive)

	newSub, _ := s.billingSvc.GetSubscription(ctx, orgID)
	seatUsage, _ := s.billingSvc.GetSeatUsage(ctx, orgID)

	logAdminAction(ctx, s.adminSvc, adminUser.ID,
		admin.AuditActionSubUnfreeze, admin.TargetTypeSubscription, orgID,
		oldSub, newSub, req.Peer().Addr, req.Header().Get("User-Agent"))

	return connect.NewResponse(toProtoAdminSubscription(newSub, seatUsage)), nil
}

func (s *Server) Cancel(
	ctx context.Context, req *connect.Request[billingv1.CancelAdminSubscriptionRequest],
) (*connect.Response[billingv1.AdminSubscription], error) {
	ctx, adminUser, err := interceptors.ResolveSystemAdmin(ctx, s.db)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrgId()
	oldSub, _ := s.billingSvc.GetSubscription(ctx, orgID)

	if err := s.billingSvc.AdminCancelSubscription(ctx, orgID); err != nil {
		return nil, mapServiceError(err)
	}

	newSub, _ := s.billingSvc.GetSubscription(ctx, orgID)
	seatUsage, _ := s.billingSvc.GetSeatUsage(ctx, orgID)

	logAdminAction(ctx, s.adminSvc, adminUser.ID,
		admin.AuditActionSubCancel, admin.TargetTypeSubscription, orgID,
		oldSub, newSub, req.Peer().Addr, req.Header().Get("User-Agent"))

	return connect.NewResponse(toProtoAdminSubscription(newSub, seatUsage)), nil
}

func (s *Server) Renew(
	ctx context.Context, req *connect.Request[billingv1.RenewAdminSubscriptionRequest],
) (*connect.Response[billingv1.AdminSubscription], error) {
	ctx, adminUser, err := interceptors.ResolveSystemAdmin(ctx, s.db)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrgId()
	months := int(req.Msg.GetMonths())
	oldSub, _ := s.billingSvc.GetSubscription(ctx, orgID)

	newSub, err := s.billingSvc.AdminRenew(ctx, orgID, months)
	if err != nil {
		return nil, mapServiceError(err)
	}

	seatUsage, _ := s.billingSvc.GetSeatUsage(ctx, orgID)

	logAdminAction(ctx, s.adminSvc, adminUser.ID,
		admin.AuditActionSubRenew, admin.TargetTypeSubscription, orgID,
		oldSub, newSub, req.Peer().Addr, req.Header().Get("User-Agent"))

	return connect.NewResponse(toProtoAdminSubscription(newSub, seatUsage)), nil
}

func (s *Server) SetAutoRenew(
	ctx context.Context, req *connect.Request[billingv1.SetAdminAutoRenewRequest],
) (*connect.Response[billingv1.AdminSubscription], error) {
	ctx, adminUser, err := interceptors.ResolveSystemAdmin(ctx, s.db)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrgId()
	oldSub, _ := s.billingSvc.GetSubscription(ctx, orgID)

	if err := s.billingSvc.SetAutoRenew(ctx, orgID, req.Msg.GetAutoRenew()); err != nil {
		return nil, mapServiceError(err)
	}

	newSub, _ := s.billingSvc.GetSubscription(ctx, orgID)
	seatUsage, _ := s.billingSvc.GetSeatUsage(ctx, orgID)

	logAdminAction(ctx, s.adminSvc, adminUser.ID,
		admin.AuditActionSubUpdate, admin.TargetTypeSubscription, orgID,
		oldSub, newSub, req.Peer().Addr, req.Header().Get("User-Agent"))

	return connect.NewResponse(toProtoAdminSubscription(newSub, seatUsage)), nil
}

func (s *Server) SetCustomQuota(
	ctx context.Context, req *connect.Request[billingv1.SetAdminCustomQuotaRequest],
) (*connect.Response[billingv1.AdminSubscription], error) {
	ctx, adminUser, err := interceptors.ResolveSystemAdmin(ctx, s.db)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrgId()
	oldSub, _ := s.billingSvc.GetSubscription(ctx, orgID)

	if err := s.billingSvc.SetCustomQuota(ctx, orgID, req.Msg.GetResource(), int(req.Msg.GetLimit())); err != nil {
		return nil, mapServiceError(err)
	}

	newSub, _ := s.billingSvc.GetSubscription(ctx, orgID)
	seatUsage, _ := s.billingSvc.GetSeatUsage(ctx, orgID)

	logAdminAction(ctx, s.adminSvc, adminUser.ID,
		admin.AuditActionSubQuota, admin.TargetTypeSubscription, orgID,
		oldSub, newSub, req.Peer().Addr, req.Header().Get("User-Agent"))

	return connect.NewResponse(toProtoAdminSubscription(newSub, seatUsage)), nil
}
