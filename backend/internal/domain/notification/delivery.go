package notification

import "context"

// DeliveryHandler is the server-side extension point for external channels (email, APNS, Slack).
// Implementations are fire-and-forget — errors are logged but do not block dispatch.
type DeliveryHandler interface {
	Channel() string
	Deliver(ctx context.Context, userID int64, req *NotificationRequest) error
}

// RealtimePusher pushes notification payloads to connected WebSocket clients.
// Implementations handle both local delivery and cross-instance relay.
type RealtimePusher interface {
	PushToUser(ctx context.Context, userID int64, data []byte) error
}
