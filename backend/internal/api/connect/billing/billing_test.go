package billingconnect

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	billingdomain "github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	billingsvc "github.com/anthropics/agentsmesh/backend/internal/service/billing"
	billingv1 "github.com/anthropics/agentsmesh/proto/gen/go/billing/v1"
)

type fakeOrg struct {
	id   int64
	slug string
}

func (f fakeOrg) GetID() int64    { return f.id }
func (f fakeOrg) GetSlug() string { return f.slug }
func (f fakeOrg) GetName() string { return f.slug }

type fakeOrgService struct {
	role string
}

func (f *fakeOrgService) GetBySlug(_ context.Context, slug string) (middleware.OrganizationGetter, error) {
	if slug == "missing" {
		return nil, errors.New("org not found")
	}
	return fakeOrg{id: 7, slug: slug}, nil
}
func (f *fakeOrgService) IsMember(context.Context, int64, int64) (bool, error) { return true, nil }
func (f *fakeOrgService) GetMemberRole(context.Context, int64, int64) (string, error) {
	return f.role, nil
}

func ctxAsUser(userID int64) context.Context {
	return middleware.SetTenant(context.Background(), &middleware.TenantContext{UserID: userID})
}

func connectCodeOf(t *testing.T, err error) connect.Code {
	t.Helper()
	var ce *connect.Error
	require.True(t, errors.As(err, &ce), "expected *connect.Error, got %v", err)
	return ce.Code()
}

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	tt, err := time.Parse(time.RFC3339, s)
	require.NoError(t, err)
	return tt
}

// --- Auth / org scope guards ---

