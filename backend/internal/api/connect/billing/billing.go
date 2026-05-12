// Package billingconnect hosts Connect-RPC handlers for the billing domain.
// Mirrors backend/internal/api/rest/v1/billing_*.go but exposes the data
// plane via Connect (binary protobuf wire, see conventions.md §2.5). REST
// stays mounted in parallel; the migration runs dual-track until all 26
// services have flipped.
//
// Two services in this package:
//   * BillingService — org-scoped, auth-required (ResolveOrgScope).
//   * BillingPublicService — no auth, no org_slug (PR #334 fix for the
//     landing-page pricing card).
//
// Handler shape follows runbook §3 + conventions §3.5: every authenticated
// RPC calls ResolveOrgScope first; ListInvoices/ListPlans follow the
// {items,total,limit,offset} envelope; errors map to Connect codes
// (conventions §10).
package billingconnect

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	billingdomain "github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	billingsvc "github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
	billingv1 "github.com/anthropics/agentsmesh/proto/gen/go/billing/v1"
)

const ServiceName = "proto.billing.v1.BillingService"

const (
	GetOverviewProcedure                  = "/" + ServiceName + "/GetOverview"
	ListPlansProcedure                    = "/" + ServiceName + "/ListPlans"
	GetSubscriptionProcedure              = "/" + ServiceName + "/GetSubscription"
	CreateSubscriptionProcedure           = "/" + ServiceName + "/CreateSubscription"
	UpdateSubscriptionProcedure           = "/" + ServiceName + "/UpdateSubscription"
	CancelSubscriptionProcedure           = "/" + ServiceName + "/CancelSubscription"
	RequestCancelSubscriptionProcedure    = "/" + ServiceName + "/RequestCancelSubscription"
	ReactivateSubscriptionProcedure       = "/" + ServiceName + "/ReactivateSubscription"
	UpgradeSubscriptionProcedure          = "/" + ServiceName + "/UpgradeSubscription"
	ChangeBillingCycleProcedure           = "/" + ServiceName + "/ChangeBillingCycle"
	UpdateAutoRenewProcedure              = "/" + ServiceName + "/UpdateAutoRenew"
	GetSeatUsageProcedure                 = "/" + ServiceName + "/GetSeatUsage"
	PurchaseSeatsProcedure                = "/" + ServiceName + "/PurchaseSeats"
	ListInvoicesProcedure                 = "/" + ServiceName + "/ListInvoices"
	CreateCheckoutProcedure               = "/" + ServiceName + "/CreateCheckout"
	GetCheckoutStatusProcedure            = "/" + ServiceName + "/GetCheckoutStatus"
	GetDeploymentInfoProcedure            = "/" + ServiceName + "/GetDeploymentInfo"
)

// Server implements BillingService — authenticated, org-scoped.
type Server struct {
	billingSvc *billingsvc.Service
	orgSvc     middleware.OrganizationService
}

func NewServer(b *billingsvc.Service, o middleware.OrganizationService) *Server {
	return &Server{billingSvc: b, orgSvc: o}
}

func (s *Server) GetOverview(
	ctx context.Context, req *connect.Request[billingv1.GetOverviewRequest],
) (*connect.Response[billingv1.BillingOverview], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	out, err := s.billingSvc.GetBillingOverview(ctx, tenant.OrganizationID)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoOverview(out)), nil
}

func (s *Server) ListPlans(
	ctx context.Context, req *connect.Request[billingv1.ListPlansRequest],
) (*connect.Response[billingv1.ListPlansResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	plans, err := s.billingSvc.ListPlans(ctx)
	if err != nil {
		return nil, mapServiceError(err)
	}
	items := make([]*billingv1.SubscriptionPlan, 0, len(plans))
	for _, p := range plans {
		items = append(items, toProtoPlan(p))
	}
	return connect.NewResponse(&billingv1.ListPlansResponse{
		Items: items,
		Total: int64(len(items)),
	}), nil
}

func (s *Server) GetSubscription(
	ctx context.Context, req *connect.Request[billingv1.GetSubscriptionRequest],
) (*connect.Response[billingv1.Subscription], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	sub, err := s.billingSvc.GetSubscription(ctx, tenant.OrganizationID)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoSubscription(sub)), nil
}

