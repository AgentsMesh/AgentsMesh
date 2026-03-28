package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/stripe/stripe-go/v76"
	checkoutsession "github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/webhook"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment/types"
)

// Provider implements payment.SubscriptionProvider for Stripe
type Provider struct {
	secretKey     string
	webhookSecret string
}

// NewProvider creates a new Stripe provider
func NewProvider(cfg *config.StripeConfig) *Provider {
	stripe.Key = cfg.SecretKey
	return &Provider{
		secretKey:     cfg.SecretKey,
		webhookSecret: cfg.WebhookSecret,
	}
}

// GetProviderName returns the provider name
func (p *Provider) GetProviderName() string {
	return billing.PaymentProviderStripe
}

// CreateCheckoutSession creates a Stripe Checkout session
func (p *Provider) CreateCheckoutSession(ctx context.Context, req *types.CheckoutRequest) (*types.CheckoutResponse, error) {
	// Build line items
	lineItems := []*stripe.CheckoutSessionLineItemParams{
		{
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency: stripe.String(req.Currency),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name: stripe.String(fmt.Sprintf("Subscription - %s", req.BillingCycle)),
				},
				UnitAmount: stripe.Int64(int64(req.ActualAmount * 100)), // Convert to cents
			},
			Quantity: stripe.Int64(int64(req.Seats)),
		},
	}

	// Determine mode based on order type
	mode := stripe.CheckoutSessionModePayment
	if req.OrderType == billing.OrderTypeSubscription || req.OrderType == billing.OrderTypeRenewal {
		mode = stripe.CheckoutSessionModeSubscription
	}

	// Build metadata
	metadata := map[string]string{
		"organization_id": fmt.Sprintf("%d", req.OrganizationID),
		"user_id":         fmt.Sprintf("%d", req.UserID),
		"order_type":      req.OrderType,
		"billing_cycle":   req.BillingCycle,
		"seats":           fmt.Sprintf("%d", req.Seats),
	}
	for k, v := range req.Metadata {
		metadata[k] = v
	}

	params := &stripe.CheckoutSessionParams{
		Mode:          stripe.String(string(mode)),
		LineItems:     lineItems,
		SuccessURL:    stripe.String(req.SuccessURL),
		CancelURL:     stripe.String(req.CancelURL),
		CustomerEmail: stripe.String(req.UserEmail),
		Metadata:      metadata,
		ExpiresAt:     stripe.Int64(time.Now().Add(30 * time.Minute).Unix()),
	}

	// Add idempotency key if provided
	if req.IdempotencyKey != "" {
		params.SetIdempotencyKey(req.IdempotencyKey)
	}

	// For subscription mode, configure subscription data
	if mode == stripe.CheckoutSessionModeSubscription {
		params.SubscriptionData = &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: metadata,
		}
	}

	sess, err := checkoutsession.New(params)
	if err != nil {
		slog.Error("failed to create Stripe checkout session", "org_id", req.OrganizationID, "order_type", req.OrderType, "error", err)
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	slog.Info("Stripe checkout session created", "session_id", sess.ID, "org_id", req.OrganizationID, "order_type", req.OrderType)

	return &types.CheckoutResponse{
		SessionID:       sess.ID,
		SessionURL:      sess.URL,
		OrderNo:         req.IdempotencyKey, // Will be set by caller
		ExternalOrderNo: sess.ID,
		ExpiresAt:       time.Unix(sess.ExpiresAt, 0),
	}, nil
}

// GetCheckoutStatus checks the status of a checkout session
func (p *Provider) GetCheckoutStatus(ctx context.Context, sessionID string) (string, error) {
	sess, err := checkoutsession.Get(sessionID, nil)
	if err != nil {
		slog.Error("failed to get Stripe checkout session", "session_id", sessionID, "error", err)
		return "", fmt.Errorf("failed to get checkout session: %w", err)
	}

	switch sess.Status {
	case stripe.CheckoutSessionStatusComplete:
		return billing.OrderStatusSucceeded, nil
	case stripe.CheckoutSessionStatusExpired:
		return billing.OrderStatusCanceled, nil
	case stripe.CheckoutSessionStatusOpen:
		return billing.OrderStatusPending, nil
	default:
		return billing.OrderStatusPending, nil
	}
}

