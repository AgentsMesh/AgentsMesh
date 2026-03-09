package notification

import "context"

// DeliveryHandler is the server-side extension point for delivering notifications
// through external channels (e.g. email, APNS, Slack).
// Implementations are fire-and-forget — errors are logged but do not block dispatch.
type DeliveryHandler interface {
	// Channel returns the delivery channel name (e.g. "email", "apns").
	Channel() string
	// Deliver sends the notification to the specified user.
	Deliver(ctx context.Context, userID int64, req *NotificationRequest) error
}