func (s *Server) CreateSubscription(
	ctx context.Context, req *connect.Request[billingv1.CreateSubscriptionRequest],
) (*connect.Response[billingv1.Subscription], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	sub, err := s.billingSvc.CreateSubscription(ctx, tenant.OrganizationID, req.Msg.GetPlanName())
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoSubscription(sub)), nil
}

func (s *Server) UpdateSubscription(
	ctx context.Context, req *connect.Request[billingv1.UpdateSubscriptionRequest],
) (*connect.Response[billingv1.Subscription], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	sub, err := s.billingSvc.UpdateSubscription(ctx, tenant.OrganizationID, req.Msg.GetPlanName())
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoSubscription(sub)), nil
}

func (s *Server) CancelSubscription(
	ctx context.Context, req *connect.Request[billingv1.CancelSubscriptionRequest],
) (*connect.Response[billingv1.CancelSubscriptionResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	if err := s.billingSvc.CancelSubscription(ctx, tenant.OrganizationID); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&billingv1.CancelSubscriptionResponse{}), nil
}

func (s *Server) RequestCancelSubscription(
	ctx context.Context, req *connect.Request[billingv1.RequestCancelSubscriptionRequest],
) (*connect.Response[billingv1.RequestCancelSubscriptionResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if err := requireOwner(ctx); err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	sub, err := s.billingSvc.GetSubscription(ctx, tenant.OrganizationID)
	if err != nil {
		return nil, mapServiceError(err)
	}

	if err := cancelViaProvider(ctx, s.billingSvc, sub, req.Msg.GetImmediate()); err != nil {
		return nil, err
	}

	if req.Msg.GetImmediate() {
		if err := s.billingSvc.CancelSubscription(ctx, tenant.OrganizationID); err != nil {
			return nil, mapServiceError(err)
		}
		return connect.NewResponse(&billingv1.RequestCancelSubscriptionResponse{Immediate: true}), nil
	}
	if err := s.billingSvc.SetCancelAtPeriodEnd(ctx, tenant.OrganizationID, true); err != nil {
		return nil, mapServiceError(err)
	}
	end := sub.CurrentPeriodEnd.UTC().Format("2006-01-02T15:04:05Z")
	return connect.NewResponse(&billingv1.RequestCancelSubscriptionResponse{
		CurrentPeriodEnd: &end,
	}), nil
}

func (s *Server) ReactivateSubscription(
	ctx context.Context, req *connect.Request[billingv1.ReactivateSubscriptionRequest],
) (*connect.Response[billingv1.Subscription], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if err := requireOwner(ctx); err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	sub, err := s.billingSvc.GetSubscription(ctx, tenant.OrganizationID)
	if err != nil {
		return nil, mapServiceError(err)
	}
	if !sub.CancelAtPeriodEnd {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("subscription is not pending cancellation"))
	}
	if err := reactivateViaProvider(ctx, s.billingSvc, sub); err != nil {
		return nil, err
	}
	if err := s.billingSvc.SetCancelAtPeriodEnd(ctx, tenant.OrganizationID, false); err != nil {
		return nil, mapServiceError(err)
	}
	updated, err := s.billingSvc.GetSubscription(ctx, tenant.OrganizationID)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoSubscription(updated)), nil
}

func (s *Server) UpgradeSubscription(
	ctx context.Context, req *connect.Request[billingv1.UpgradeSubscriptionRequest],
) (*connect.Response[billingv1.Subscription], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if err := requireOwner(ctx); err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	sub, err := s.billingSvc.UpgradePlan(ctx, tenant.OrganizationID, req.Msg.GetPlanName())
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoSubscription(sub)), nil
}

func (s *Server) ChangeBillingCycle(
	ctx context.Context, req *connect.Request[billingv1.ChangeBillingCycleRequest],
) (*connect.Response[billingv1.ChangeBillingCycleResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if err := requireOwner(ctx); err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	sub, err := s.billingSvc.GetSubscription(ctx, tenant.OrganizationID)
	if err != nil {
		return nil, mapServiceError(err)
	}
	if sub.BillingCycle == req.Msg.GetBillingCycle() {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("already on this billing cycle"))
	}
	if err := s.billingSvc.SetNextBillingCycle(ctx, tenant.OrganizationID, req.Msg.GetBillingCycle()); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&billingv1.ChangeBillingCycleResponse{
		CurrentCycle:  sub.BillingCycle,
		NextCycle:     req.Msg.GetBillingCycle(),
		EffectiveDate: sub.CurrentPeriodEnd.UTC().Format("2006-01-02T15:04:05Z"),
	}), nil
}

