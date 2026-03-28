package lemonsqueezy

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	lemonsqueezy "github.com/NdoleStudio/lemonsqueezy-go"

	"github.com/anthropics/agentsmesh/backend/internal/service/payment/types"
)

// CancelSubscription cancels a LemonSqueezy subscription
func (p *Provider) CancelSubscription(ctx context.Context, subscriptionID string, immediate bool) error {
	if immediate {
		// Cancel immediately
		_, _, err := p.client.Subscriptions.Cancel(ctx, subscriptionID)
		if err != nil {
			slog.Error("failed to cancel lemonsqueezy subscription immediately", "subscription_id", subscriptionID, "error", err)
			return fmt.Errorf("failed to cancel subscription: %w", err)
		}
	} else {
		// Cancel at period end (update subscription with cancelled=true)
		_, _, err := p.client.Subscriptions.Update(ctx, &lemonsqueezy.SubscriptionUpdateParams{
			ID: subscriptionID,
			Attributes: lemonsqueezy.SubscriptionUpdateParamsAttributes{
				Cancelled: true,
			},
		})
		if err != nil {
			slog.Error("failed to set lemonsqueezy cancel at period end", "subscription_id", subscriptionID, "error", err)
			return fmt.Errorf("failed to set cancel at period end: %w", err)
		}
	}
	slog.Info("lemonsqueezy subscription canceled", "subscription_id", subscriptionID, "immediate", immediate)
	return nil
}

// GetCustomerPortalURL returns a URL for the customer to manage their billing
func (p *Provider) GetCustomerPortalURL(ctx context.Context, req *types.CustomerPortalRequest) (*types.CustomerPortalResponse, error) {
	// LemonSqueezy uses customer portal URLs from subscription data
	// The SubscriptionID is required to get the portal URL
	subscriptionID := req.SubscriptionID
	if subscriptionID == "" {
		// Fallback: some callers may pass subscription ID as CustomerID for backwards compatibility
		subscriptionID = req.CustomerID
	}
	if subscriptionID == "" {
		return nil, fmt.Errorf("subscription_id is required for LemonSqueezy customer portal")
	}

	// Get customer portal URL from the subscription
	sub, _, err := p.client.Subscriptions.Get(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Use the customer portal URL or update payment method URL
	portalURL := sub.Data.Attributes.Urls.CustomerPortal
	if portalURL == "" {
		portalURL = sub.Data.Attributes.Urls.UpdatePaymentMethod
	}

	return &types.CustomerPortalResponse{
		URL: portalURL,
	}, nil
}

// UpdateSubscriptionSeats updates the seat count for a subscription
// Note: LemonSqueezy uses subscription items for quantity management
func (p *Provider) UpdateSubscriptionSeats(ctx context.Context, subscriptionID string, seats int) error {
	// Get subscription to find the first subscription item
	sub, _, err := p.client.Subscriptions.Get(ctx, subscriptionID)
	if err != nil {
		slog.Error("failed to get lemonsqueezy subscription for seat update", "subscription_id", subscriptionID, "error", err)
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Get the first subscription item ID
	if sub.Data.Attributes.FirstSubscriptionItem == nil {
		return fmt.Errorf("subscription has no subscription items")
	}

	itemID := strconv.Itoa(sub.Data.Attributes.FirstSubscriptionItem.ID)

	// Update the subscription item quantity
	_, _, err = p.client.SubscriptionItems.Update(ctx, &lemonsqueezy.SubscriptionItemUpdateParams{
		ID: itemID,
		Attributes: lemonsqueezy.SubscriptionItemUpdateParamsAttributes{
			Quantity: seats,
		},
	})
	if err != nil {
		slog.Error("failed to update lemonsqueezy subscription seats", "subscription_id", subscriptionID, "seats", seats, "error", err)
		return fmt.Errorf("failed to update subscription seats: %w", err)
	}

	slog.Info("lemonsqueezy subscription seats updated", "subscription_id", subscriptionID, "seats", seats)
	return nil
}

// UpdateSubscriptionPlan changes the subscription to a new plan variant
func (p *Provider) UpdateSubscriptionPlan(ctx context.Context, subscriptionID string, newVariantID string) error {
	variantID, err := strconv.Atoi(newVariantID)
	if err != nil {
		return fmt.Errorf("invalid variant ID: %w", err)
	}

	_, _, err = p.client.Subscriptions.Update(ctx, &lemonsqueezy.SubscriptionUpdateParams{
		ID: subscriptionID,
		Attributes: lemonsqueezy.SubscriptionUpdateParamsAttributes{
			VariantID: variantID,
		},
	})
	if err != nil {
		slog.Error("failed to update lemonsqueezy subscription plan", "subscription_id", subscriptionID, "variant_id", newVariantID, "error", err)
		return fmt.Errorf("failed to update subscription plan: %w", err)
	}

	slog.Info("lemonsqueezy subscription plan updated", "subscription_id", subscriptionID, "variant_id", newVariantID)
	return nil
}

// GetSubscription retrieves subscription details
func (p *Provider) GetSubscription(ctx context.Context, subscriptionID string) (*types.SubscriptionDetails, error) {
	sub, _, err := p.client.Subscriptions.Get(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	renewsAt := sub.Data.Attributes.RenewsAt
	result := &types.SubscriptionDetails{
		ID:                subscriptionID,
		CustomerID:        strconv.Itoa(sub.Data.Attributes.CustomerID),
		Status:            sub.Data.Attributes.Status,
		CurrentPeriodEnd:  renewsAt,
		CancelAtPeriodEnd: sub.Data.Attributes.Cancelled,
	}

	// Calculate CurrentPeriodStart based on billing interval.
	// LemonSqueezy doesn't provide period_start directly, so we infer it from
	// the gap between created_at and renews_at. A gap > 6 months indicates yearly billing.
	// NOTE: BillingAnchor is the day-of-month for billing (1-31), NOT the interval.
	createdAt := sub.Data.Attributes.CreatedAt
	if !renewsAt.IsZero() && !createdAt.IsZero() {
		monthsGap := (renewsAt.Year()-createdAt.Year())*12 + int(renewsAt.Month()-createdAt.Month())
		if monthsGap > 6 {
			// Yearly billing: period start is 1 year before renews_at
			result.CurrentPeriodStart = renewsAt.AddDate(-1, 0, 0)
		} else {
			// Monthly billing (most common)
			result.CurrentPeriodStart = renewsAt.AddDate(0, -1, 0)
		}
	} else if !renewsAt.IsZero() {
		// No created_at available, default to monthly
		result.CurrentPeriodStart = renewsAt.AddDate(0, -1, 0)
	} else if !createdAt.IsZero() {
		// Fallback to created_at if renews_at is zero value
		result.CurrentPeriodStart = createdAt
	}

	// Get seats from first subscription item
	if sub.Data.Attributes.FirstSubscriptionItem != nil {
		result.Seats = sub.Data.Attributes.FirstSubscriptionItem.Quantity
	}

	return result, nil
}