func TestGetOverview_MissingOrgSlug_InvalidArgument(t *testing.T) {
	srv := NewServer(nil, &fakeOrgService{role: "member"})
	_, err := srv.GetOverview(ctxAsUser(42), connect.NewRequest(&billingv1.GetOverviewRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestGetOverview_NoAuth_Unauthenticated(t *testing.T) {
	srv := NewServer(nil, &fakeOrgService{role: "member"})
	_, err := srv.GetOverview(context.Background(),
		connect.NewRequest(&billingv1.GetOverviewRequest{OrgSlug: "acme"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connectCodeOf(t, err))
}

func TestRequestCancelSubscription_NonOwner_PermissionDenied(t *testing.T) {
	srv := NewServer(nil, &fakeOrgService{role: "member"})
	_, err := srv.RequestCancelSubscription(ctxAsUser(42),
		connect.NewRequest(&billingv1.RequestCancelSubscriptionRequest{OrgSlug: "acme"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connectCodeOf(t, err))
}

func TestPurchaseSeats_NonOwner_PermissionDenied(t *testing.T) {
	srv := NewServer(nil, &fakeOrgService{role: "member"})
	_, err := srv.PurchaseSeats(ctxAsUser(42),
		connect.NewRequest(&billingv1.PurchaseSeatsRequest{OrgSlug: "acme", Seats: 5}))
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connectCodeOf(t, err))
}

func TestCreateCheckout_NonOwner_PermissionDenied(t *testing.T) {
	srv := NewServer(nil, &fakeOrgService{role: "admin"})
	planName := "pro"
	_, err := srv.CreateCheckout(ctxAsUser(42),
		connect.NewRequest(&billingv1.CreateCheckoutRequest{
			OrgSlug:    "acme",
			OrderType:  billingdomain.OrderTypeSubscription,
			PlanName:   &planName,
			SuccessUrl: "https://x/success",
			CancelUrl:  "https://x/cancel",
		}))
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connectCodeOf(t, err))
}

// --- requireOwner helper ---

func TestRequireOwner_NoTenant_Unauthenticated(t *testing.T) {
	err := requireOwner(context.Background())
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connectCodeOf(t, err))
}

func TestRequireOwner_NonOwner_PermissionDenied(t *testing.T) {
	ctx := middleware.SetTenant(context.Background(), &middleware.TenantContext{UserRole: "member"})
	err := requireOwner(ctx)
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connectCodeOf(t, err))
}

func TestRequireOwner_Owner_NoError(t *testing.T) {
	ctx := middleware.SetTenant(context.Background(), &middleware.TenantContext{UserRole: "owner"})
	require.NoError(t, requireOwner(ctx))
}

func TestRequireOwnerOrAdmin_AdminAllowed(t *testing.T) {
	ctx := middleware.SetTenant(context.Background(), &middleware.TenantContext{UserRole: "admin"})
	require.NoError(t, requireOwnerOrAdmin(ctx))
}

// --- mapServiceError table ---

func TestMapServiceError(t *testing.T) {
	cases := []struct {
		name string
		in   error
		want connect.Code
	}{
		{"subscription_not_found", billingsvc.ErrSubscriptionNotFound, connect.CodeNotFound},
		{"plan_not_found", billingsvc.ErrPlanNotFound, connect.CodeNotFound},
		{"price_not_found", billingsvc.ErrPriceNotFound, connect.CodeNotFound},
		{"order_not_found", billingsvc.ErrOrderNotFound, connect.CodeNotFound},
		{"already_exists", billingsvc.ErrSubscriptionAlreadyExists, connect.CodeAlreadyExists},
		{"invalid_plan", billingsvc.ErrInvalidPlan, connect.CodeFailedPrecondition},
		{"invalid_order_status", billingsvc.ErrInvalidOrderStatus, connect.CodeFailedPrecondition},
		{"seat_count_exceeds_limit", billingsvc.ErrSeatCountExceedsLimit, connect.CodeFailedPrecondition},
		{"subscription_not_active", billingsvc.ErrSubscriptionNotActive, connect.CodeFailedPrecondition},
		{"subscription_frozen", billingsvc.ErrSubscriptionFrozen, connect.CodeFailedPrecondition},
		{"quota_exceeded", billingsvc.ErrQuotaExceeded, connect.CodeFailedPrecondition},
		{"order_expired", billingsvc.ErrOrderExpired, connect.CodeDeadlineExceeded},
		{"generic_error", errors.New("oops"), connect.CodeInternal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := connectCodeOf(t, mapServiceError(tc.in))
			assert.Equal(t, tc.want, got)
		})
	}
}

// --- toProto* mappers ---

func TestToProtoPlan_AllFields(t *testing.T) {
	created := mustParseTime(t, "2026-05-01T00:00:00Z")
	p := &billingdomain.SubscriptionPlan{
		ID:                  1,
		Name:                "pro",
		DisplayName:         "Pro",
		PricePerSeatMonthly: 9.99,
		PricePerSeatYearly:  99.0,
		IncludedPodMinutes:  100,
		PricePerExtraMinute: 0.05,
		MaxUsers:            10,
		MaxRunners:          5,
		MaxConcurrentPods:   3,
		MaxRepositories:     20,
		IsActive:            true,
		CreatedAt:           created,
	}
	out := toProtoPlan(p)
	require.NotNil(t, out)
	assert.Equal(t, int64(1), out.GetId())
	assert.Equal(t, "pro", out.GetName())
	assert.Equal(t, "Pro", out.GetDisplayName())
	assert.Equal(t, 9.99, out.GetPricePerSeatMonthly())
	assert.Equal(t, 99.0, out.GetPricePerSeatYearly())
	assert.Equal(t, int32(100), out.GetIncludedPodMinutes())
	assert.Equal(t, int32(10), out.GetMaxUsers())
	assert.True(t, out.GetIsActive())
	assert.Equal(t, "2026-05-01T00:00:00Z", out.GetCreatedAt())
}

func TestToProtoPlan_Nil(t *testing.T) {
	assert.Nil(t, toProtoPlan(nil))
}

func TestToProtoSubscription_AllFieldsAndOptionals(t *testing.T) {
	stripeCustomerID := "cus_123"
	stripeSubscriptionID := "sub_456"
	lsCustomerID := "ls_cust_789"
	canceledAt := mustParseTime(t, "2026-05-10T00:00:00Z")
	frozenAt := mustParseTime(t, "2026-05-11T00:00:00Z")
	downgradeToPlan := "based"

	s := &billingdomain.Subscription{
		ID:                         42,
		OrganizationID:             7,
		PlanID:                     1,
		Status:                     billingdomain.SubscriptionStatusActive,
		BillingCycle:               billingdomain.BillingCycleMonthly,
		CurrentPeriodStart:         mustParseTime(t, "2026-05-01T00:00:00Z"),
		CurrentPeriodEnd:           mustParseTime(t, "2026-06-01T00:00:00Z"),
		AutoRenew:                  true,
		SeatCount:                  3,
		StripeCustomerID:           &stripeCustomerID,
		StripeSubscriptionID:       &stripeSubscriptionID,
		LemonSqueezyCustomerID:     &lsCustomerID,
		CanceledAt:                 &canceledAt,
		CancelAtPeriodEnd:          true,
		FrozenAt:                   &frozenAt,
		DowngradeToPlan:            &downgradeToPlan,
		CreatedAt:                  mustParseTime(t, "2026-05-01T00:00:00Z"),
		UpdatedAt:                  mustParseTime(t, "2026-05-10T00:00:00Z"),
	}
	out := toProtoSubscription(s)
	require.NotNil(t, out)
	assert.Equal(t, int64(42), out.GetId())
	assert.Equal(t, "active", out.GetStatus())
	assert.True(t, out.GetCancelAtPeriodEnd())
	assert.Equal(t, "cus_123", out.GetStripeCustomerId())
	assert.Equal(t, "sub_456", out.GetStripeSubscriptionId())
	assert.Equal(t, "ls_cust_789", out.GetLemonsqueezyCustomerId())
	assert.Equal(t, "based", out.GetDowngradeToPlan())
	assert.Equal(t, "2026-05-10T00:00:00Z", out.GetCanceledAt())
	assert.Equal(t, "2026-05-11T00:00:00Z", out.GetFrozenAt())
}

func TestToProtoSubscription_OptionalsAbsent(t *testing.T) {
	s := &billingdomain.Subscription{
		Status:             billingdomain.SubscriptionStatusActive,
		BillingCycle:       billingdomain.BillingCycleMonthly,
		CurrentPeriodStart: mustParseTime(t, "2026-05-01T00:00:00Z"),
		CurrentPeriodEnd:   mustParseTime(t, "2026-06-01T00:00:00Z"),
	}
	out := toProtoSubscription(s)
	require.NotNil(t, out)
	assert.Nil(t, out.StripeCustomerId, "absent stripe_customer_id round-trips as nil")
	assert.Nil(t, out.CanceledAt, "absent canceled_at round-trips as nil")
	assert.Nil(t, out.FrozenAt, "absent frozen_at round-trips as nil")
	assert.Nil(t, out.DowngradeToPlan)
}

func TestToProtoSubscription_Nil(t *testing.T) {
	assert.Nil(t, toProtoSubscription(nil))
}

func TestToProtoSeatUsage(t *testing.T) {
	u := &billingsvc.SeatUsage{
		TotalSeats:     5,
		UsedSeats:      3,
		AvailableSeats: 2,
		MaxSeats:       10,
		CanAddSeats:    true,
	}
	out := toProtoSeatUsage(u)
	require.NotNil(t, out)
	assert.Equal(t, int32(5), out.GetTotalSeats())
	assert.Equal(t, int32(3), out.GetUsedSeats())
	assert.Equal(t, int32(2), out.GetAvailableSeats())
	assert.Equal(t, int32(10), out.GetMaxSeats())
	assert.True(t, out.GetCanAddSeats())
}

func TestToProtoSeatUsage_Nil(t *testing.T) {
	assert.Nil(t, toProtoSeatUsage(nil))
}

func TestToProtoDeploymentInfo(t *testing.T) {
	d := &billingsvc.DeploymentInfo{
		DeploymentType:     "global",
		AvailableProviders: []string{"stripe", "lemonsqueezy"},
	}
	out := toProtoDeploymentInfo(d)
	require.NotNil(t, out)
	assert.Equal(t, "global", out.GetDeploymentType())
	assert.Equal(t, []string{"stripe", "lemonsqueezy"}, out.GetAvailableProviders())
}

func TestToProtoDeploymentInfo_Nil(t *testing.T) {
	out := toProtoDeploymentInfo(nil)
	require.NotNil(t, out)
	assert.Equal(t, "", out.GetDeploymentType())
}

func TestToProtoInvoice_AllAndOptional(t *testing.T) {
	pdfURL := "https://invoices/123.pdf"
	paid := mustParseTime(t, "2026-05-10T00:00:00Z")
	issued := mustParseTime(t, "2026-05-01T00:00:00Z")
	i := &billingdomain.Invoice{
		ID:             1,
		OrganizationID: 7,
		InvoiceNo:      "INV-001",
		Status:         billingdomain.InvoiceStatusPaid,
		Currency:       "USD",
		Subtotal:       100,
		TaxAmount:      8,
		Total:          108,
		PeriodStart:    mustParseTime(t, "2026-05-01T00:00:00Z"),
		PeriodEnd:      mustParseTime(t, "2026-06-01T00:00:00Z"),
		PDFURL:         &pdfURL,
		IssuedAt:       &issued,
		PaidAt:         &paid,
		CreatedAt:      mustParseTime(t, "2026-05-01T00:00:00Z"),
		UpdatedAt:      mustParseTime(t, "2026-05-10T00:00:00Z"),
	}
	out := toProtoInvoice(i)
	require.NotNil(t, out)
	assert.Equal(t, "INV-001", out.GetInvoiceNo())
	assert.Equal(t, "USD", out.GetCurrency())
	assert.Equal(t, 108.0, out.GetTotal())
	assert.Equal(t, pdfURL, out.GetPdfUrl())
	assert.Equal(t, "2026-05-10T00:00:00Z", out.GetPaidAt())
}

func TestToProtoInvoice_Nil(t *testing.T) {
	assert.Nil(t, toProtoInvoice(nil))
}

func TestToProtoCheckoutStatus(t *testing.T) {
	paid := mustParseTime(t, "2026-05-10T00:00:00Z")
	o := &billingdomain.PaymentOrder{
		OrderNo:      "ORD-123",
		Status:       billingdomain.OrderStatusSucceeded,
		OrderType:    billingdomain.OrderTypeSubscription,
		Amount:       100,
		ActualAmount: 100,
		Currency:     "USD",
		CreatedAt:    mustParseTime(t, "2026-05-01T00:00:00Z"),
		PaidAt:       &paid,
	}
	out := toProtoCheckoutStatus(o)
	require.NotNil(t, out)
	assert.Equal(t, "ORD-123", out.GetOrderNo())
	assert.Equal(t, "succeeded", out.GetStatus())
	assert.Equal(t, 100.0, out.GetAmount())
	assert.Equal(t, "2026-05-10T00:00:00Z", out.GetPaidAt())
}

func TestToProtoCheckoutStatus_Nil(t *testing.T) {
	assert.Nil(t, toProtoCheckoutStatus(nil))
}

// PR #334 regression — every UI field of the public-pricing card must
// survive the mapper unchanged.
func TestToProtoPublicPlanPricing_PR334Fields(t *testing.T) {
	plan := &billingdomain.SubscriptionPlan{
		Name:              "based",
		DisplayName:       "Based",
		MaxUsers:          1,
		MaxRunners:        1,
		MaxRepositories:   5,
		MaxConcurrentPods: 5,
	}
	price := &billingdomain.PlanPrice{
		PriceMonthly: 9.9,
		PriceYearly:  99.0,
	}
	out := toProtoPublicPlanPricing(plan, price)
	require.NotNil(t, out)
	assert.Equal(t, "based", out.GetName())
	assert.Equal(t, "Based", out.GetDisplayName())
	assert.Equal(t, 9.9, out.GetPriceMonthly(),
		"PR #334: price_monthly must round-trip")
	assert.Equal(t, 99.0, out.GetPriceYearly(),
		"PR #334: price_yearly must round-trip")
	assert.Equal(t, int32(1), out.GetMaxUsers())
}

func TestToProtoOverview_NestedUsage(t *testing.T) {
	o := &billingsvc.BillingOverview{
		Plan: &billingdomain.SubscriptionPlan{
			ID: 1, Name: "pro", DisplayName: "Pro",
			CreatedAt: mustParseTime(t, "2026-05-01T00:00:00Z"),
		},
		Status:             "active",
		BillingCycle:       "monthly",
		CurrentPeriodStart: mustParseTime(t, "2026-05-01T00:00:00Z"),
		CurrentPeriodEnd:   mustParseTime(t, "2026-06-01T00:00:00Z"),
		CancelAtPeriodEnd:  false,
		Usage: billingsvc.UsageOverview{
			PodMinutes:         42.5,
			IncludedPodMinutes: 100,
			Users:              3,
			MaxUsers:           5,
		},
	}
	out := toProtoOverview(o)
	require.NotNil(t, out)
	require.NotNil(t, out.GetPlan())
	assert.Equal(t, "pro", out.GetPlan().GetName())
	require.NotNil(t, out.GetUsage())
	assert.Equal(t, 42.5, out.GetUsage().GetPodMinutes())
	assert.Equal(t, int32(3), out.GetUsage().GetUsers())
}

// --- validateCheckoutRequest table ---

func TestValidateCheckoutRequest_OK(t *testing.T) {
	planName := "pro"
	seats := int32(3)
	cases := []*billingv1.CreateCheckoutRequest{
		{
			OrgSlug:    "acme",
			OrderType:  billingdomain.OrderTypeSubscription,
			PlanName:   &planName,
			SuccessUrl: "https://x/s",
			CancelUrl:  "https://x/c",
		},
		{
			OrgSlug:    "acme",
			OrderType:  billingdomain.OrderTypeSeatPurchase,
			Seats:      &seats,
			SuccessUrl: "https://x/s",
			CancelUrl:  "https://x/c",
		},
		{
			OrgSlug:    "acme",
			OrderType:  billingdomain.OrderTypeRenewal,
			SuccessUrl: "https://x/s",
			CancelUrl:  "https://x/c",
		},
	}
	for _, c := range cases {
		assert.NoError(t, validateCheckoutRequest(c), "order_type=%s", c.OrderType)
	}
}

func TestValidateCheckoutRequest_InvalidOrderType(t *testing.T) {
	err := validateCheckoutRequest(&billingv1.CreateCheckoutRequest{
		OrderType:  "garbage",
		SuccessUrl: "https://x/s",
		CancelUrl:  "https://x/c",
	})
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestValidateCheckoutRequest_SubscriptionWithoutPlan(t *testing.T) {
	err := validateCheckoutRequest(&billingv1.CreateCheckoutRequest{
		OrderType:  billingdomain.OrderTypeSubscription,
		SuccessUrl: "https://x/s",
		CancelUrl:  "https://x/c",
	})
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestValidateCheckoutRequest_SeatPurchaseZeroSeats(t *testing.T) {
	seats := int32(0)
	err := validateCheckoutRequest(&billingv1.CreateCheckoutRequest{
		OrderType:  billingdomain.OrderTypeSeatPurchase,
		Seats:      &seats,
		SuccessUrl: "https://x/s",
		CancelUrl:  "https://x/c",
	})
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestValidateCheckoutRequest_MissingURLs(t *testing.T) {
	planName := "pro"
	err := validateCheckoutRequest(&billingv1.CreateCheckoutRequest{
		OrderType: billingdomain.OrderTypeSubscription,
		PlanName:  &planName,
	})
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}