func (s *Server) UpdateAutoRenew(
	ctx context.Context, req *connect.Request[billingv1.UpdateAutoRenewRequest],
) (*connect.Response[billingv1.Subscription], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if err := requireOwner(ctx); err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	sub, err := s.billingSvc.GetSubscription(ctx, tenant.OrganizationID)
	if err != nil {
		return nil, mapServiceError(err)
	}
	if err := s.billingSvc.SetAutoRenew(ctx, tenant.OrganizationID, req.Msg.GetAutoRenew()); err != nil {
		return nil, mapServiceError(err)
	}
	sub.AutoRenew = req.Msg.GetAutoRenew()
	return connect.NewResponse(toProtoSubscription(sub)), nil
}

func (s *Server) GetSeatUsage(
	ctx context.Context, req *connect.Request[billingv1.GetSeatUsageRequest],
) (*connect.Response[billingv1.SeatUsage], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	usage, err := s.billingSvc.GetSeatUsage(ctx, tenant.OrganizationID)
	if err != nil {
		if errors.Is(err, billingsvc.ErrSubscriptionNotFound) {
			// Mirror REST default for free plan (billing_seats.go:18).
			return connect.NewResponse(&billingv1.SeatUsage{
				TotalSeats: 1, UsedSeats: 1, AvailableSeats: 0, MaxSeats: 1, CanAddSeats: false,
			}), nil
		}
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoSeatUsage(usage)), nil
}

