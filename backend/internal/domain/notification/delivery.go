package notification

import "context"

type DeliveryHandler interface {
	Channel() string
	Deliver(ctx context.Context, userID int64, req *NotificationRequest) error
}

type RealtimePusher interface {
	PushToUser(ctx context.Context, userID int64, data []byte) error
}
