package billing

import (
	"context"
	"errors"

	"github.com/stripe/stripe-go/v76"
	"gorm.io/gorm"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/config"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
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
	ErrSubscriptionFrozen        = errors.New("subscription is frozen, please renew to continue")
	ErrSubscriptionAlreadyExists = errors.New("subscription already exists for this organization")
)

// Service handles billing operations
type Service struct {
	db             *gorm.DB
	stripeEnabled  bool
	paymentFactory *payment.Factory
	paymentConfig  *config.PaymentConfig
	stripeClient   StripeClient // Stripe client for API operations (allows mocking)
}

// NewService creates a new billing service without payment configuration.
// This is primarily for testing purposes where payment providers are not needed.
// For production use, prefer NewServiceWithConfig which supports all payment providers.
func NewService(db *gorm.DB, stripeKey string) *Service {
	if stripeKey != "" {
		stripe.Key = stripeKey
	}
	svc := &Service{
		db:            db,
		stripeEnabled: stripeKey != "",
	}
	// Use default Stripe client if Stripe is enabled
	if svc.stripeEnabled {
		svc.stripeClient = NewDefaultStripeClient()
	}
	return svc
}

// NewServiceWithConfig creates a new billing service with full configuration
// appConfig is needed for URL derivation (AlipayNotifyURL, WeChatNotifyURL, etc.)
// If appConfig is nil, returns a service with no payment providers configured.
func NewServiceWithConfig(db *gorm.DB, appConfig *config.Config) *Service {
	svc := &Service{
		db: db,
	}

	// Handle nil config gracefully - return service without payment providers
	if appConfig == nil {
		return svc
	}

	cfg := &appConfig.Payment
	svc.paymentConfig = cfg

	// Use NewFactoryWithDB to support license provider and URL derivation
	svc.paymentFactory = payment.NewFactoryWithDB(appConfig, db)
	svc.stripeEnabled = cfg.StripeEnabled()

	// Set Stripe key and client if enabled
	if cfg.StripeEnabled() {
		stripe.Key = cfg.Stripe.SecretKey
		svc.stripeClient = NewDefaultStripeClient()
	}

	return svc
}

// SetStripeClient sets a custom Stripe client (for testing with mocks)
func (s *Service) SetStripeClient(client StripeClient) {
	s.stripeClient = client
}

// SetStripeEnabled enables or disables Stripe (for testing)
func (s *Service) SetStripeEnabled(enabled bool) {
	s.stripeEnabled = enabled
}

// GetPaymentFactory returns the payment factory
func (s *Service) GetPaymentFactory() *payment.Factory {
	return s.paymentFactory
}

// CreateStripeCustomer creates a Stripe customer for an organization
func (s *Service) CreateStripeCustomer(ctx context.Context, orgID int64, email, name string) (string, error) {
	if !s.stripeEnabled || s.stripeClient == nil {
		return "", nil
	}

	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
		Metadata: map[string]string{
			"organization_id": string(rune(orgID)),
		},
	}

	c, err := s.stripeClient.CreateCustomer(params)
	if err != nil {
		return "", err
	}

	// Update subscription with Stripe customer ID
	s.db.WithContext(ctx).Model(&billing.Subscription{}).
		Where("organization_id = ?", orgID).
		Update("stripe_customer_id", c.ID)

	return c.ID, nil
}