func (s *Server) PurchaseSeats(
	ctx context.Context, req *connect.Request[billingv1.PurchaseSeatsRequest],
) (*connect.Response[billingv1.PurchaseSeatsResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if err := requireOwner(ctx); err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	if err := s.billingSvc.UpdateSeats(ctx, tenant.OrganizationID, int(req.Msg.GetSeats())); err != nil {
		return nil, mapServiceError(err)
	}
	out := &billingv1.PurchaseSeatsResponse{}
	if usage, e := s.billingSvc.GetSeatUsage(ctx, tenant.OrganizationID); e == nil {
		out.Seats = toProtoSeatUsage(usage)
	}
	return connect.NewResponse(out), nil
}

func (s *Server) ListInvoices(
	ctx context.Context, req *connect.Request[billingv1.ListInvoicesRequest],
) (*connect.Response[billingv1.ListInvoicesResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	limit := int(req.Msg.GetLimit())
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := int(req.Msg.GetOffset())
	invoices, err := s.billingSvc.GetInvoicesByOrg(ctx, tenant.OrganizationID, limit, offset)
	if err != nil {
		return nil, mapServiceError(err)
	}
	items := make([]*billingv1.Invoice, 0, len(invoices))
	for _, i := range invoices {
		items = append(items, toProtoInvoice(i))
	}
	return connect.NewResponse(&billingv1.ListInvoicesResponse{
		Items:  items,
		Total:  int64(len(items)),
		Limit:  int32(limit),
		Offset: int32(offset),
	}), nil
}

func (s *Server) CreateCheckout(
	ctx context.Context, req *connect.Request[billingv1.CreateCheckoutRequest],
) (*connect.Response[billingv1.CreateCheckoutResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if err := requireOwner(ctx); err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)

	priceCalc, providerName, provider, err := validateAndCalculateCheckout(ctx, s.billingSvc, tenant, req.Msg)
	if err != nil {
		return nil, err
	}

	resp, err := createCheckoutSession(ctx, s.billingSvc, tenant, req.Msg, priceCalc, providerName, provider)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (s *Server) GetCheckoutStatus(
	ctx context.Context, req *connect.Request[billingv1.GetCheckoutStatusRequest],
) (*connect.Response[billingv1.CheckoutStatus], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	order, err := s.billingSvc.GetPaymentOrderByNo(ctx, req.Msg.GetOrderNo())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("order not found"))
	}
	if order.OrganizationID != tenant.OrganizationID {
		return nil, connect.NewError(connect.CodePermissionDenied,
			errors.New("order belongs to another organization"))
	}
	return connect.NewResponse(toProtoCheckoutStatus(order)), nil
}

func (s *Server) GetDeploymentInfo(
	ctx context.Context, req *connect.Request[billingv1.GetDeploymentInfoRequest],
) (*connect.Response[billingv1.DeploymentInfo], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	_ = ctx
	return connect.NewResponse(toProtoDeploymentInfo(s.billingSvc.GetDeploymentInfo())), nil
}

// Mount registers all BillingService procedures on mux behind the auth
// interceptor supplied via opts (cmd/server/connect_init.go).
func Mount(mux *http.ServeMux, srv *Server, opts ...connect.HandlerOption) {
	mux.Handle(GetOverviewProcedure, connect.NewUnaryHandler(GetOverviewProcedure, srv.GetOverview, opts...))
	mux.Handle(ListPlansProcedure, connect.NewUnaryHandler(ListPlansProcedure, srv.ListPlans, opts...))
	mux.Handle(GetSubscriptionProcedure, connect.NewUnaryHandler(GetSubscriptionProcedure, srv.GetSubscription, opts...))
	mux.Handle(CreateSubscriptionProcedure, connect.NewUnaryHandler(CreateSubscriptionProcedure, srv.CreateSubscription, opts...))
	mux.Handle(UpdateSubscriptionProcedure, connect.NewUnaryHandler(UpdateSubscriptionProcedure, srv.UpdateSubscription, opts...))
	mux.Handle(CancelSubscriptionProcedure, connect.NewUnaryHandler(CancelSubscriptionProcedure, srv.CancelSubscription, opts...))
	mux.Handle(RequestCancelSubscriptionProcedure, connect.NewUnaryHandler(RequestCancelSubscriptionProcedure, srv.RequestCancelSubscription, opts...))
	mux.Handle(ReactivateSubscriptionProcedure, connect.NewUnaryHandler(ReactivateSubscriptionProcedure, srv.ReactivateSubscription, opts...))
	mux.Handle(UpgradeSubscriptionProcedure, connect.NewUnaryHandler(UpgradeSubscriptionProcedure, srv.UpgradeSubscription, opts...))
	mux.Handle(ChangeBillingCycleProcedure, connect.NewUnaryHandler(ChangeBillingCycleProcedure, srv.ChangeBillingCycle, opts...))
	mux.Handle(UpdateAutoRenewProcedure, connect.NewUnaryHandler(UpdateAutoRenewProcedure, srv.UpdateAutoRenew, opts...))
	mux.Handle(GetSeatUsageProcedure, connect.NewUnaryHandler(GetSeatUsageProcedure, srv.GetSeatUsage, opts...))
	mux.Handle(PurchaseSeatsProcedure, connect.NewUnaryHandler(PurchaseSeatsProcedure, srv.PurchaseSeats, opts...))
	mux.Handle(ListInvoicesProcedure, connect.NewUnaryHandler(ListInvoicesProcedure, srv.ListInvoices, opts...))
	mux.Handle(CreateCheckoutProcedure, connect.NewUnaryHandler(CreateCheckoutProcedure, srv.CreateCheckout, opts...))
	mux.Handle(GetCheckoutStatusProcedure, connect.NewUnaryHandler(GetCheckoutStatusProcedure, srv.GetCheckoutStatus, opts...))
	mux.Handle(GetDeploymentInfoProcedure, connect.NewUnaryHandler(GetDeploymentInfoProcedure, srv.GetDeploymentInfo, opts...))
}

// requireOwner mirrors REST's `if tenant.UserRole != "owner"` guard.
// ResolveOrgScope already populated TenantContext with the user role.
func requireOwner(ctx context.Context) error {
	tenant := middleware.GetTenant(ctx)
	if tenant == nil {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("missing tenant context"))
	}
	if tenant.UserRole != "owner" {
		return connect.NewError(connect.CodePermissionDenied,
			errors.New("organization owner role required"))
	}
	return nil
}

