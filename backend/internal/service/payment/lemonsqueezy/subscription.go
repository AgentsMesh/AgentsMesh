package lemonsqueezy

import (
	"context"
	"fmt"
	"strconv"

	lemonsqueezy "github.com/NdoleStudio/lemonsqueezy-go"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment/types"
)

// CancelSubscription cancels a LemonSqueezy subscription
func (p *Provider) CancelSubscription(ctx context.Context, subscriptionID string, immediate bool) error {
	if immediate {
		// Cancel immediately
		_, _, err := p.client.Subscriptions.Cancel(ctx, subscriptionID)
		if err != nil {
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
			return fmt.Errorf("failed to set cancel at period end: %w", err)
		}
	}
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
		return fmt.Errorf("failed to update subscription seats: %w", err)
	}

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

	// Calculate CurrentPeriodStart based on billing interval
	// LemonSqueezy doesn't provide period_start directly, so we calculate it from RenewsAt
	if !renewsAt.IsZero() {
		billingInterval := sub.Data.Attributes.BillingAnchor
		// Default to monthly if billing anchor not specified
		if billingInterval == 0 {
			// Check variant interval from subscription item
			if sub.Data.Attributes.FirstSubscriptionItem != nil {
				// Assume monthly if we can't determine
				result.CurrentPeriodStart = renewsAt.AddDate(0, -1, 0)
			}
		} else if billingInterval == 12 {
			// Yearly
			result.CurrentPeriodStart = renewsAt.AddDate(-1, 0, 0)
		} else {
			// Monthly (most common)
			result.CurrentPeriodStart = renewsAt.AddDate(0, -1, 0)
		}
	} else {
		// Fallback to created_at if renews_at is zero value
		result.CurrentPeriodStart = sub.Data.Attributes.CreatedAt
	}

	// Get seats from first subscription item
	if sub.Data.Attributes.FirstSubscriptionItem != nil {
		result.Seats = sub.Data.Attributes.FirstSubscriptionItem.Quantity
	}

	return result, nil
}
