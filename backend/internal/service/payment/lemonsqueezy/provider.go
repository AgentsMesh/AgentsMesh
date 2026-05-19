package lemonsqueezy

import (
	"context"
	"fmt"
	"strconv"

	lemonsqueezy "github.com/NdoleStudio/lemonsqueezy-go"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment/types"
)

type Provider struct {
	client        *lemonsqueezy.Client
	storeID       string
	webhookSecret string
}

func NewProvider(cfg *config.LemonSqueezyConfig) *Provider {
	client := lemonsqueezy.New(lemonsqueezy.WithAPIKey(cfg.APIKey))
	return &Provider{
		client:        client,
		storeID:       cfg.StoreID,
		webhookSecret: cfg.WebhookSecret,
	}
}

func (p *Provider) GetProviderName() string {
	return billing.PaymentProviderLemonSqueezy
}

func (p *Provider) GetCheckoutStatus(ctx context.Context, sessionID string) (string, error) {
	return billing.OrderStatusPending, nil
}

func (p *Provider) RefundPayment(ctx context.Context, req *types.RefundRequest) (*types.RefundResponse, error) {
	return nil, fmt.Errorf("refunds must be processed through the LemonSqueezy dashboard")
}

func (p *Provider) CreateCustomer(ctx context.Context, email string, name string, metadata map[string]string) (string, error) {
	return "", nil
}

func stringToInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