// mapServiceError translates billing-domain sentinels to Connect codes
// (conventions §10). Mirrors handleQuotaError + the per-handler switches in
// billing_*.go but consolidated for the dual-track migration.
func mapServiceError(err error) error {
	switch {
	case errors.Is(err, billingsvc.ErrSubscriptionNotFound),
		errors.Is(err, billingsvc.ErrPlanNotFound),
		errors.Is(err, billingsvc.ErrPriceNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, billingsvc.ErrQuotaExceeded):
		return connect.NewError(connect.CodeResourceExhausted, err)
	case errors.Is(err, billingsvc.ErrSubscriptionFrozen),
		errors.Is(err, billingsvc.ErrSubscriptionNotActive),
		errors.Is(err, billingsvc.ErrInvalidPlan),
		errors.Is(err, billingsvc.ErrSeatCountExceedsLimit):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

// cancelViaProvider drives Stripe/LemonSqueezy cancellation, mirroring
// billing_subscription_cancel.go:45-66. Returns Connect-coded errors when
// the provider call fails.
func cancelViaProvider(ctx context.Context, svc *billingsvc.Service, sub *billingdomain.Subscription, immediate bool) error {
	factory := svc.GetPaymentFactory()
	if factory == nil {
		return nil
	}
	provider, subscriptionID := pickProvider(factory, sub)
	if provider == nil || subscriptionID == "" {
		return nil
	}
	if err := provider.CancelSubscription(ctx, subscriptionID, immediate); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("provider cancel: %w", err))
	}
	return nil
}

// reactivateViaProvider mirrors billing_subscription_cancel.go:107-132 —
// reactivate = "cancel with immediate=false" against the same provider API.
func reactivateViaProvider(ctx context.Context, svc *billingsvc.Service, sub *billingdomain.Subscription) error {
	factory := svc.GetPaymentFactory()
	if factory == nil {
		return nil
	}
	provider, subscriptionID := pickProvider(factory, sub)
	if provider == nil || subscriptionID == "" {
		return nil
	}
	if err := provider.CancelSubscription(ctx, subscriptionID, false); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("provider reactivate: %w", err))
	}
	return nil
}

// pickProvider mirrors the REST handler's provider selection (subscription
// has either LemonSqueezy or Stripe IDs but not both).
func pickProvider(factory *payment.Factory, sub *billingdomain.Subscription) (payment.Provider, string) {
	if sub.LemonSqueezySubscriptionID != nil && *sub.LemonSqueezySubscriptionID != "" {
		p, err := factory.GetProvider(billingdomain.PaymentProviderLemonSqueezy)
		if err == nil {
			return p, *sub.LemonSqueezySubscriptionID
		}
	}
	if sub.StripeSubscriptionID != nil && *sub.StripeSubscriptionID != "" {
		p, err := factory.GetProvider(billingdomain.PaymentProviderStripe)
		if err == nil {
			return p, *sub.StripeSubscriptionID
		}
	}
	return nil, ""
}

// validateAndCalculateCheckout mirrors REST's same-named helper. Returns
// Connect-coded errors so handlers don't need to translate twice.
func validateAndCalculateCheckout(
	ctx context.Context, svc *billingsvc.Service, tenant *middleware.TenantContext,
	req *billingv1.CreateCheckoutRequest,
) (*billingsvc.PriceCalculation, string, payment.Provider, error) {
	orderType := req.GetOrderType()
	planName := req.GetPlanName()
	seats := int(req.GetSeats())
	billingCycle := req.GetBillingCycle()
	if billingCycle == "" {
		billingCycle = billingdomain.BillingCycleMonthly
	}

	if (orderType == billingdomain.OrderTypeSubscription || orderType == billingdomain.OrderTypePlanUpgrade) && planName == "" {
		return nil, "", nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("plan_name required for subscription / plan_upgrade"))
	}
	if orderType == billingdomain.OrderTypeSeatPurchase && seats <= 0 {
		return nil, "", nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("seats must be positive for seat_purchase"))
	}

	factory := svc.GetPaymentFactory()
	if factory == nil {
		return nil, "", nil, connect.NewError(connect.CodeUnavailable,
			errors.New("payment service not configured"))
	}

	var provider payment.Provider
	var providerName string
	var err error
	if req.GetProvider() != "" {
		providerName = req.GetProvider()
		provider, err = factory.GetProvider(providerName)
	} else {
		provider, err = factory.GetDefaultProvider()
		if provider != nil {
			providerName = provider.GetProviderName()
		}
	}
	if err != nil {
		return nil, "", nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	priceCalc, err := calculatePrice(ctx, svc, tenant.OrganizationID, orderType, planName, billingCycle, seats)
	if err != nil {
		return nil, "", nil, err
	}
	return priceCalc, providerName, provider, nil
}

