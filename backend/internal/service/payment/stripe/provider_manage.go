package stripe

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/stripe/stripe-go/v76"
	portalsession "github.com/stripe/stripe-go/v76/billingportal/session"
	"github.com/stripe/stripe-go/v76/customer"
	"github.com/stripe/stripe-go/v76/refund"
	"github.com/stripe/stripe-go/v76/subscription"

	checkoutsession "github.com/stripe/stripe-go/v76/checkout/session"

	"github.com/anthropics/agentsmesh/backend/internal/service/payment/types"
)

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
		slog.Error("failed to create stripe refund", "order_no", req.OrderNo, "amount", req.Amount, "error", err)
		return nil, fmt.Errorf("failed to create refund: %w", err)
	}

	slog.Info("stripe refund created", "refund_id", r.ID, "amount", float64(r.Amount)/100, "currency", string(r.Currency))
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
			slog.Error("failed to cancel stripe subscription immediately", "subscription_id", subscriptionID, "error", err)
			return fmt.Errorf("failed to cancel subscription: %w", err)
		}
	} else {
		_, err := subscription.Update(subscriptionID, &stripe.SubscriptionParams{
			CancelAtPeriodEnd: stripe.Bool(true),
		})
		if err != nil {
			slog.Error("failed to set stripe cancel at period end", "subscription_id", subscriptionID, "error", err)
			return fmt.Errorf("failed to set cancel at period end: %w", err)
		}
	}
	slog.Info("stripe subscription canceled", "subscription_id", subscriptionID, "immediate", immediate)
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
		slog.Error("failed to create stripe customer", "email", email, "error", err)
		return "", fmt.Errorf("failed to create customer: %w", err)
	}

	slog.Info("stripe customer created", "customer_id", c.ID, "email", email)
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
		slog.Error("failed to get stripe subscription for seat update", "subscription_id", subscriptionID, "error", err)
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
		slog.Error("failed to update stripe subscription seats", "subscription_id", subscriptionID, "seats", seats, "error", err)
		return fmt.Errorf("failed to update subscription seats: %w", err)
	}

	slog.Info("stripe subscription seats updated", "subscription_id", subscriptionID, "seats", seats)
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
