package billing

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

func (s *Service) HandleSubscriptionUpdated(c *gin.Context, event *payment.WebhookEvent) (retErr error) {
	ctx := c.Request.Context()

	if event.SubscriptionID == "" {
		return nil
	}

	if err := s.CheckAndMarkWebhookProcessed(ctx, event.EventID, event.Provider, event.EventType); err != nil {
		if errors.Is(err, ErrWebhookAlreadyProcessed) {
			return nil
		}
		return err
	}
	// Roll back idempotency mark on handler failure so the event can be retried on next delivery.
	defer func() {
		if retErr != nil {
			s.DeleteWebhookProcessedMark(ctx, event.EventID, event.Provider)
		}
	}()

	sub, err := s.findSubscriptionByProviderID(ctx, event.Provider, event.SubscriptionID)
	if err != nil {
		// P0 #2: Race condition fallback — subscription_created may arrive before order_created,
		// leaving LemonSqueezySubscriptionID unset. Recover via customer_id lookup.
		if event.Provider == billing.PaymentProviderLemonSqueezy && event.CustomerID != "" {
			sub, err = s.findAndLinkLSSubscription(ctx, event)
			if err != nil {
				slog.WarnContext(c.Request.Context(), "subscription not found for provider",
				"provider", event.Provider, "subscription_id", event.SubscriptionID, "customer_id", event.CustomerID)
				return nil
			}
		} else {
			return nil
		}
	}

	if event.Provider == billing.PaymentProviderLemonSqueezy {
		mappedStatus := billing.MapLSStatusToInternal(event.Status)
		switch mappedStatus {
		case billing.SubscriptionStatusActive, billing.SubscriptionStatusTrialing,
			billing.SubscriptionStatusPaused, billing.SubscriptionStatusPastDue,
			billing.SubscriptionStatusFrozen, billing.SubscriptionStatusCanceled,
			billing.SubscriptionStatusExpired:
			sub.Status = mappedStatus
		default:
			slog.WarnContext(c.Request.Context(), "unknown LemonSqueezy subscription status",
				"status", event.Status, "subscription_id", event.SubscriptionID)
		}
	} else {
		switch event.Status {
		case "active":
			sub.Status = billing.SubscriptionStatusActive
			sub.FrozenAt = nil
		case "past_due":
			sub.Status = billing.SubscriptionStatusPastDue
		case "canceled", "cancelled":
			sub.Status = billing.SubscriptionStatusCanceled
		case "trialing":
			sub.Status = billing.SubscriptionStatusTrialing
		case "paused":
			sub.Status = billing.SubscriptionStatusPaused
		case "expired":
			sub.Status = billing.SubscriptionStatusExpired
		default:
			slog.WarnContext(c.Request.Context(), "unknown subscription status from provider",
				"status", event.Status, "provider", event.Provider, "subscription_id", event.SubscriptionID)
		}
	}

	if sub.Status == billing.SubscriptionStatusActive {
		sub.FrozenAt = nil
	}

	if sub.Status == billing.SubscriptionStatusFrozen && sub.FrozenAt == nil {
		now := time.Now()
		sub.FrozenAt = &now
	}

	if event.Seats > 0 && event.Seats != sub.SeatCount {
		sub.SeatCount = event.Seats
	}

	var planName *string
	if event.VariantID != "" {
		if plan, err := s.findPlanByVariantID(ctx, event.VariantID); err == nil && plan != nil && plan.ID != sub.PlanID {
			sub.PlanID = plan.ID
			sub.DowngradeToPlan = nil
			planName = &plan.Name
		}
	}

	if err := s.repo.SaveSubscription(ctx, sub); err != nil {
		return err
	}

	s.syncOrganizationSubscription(ctx, sub.OrganizationID, planName, &sub.Status)
	return nil
}

// findAndLinkLSSubscription is the race-recovery path for P0 #2 when
// subscription_created arrived before order_created.
func (s *Service) findAndLinkLSSubscription(ctx context.Context, event *payment.WebhookEvent) (*billing.Subscription, error) {
	sub, err := s.repo.FindSubscriptionByLSCustomerID(ctx, event.CustomerID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrSubscriptionNotFound
	}

	if sub.LemonSqueezySubscriptionID == nil {
		sub.LemonSqueezySubscriptionID = &event.SubscriptionID
		slog.InfoContext(ctx, "linked LS subscription via customer ID (race condition recovery)",
			"subscription_id", event.SubscriptionID, "org_id", sub.OrganizationID, "customer_id", event.CustomerID)
	}

	return sub, nil
}
