package lemonsqueezy

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	lemonsqueezy "github.com/NdoleStudio/lemonsqueezy-go"

	"github.com/anthropics/agentsmesh/backend/internal/service/payment/types"
)

func (p *Provider) CancelSubscription(ctx context.Context, subscriptionID string, immediate bool) error {
	if immediate {
		_, _, err := p.client.Subscriptions.Cancel(ctx, subscriptionID)
		if err != nil {
			slog.ErrorContext(ctx, "failed to cancel lemonsqueezy subscription immediately", "subscription_id", subscriptionID, "error", err)
			return fmt.Errorf("failed to cancel subscription: %w", err)
		}
	} else {
		_, _, err := p.client.Subscriptions.Update(ctx, &lemonsqueezy.SubscriptionUpdateParams{
			ID: subscriptionID,
			Attributes: lemonsqueezy.SubscriptionUpdateParamsAttributes{
				Cancelled: true,
			},
		})
		if err != nil {
			slog.ErrorContext(ctx, "failed to set lemonsqueezy cancel at period end", "subscription_id", subscriptionID, "error", err)
			return fmt.Errorf("failed to set cancel at period end: %w", err)
		}
	}
	slog.InfoContext(ctx, "lemonsqueezy subscription canceled", "subscription_id", subscriptionID, "immediate", immediate)
	return nil
}

func (p *Provider) GetCustomerPortalURL(ctx context.Context, req *types.CustomerPortalRequest) (*types.CustomerPortalResponse, error) {
	subscriptionID := req.SubscriptionID
	if subscriptionID == "" {
		return nil, fmt.Errorf("subscription_id is required for LemonSqueezy customer portal")
	}

	sub, _, err := p.client.Subscriptions.Get(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	portalURL := sub.Data.Attributes.Urls.CustomerPortal
	if portalURL == "" {
		portalURL = sub.Data.Attributes.Urls.UpdatePaymentMethod
	}

	return &types.CustomerPortalResponse{
		URL: portalURL,
	}, nil
}

func (p *Provider) UpdateSubscriptionSeats(ctx context.Context, subscriptionID string, seats int) error {
	sub, _, err := p.client.Subscriptions.Get(ctx, subscriptionID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get lemonsqueezy subscription for seat update", "subscription_id", subscriptionID, "error", err)
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	if sub.Data.Attributes.FirstSubscriptionItem == nil {
		return fmt.Errorf("subscription has no subscription items")
	}

	itemID := strconv.Itoa(sub.Data.Attributes.FirstSubscriptionItem.ID)

	_, _, err = p.client.SubscriptionItems.Update(ctx, &lemonsqueezy.SubscriptionItemUpdateParams{
		ID: itemID,
		Attributes: lemonsqueezy.SubscriptionItemUpdateParamsAttributes{
			Quantity: seats,
		},
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to update lemonsqueezy subscription seats", "subscription_id", subscriptionID, "seats", seats, "error", err)
		return fmt.Errorf("failed to update subscription seats: %w", err)
	}

	slog.InfoContext(ctx, "lemonsqueezy subscription seats updated", "subscription_id", subscriptionID, "seats", seats)
	return nil
}

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
		slog.ErrorContext(ctx, "failed to update lemonsqueezy subscription plan", "subscription_id", subscriptionID, "variant_id", newVariantID, "error", err)
		return fmt.Errorf("failed to update subscription plan: %w", err)
	}

	slog.InfoContext(ctx, "lemonsqueezy subscription plan updated", "subscription_id", subscriptionID, "variant_id", newVariantID)
	return nil
}

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

	// LS doesn't expose period_start. Infer from created→renews gap: >6 months ⇒ yearly.
	// BillingAnchor is day-of-month (1-31), NOT the interval — do not use it here.
	createdAt := sub.Data.Attributes.CreatedAt
	if !renewsAt.IsZero() && !createdAt.IsZero() {
		monthsGap := (renewsAt.Year()-createdAt.Year())*12 + int(renewsAt.Month()-createdAt.Month())
		if monthsGap > 6 {
			result.CurrentPeriodStart = renewsAt.AddDate(-1, 0, 0)
		} else {
			result.CurrentPeriodStart = renewsAt.AddDate(0, -1, 0)
		}
	} else if !renewsAt.IsZero() {
		result.CurrentPeriodStart = renewsAt.AddDate(0, -1, 0)
	} else if !createdAt.IsZero() {
		result.CurrentPeriodStart = createdAt
	}

	if sub.Data.Attributes.FirstSubscriptionItem != nil {
		result.Seats = sub.Data.Attributes.FirstSubscriptionItem.Quantity
	}

	return result, nil
}