// HandleWebhook parses and validates a Stripe webhook
func (p *Provider) HandleWebhook(ctx context.Context, payload []byte, signature string) (*types.WebhookEvent, error) {
	event, err := webhook.ConstructEvent(payload, signature, p.webhookSecret)
	if err != nil {
		slog.Error("failed to verify Stripe webhook signature", "error", err)
		return nil, fmt.Errorf("failed to verify webhook signature: %w", err)
	}

	result := &types.WebhookEvent{
		EventID:   event.ID,
		EventType: string(event.Type),
		Provider:  billing.PaymentProviderStripe,
	}

	// Parse event data based on type
	switch string(event.Type) {
	case billing.WebhookEventCheckoutCompleted:
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			slog.Error("failed to parse Stripe checkout session", "event_id", event.ID, "error", err)
			return nil, fmt.Errorf("failed to parse checkout session: %w", err)
		}
		result.ExternalOrderNo = sess.ID
		if sess.Customer != nil {
			result.CustomerID = sess.Customer.ID
		}
		if sess.Subscription != nil {
			result.SubscriptionID = sess.Subscription.ID
		}
		result.Amount = float64(sess.AmountTotal) / 100
		result.Currency = string(sess.Currency)
		result.Status = billing.OrderStatusSucceeded
		if sess.Metadata != nil {
			result.OrderNo = sess.Metadata["order_no"]
		}

	case billing.WebhookEventInvoicePaid:
		var inv stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
			slog.Error("failed to parse Stripe invoice", "event_id", event.ID, "error", err)
			return nil, fmt.Errorf("failed to parse invoice: %w", err)
		}
		result.ExternalOrderNo = inv.ID
		if inv.Customer != nil {
			result.CustomerID = inv.Customer.ID
		}
		if inv.Subscription != nil {
			result.SubscriptionID = inv.Subscription.ID
		}
		result.Amount = float64(inv.AmountPaid) / 100
		result.Currency = string(inv.Currency)
		result.Status = billing.OrderStatusSucceeded

	case billing.WebhookEventInvoiceFailed:
		var inv stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
			slog.Error("failed to parse Stripe invoice", "event_id", event.ID, "error", err)
			return nil, fmt.Errorf("failed to parse invoice: %w", err)
		}
		result.ExternalOrderNo = inv.ID
		if inv.Customer != nil {
			result.CustomerID = inv.Customer.ID
		}
		if inv.Subscription != nil {
			result.SubscriptionID = inv.Subscription.ID
		}
		result.Amount = float64(inv.AmountDue) / 100
		result.Currency = string(inv.Currency)
		result.Status = billing.OrderStatusFailed
		if inv.LastFinalizationError != nil {
			result.FailedReason = inv.LastFinalizationError.Msg
		}

	case billing.WebhookEventSubscriptionDeleted:
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			slog.Error("failed to parse Stripe subscription", "event_id", event.ID, "error", err)
			return nil, fmt.Errorf("failed to parse subscription: %w", err)
		}
		result.SubscriptionID = sub.ID
		if sub.Customer != nil {
			result.CustomerID = sub.Customer.ID
		}
		result.Status = billing.SubscriptionStatusCanceled

	case billing.WebhookEventSubscriptionUpdated:
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			slog.Error("failed to parse Stripe subscription", "event_id", event.ID, "error", err)
			return nil, fmt.Errorf("failed to parse subscription: %w", err)
		}
		result.SubscriptionID = sub.ID
		if sub.Customer != nil {
			result.CustomerID = sub.Customer.ID
		}
		result.Status = string(sub.Status)
	}

	// Store raw payload
	result.RawPayload = make(map[string]interface{})
	_ = json.Unmarshal(event.Data.Raw, &result.RawPayload)

	return result, nil
}

