package billing

// LemonSqueezy webhook event type constants
// See: https://docs.lemonsqueezy.com/help/webhooks
const (
	WebhookEventLSOrderCreated = "order_created"

	WebhookEventLSSubscriptionCreated   = "subscription_created"
	WebhookEventLSSubscriptionUpdated   = "subscription_updated"
	WebhookEventLSSubscriptionCancelled = "subscription_cancelled"
	WebhookEventLSSubscriptionPaused    = "subscription_paused"
	WebhookEventLSSubscriptionResumed   = "subscription_resumed"
	WebhookEventLSSubscriptionExpired   = "subscription_expired"

	WebhookEventLSPaymentSuccess = "subscription_payment_success"
	WebhookEventLSPaymentFailed  = "subscription_payment_failed"
)

const (
	LSSubscriptionStatusOnTrial   = "on_trial"
	LSSubscriptionStatusActive    = "active"
	LSSubscriptionStatusPaused    = "paused"
	LSSubscriptionStatusPastDue   = "past_due"
	LSSubscriptionStatusUnpaid    = "unpaid"
	LSSubscriptionStatusCancelled = "cancelled"
	LSSubscriptionStatusExpired   = "expired"
)

func MapLSStatusToInternal(lsStatus string) string {
	switch lsStatus {
	case LSSubscriptionStatusOnTrial:
		return SubscriptionStatusTrialing
	case LSSubscriptionStatusActive:
		return SubscriptionStatusActive
	case LSSubscriptionStatusPaused:
		return SubscriptionStatusPaused
	case LSSubscriptionStatusPastDue:
		return SubscriptionStatusPastDue
	case LSSubscriptionStatusUnpaid:
		return SubscriptionStatusFrozen
	case LSSubscriptionStatusCancelled:
		return SubscriptionStatusCanceled
	case LSSubscriptionStatusExpired:
		return SubscriptionStatusExpired
	default:
		return lsStatus
	}
}