func calculatePrice(
	ctx context.Context, svc *billingsvc.Service, orgID int64,
	orderType, planName, billingCycle string, seats int,
) (*billingsvc.PriceCalculation, error) {
	switch orderType {
	case billingdomain.OrderTypeSubscription:
		if seats <= 0 {
			seats = 1
		}
		pc, err := svc.CalculateSubscriptionPrice(ctx, planName, billingCycle, seats)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return pc, nil
	case billingdomain.OrderTypePlanUpgrade:
		pc, err := svc.CalculateUpgradePrice(ctx, orgID, planName)
		if err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("upgrade calculation: %w", err))
		}
		return pc, nil
	case billingdomain.OrderTypeSeatPurchase:
		pc, err := svc.CalculateSeatPurchasePrice(ctx, orgID, seats)
		if err != nil {
			if errors.Is(err, billingsvc.ErrInvalidPlan) ||
				errors.Is(err, billingsvc.ErrQuotaExceeded) {
				return nil, connect.NewError(connect.CodeFailedPrecondition, err)
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return pc, nil
	case billingdomain.OrderTypeRenewal:
		pc, err := svc.CalculateRenewalPrice(ctx, orgID, billingCycle)
		if err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("no subscription to renew: %w", err))
		}
		return pc, nil
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("invalid order type: %s", orderType))
	}
}

func createCheckoutSession(
	ctx context.Context, svc *billingsvc.Service, tenant *middleware.TenantContext,
	req *billingv1.CreateCheckoutRequest, priceCalc *billingsvc.PriceCalculation,
	providerName string, provider payment.Provider,
) (*billingv1.CreateCheckoutResponse, error) {
	orderNo := fmt.Sprintf("ORD-%d-%s", tenant.OrganizationID, uuid.New().String()[:8])
	metadata := map[string]string{"order_no": orderNo}
	if priceCalc.LemonSqueezyVariantID != "" {
		metadata["variant_id"] = priceCalc.LemonSqueezyVariantID
	}
	if priceCalc.StripePrice != "" {
		metadata["stripe_price_id"] = priceCalc.StripePrice
	}

	billingCycle := req.GetBillingCycle()
	if billingCycle == "" {
		billingCycle = billingdomain.BillingCycleMonthly
	}

	checkoutReq := &payment.CheckoutRequest{
		OrganizationID: tenant.OrganizationID,
		UserID:         tenant.UserID,
		OrderType:      req.GetOrderType(),
		BillingCycle:   billingCycle,
		Seats:          priceCalc.Seats,
		Currency:       "usd",
		Amount:         priceCalc.Amount,
		ActualAmount:   priceCalc.ActualAmount,
		SuccessURL:     req.GetSuccessUrl(),
		CancelURL:      req.GetCancelUrl(),
		IdempotencyKey: orderNo,
		Metadata:       metadata,
	}
	if priceCalc.PlanID > 0 {
		checkoutReq.PlanID = priceCalc.PlanID
	}

	resp, err := provider.CreateCheckoutSession(ctx, checkoutReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("create checkout: %w", err))
	}

	var planID *int64
	if priceCalc.PlanID > 0 {
		planID = &priceCalc.PlanID
	}
	order := &billingdomain.PaymentOrder{
		OrganizationID:  tenant.OrganizationID,
		OrderNo:         orderNo,
		ExternalOrderNo: &resp.ExternalOrderNo,
		OrderType:       req.GetOrderType(),
		PlanID:          planID,
		BillingCycle:    billingCycle,
		Seats:           priceCalc.Seats,
		Amount:          priceCalc.Amount,
		ActualAmount:    priceCalc.ActualAmount,
		Currency:        "usd",
		Status:          billingdomain.OrderStatusPending,
		PaymentProvider: providerName,
		ExpiresAt:       &resp.ExpiresAt,
		CreatedByID:     tenant.UserID,
	}
	if err := svc.CreatePaymentOrder(ctx, order); err != nil {
		slog.ErrorContext(ctx, "failed to save payment order", "order_no", orderNo, "error", err)
		return nil, connect.NewError(connect.CodeInternal,
			errors.New("failed to create payment order"))
	}

	out := &billingv1.CreateCheckoutResponse{
		OrderNo:    orderNo,
		SessionId:  resp.SessionID,
		SessionUrl: resp.SessionURL,
		ExpiresAt:  resp.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
		Provider:   providerName,
	}
	if resp.QRCodeURL != "" {
		out.QrCodeUrl = &resp.QRCodeURL
	}
	return out, nil
}
