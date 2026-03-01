package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v76"
	portalsession "github.com/stripe/stripe-go/v76/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/customer"
	"github.com/stripe/stripe-go/v76/refund"
	"github.com/stripe/stripe-go/v76/subscription"
	"github.com/stripe/stripe-go/v76/webhook"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/config"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment/types"
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
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

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

// RefundPayment initiates a refund
func (p *Provider) RefundPayment(ctx context.Context, req *types.RefundRequest) (*types.RefundResponse, error) {
	params := &stripe.RefundParams{
		Amount: stripe.Int64(int64(req.Amount * 100)),
	}

	// Set reason if provided
	if req.Reason != "" {
		params.Reason = stripe.String(req.Reason)
	}

	// Try to find the payment intent from checkout session
	if req.ExternalOrderNo != "" {
		sess, err := checkoutsession.Get(req.ExternalOrderNo, nil)
		if err == nil && sess.PaymentIntent != nil {
			params.PaymentIntent = stripe.String(sess.PaymentIntent.ID)
		}
	}

	if req.IdempotencyKey != "" {
		params.SetIdempotencyKey(req.IdempotencyKey)
	}

	r, err := refund.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create refund: %w", err)
	}

	return &types.RefundResponse{
		RefundID: r.ID,
		Status:   string(r.Status),
		Amount:   float64(r.Amount) / 100,
		Currency: string(r.Currency),
	}, nil
}

// CancelSubscription cancels a Stripe subscription
func (p *Provider) CancelSubscription(ctx context.Context, subscriptionID string, immediate bool) error {
	if immediate {
		_, err := subscription.Cancel(subscriptionID, nil)
		if err != nil {
			return fmt.Errorf("failed to cancel subscription: %w", err)
		}
	} else {
		_, err := subscription.Update(subscriptionID, &stripe.SubscriptionParams{
			CancelAtPeriodEnd: stripe.Bool(true),
		})
		if err != nil {
			return fmt.Errorf("failed to set cancel at period end: %w", err)
		}
	}
	return nil
}

// CreateCustomer creates a Stripe customer
func (p *Provider) CreateCustomer(ctx context.Context, email string, name string, metadata map[string]string) (string, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
	}
	if metadata != nil {
		params.Metadata = metadata
	}

	c, err := customer.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create customer: %w", err)
	}

	return c.ID, nil
}

// GetCustomerPortalURL returns a URL for the customer billing portal
func (p *Provider) GetCustomerPortalURL(ctx context.Context, req *types.CustomerPortalRequest) (*types.CustomerPortalResponse, error) {
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(req.CustomerID),
		ReturnURL: stripe.String(req.ReturnURL),
	}

	sess, err := portalsession.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create portal session: %w", err)
	}

	return &types.CustomerPortalResponse{
		URL: sess.URL,
	}, nil
}

// UpdateSubscriptionSeats updates the seat count for a subscription
func (p *Provider) UpdateSubscriptionSeats(ctx context.Context, subscriptionID string, seats int) error {
	// Get current subscription
	sub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	if len(sub.Items.Data) == 0 {
		return fmt.Errorf("subscription has no items")
	}

	// Update the first item's quantity
	_, err = subscription.Update(subscriptionID, &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:       stripe.String(sub.Items.Data[0].ID),
				Quantity: stripe.Int64(int64(seats)),
			},
		},
		ProrationBehavior: stripe.String(string(stripe.SubscriptionSchedulePhaseProrationBehaviorCreateProrations)),
	})
	if err != nil {
		return fmt.Errorf("failed to update subscription seats: %w", err)
	}

	return nil
}

// GetSubscription retrieves subscription details
func (p *Provider) GetSubscription(ctx context.Context, subscriptionID string) (*types.SubscriptionDetails, error) {
	sub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	result := &types.SubscriptionDetails{
		ID:                 sub.ID,
		Status:             string(sub.Status),
		CurrentPeriodStart: time.Unix(sub.CurrentPeriodStart, 0),
		CurrentPeriodEnd:   time.Unix(sub.CurrentPeriodEnd, 0),
		CancelAtPeriodEnd:  sub.CancelAtPeriodEnd,
	}

	if sub.Customer != nil {
		result.CustomerID = sub.Customer.ID
	}

	// Get seats from first item
	if len(sub.Items.Data) > 0 {
		result.Seats = int(sub.Items.Data[0].Quantity)
		if sub.Items.Data[0].Price != nil {
			result.PriceID = sub.Items.Data[0].Price.ID
		}
	}

	return result, nil
}
