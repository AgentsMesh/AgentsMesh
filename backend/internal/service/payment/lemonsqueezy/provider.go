package lemonsqueezy

import (
	"context"
	"fmt"
	"strconv"

	lemonsqueezy "github.com/NdoleStudio/lemonsqueezy-go"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/config"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment/types"
)

// Provider implements payment.SubscriptionProvider for LemonSqueezy
type Provider struct {
	client        *lemonsqueezy.Client
	storeID       string
	webhookSecret string
}

// NewProvider creates a new LemonSqueezy provider
func NewProvider(cfg *config.LemonSqueezyConfig) *Provider {
	client := lemonsqueezy.New(lemonsqueezy.WithAPIKey(cfg.APIKey))
	return &Provider{
		client:        client,
		storeID:       cfg.StoreID,
		webhookSecret: cfg.WebhookSecret,
	}
}

// GetProviderName returns the provider name
func (p *Provider) GetProviderName() string {
	return billing.PaymentProviderLemonSqueezy
}

// GetCheckoutStatus checks the status of a checkout session
// LemonSqueezy doesn't have a direct checkout status API, so we return pending
func (p *Provider) GetCheckoutStatus(ctx context.Context, sessionID string) (string, error) {
	// LemonSqueezy checkouts are one-time use and don't have a status check API
	// The status is determined via webhooks
	return billing.OrderStatusPending, nil
}

// RefundPayment initiates a refund
// Note: LemonSqueezy refunds are primarily handled through the dashboard
func (p *Provider) RefundPayment(ctx context.Context, req *types.RefundRequest) (*types.RefundResponse, error) {
	// LemonSqueezy doesn't have a public refund API
	// Refunds should be processed through the LemonSqueezy dashboard
	return nil, fmt.Errorf("refunds must be processed through the LemonSqueezy dashboard")
}

// CreateCustomer creates a customer in LemonSqueezy
// Note: LemonSqueezy automatically creates customers during checkout
func (p *Provider) CreateCustomer(ctx context.Context, email string, name string, metadata map[string]string) (string, error) {
	// LemonSqueezy doesn't have a separate customer creation API
	// Customers are created automatically during the checkout process
	return "", nil
}

// stringToInt converts a string to int, returns 0 on error
func stringToInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
