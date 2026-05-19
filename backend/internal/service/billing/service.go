package billing

import (
	"context"
	"errors"
	"strconv"

	"github.com/stripe/stripe-go/v76"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

var (
	ErrSubscriptionNotFound  = errors.New("subscription not found")
	ErrPlanNotFound          = errors.New("plan not found")
	ErrPriceNotFound         = errors.New("price not found for currency")
	ErrQuotaExceeded         = errors.New("quota exceeded")
	ErrInvalidPlan           = errors.New("invalid plan")
	ErrOrderNotFound         = errors.New("order not found")
	ErrOrderExpired          = errors.New("order expired")
	ErrInvalidOrderStatus    = errors.New("invalid order status")
	ErrSeatCountExceedsLimit = errors.New("current seat count exceeds target plan limit")
	ErrSubscriptionNotActive     = errors.New("subscription is not active")
	ErrSubscriptionFrozen        = errors.New("subscription is frozen, please renew to continue")
	ErrSubscriptionAlreadyExists = errors.New("subscription already exists for this organization")
)

type Service struct {
	repo           billing.BillingRepository
	stripeEnabled  bool
	paymentFactory *payment.Factory
	paymentConfig  *config.PaymentConfig
	stripeClient   StripeClient // Stripe client for API operations (allows mocking)
}

func NewService(repo billing.BillingRepository, stripeKey string) *Service {
	if stripeKey != "" {
		stripe.Key = stripeKey
	}
	svc := &Service{
		repo:          repo,
		stripeEnabled: stripeKey != "",
	}
	if svc.stripeEnabled {
		svc.stripeClient = NewDefaultStripeClient()
	}
	return svc
}

func NewServiceWithConfig(repo billing.BillingRepository, appConfig *config.Config) *Service {
	svc := &Service{
		repo: repo,
	}

	if appConfig == nil {
		return svc
	}

	cfg := &appConfig.Payment
	svc.paymentConfig = cfg

	svc.paymentFactory = payment.NewFactoryFromConfig(appConfig)
	svc.stripeEnabled = cfg.StripeEnabled()

	if cfg.StripeEnabled() {
		stripe.Key = cfg.Stripe.SecretKey
		svc.stripeClient = NewDefaultStripeClient()
	}

	return svc
}

func (s *Service) SetStripeClient(client StripeClient) {
	s.stripeClient = client
}

func (s *Service) SetStripeEnabled(enabled bool) {
	s.stripeEnabled = enabled
}

func (s *Service) GetPaymentFactory() *payment.Factory {
	return s.paymentFactory
}

func (s *Service) CreateStripeCustomer(ctx context.Context, orgID int64, email, name string) (string, error) {
	if !s.stripeEnabled || s.stripeClient == nil {
		return "", nil
	}

	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
		Metadata: map[string]string{
			"organization_id": strconv.FormatInt(orgID, 10),
		},
	}

	c, err := s.stripeClient.CreateCustomer(params)
	if err != nil {
		return "", err
	}

	_ = s.repo.UpdateSubscriptionFieldsByOrg(ctx, orgID, map[string]interface{}{
		"stripe_customer_id": c.ID,
	})

	return c.ID, nil
}
