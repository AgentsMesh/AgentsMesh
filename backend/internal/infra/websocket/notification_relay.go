package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type notificationRelayMessage struct {
	InstanceID string          `json:"instance_id"`
	UserID     int64           `json:"user_id"`
	Data       json.RawMessage `json:"data"`
}

const notificationRelayChannel = "notif:push"

// NotificationRelay delivers notification payloads to WebSocket clients.
// Local delivery via Hub is immediate; Redis relay handles cross-instance sync.
// Implements notification.RealtimePusher.
type NotificationRelay struct {
	hub         *Hub
	redisClient *redis.Client
	instanceID  string
	logger      *slog.Logger
}

func NewNotificationRelay(hub *Hub, redisClient *redis.Client, logger *slog.Logger) *NotificationRelay {
	if logger == nil {
		logger = slog.Default()
	}
	hostname, _ := os.Hostname()
	return &NotificationRelay{
		hub:         hub,
		redisClient: redisClient,
		instanceID:  fmt.Sprintf("%s-%s", hostname, uuid.New().String()[:8]),
		logger:      logger.With("component", "notification_relay"),
	}
}

func (r *NotificationRelay) PushToUser(ctx context.Context, userID int64, data []byte) error {
	r.hub.SendToUser(userID, data)

	if r.redisClient != nil {
		msg := notificationRelayMessage{
			InstanceID: r.instanceID,
			UserID:     userID,
			Data:       data,
		}
		payload, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("marshal relay message: %w", err)
		}
		if err := r.redisClient.Publish(ctx, notificationRelayChannel, payload).Err(); err != nil {
			r.logger.Error("failed to publish notification relay", "error", err, "user_id", userID)
		}
	}
	return nil
}

// StartSubscriber must be called once at startup. It blocks until ctx is cancelled.
func (r *NotificationRelay) StartSubscriber(ctx context.Context) {
	if r.redisClient == nil {
		r.logger.Warn("Redis not available, notification relay disabled")
		return
	}

	go func() {
		pubsub := r.redisClient.Subscribe(ctx, notificationRelayChannel)
		defer pubsub.Close()

		r.logger.Info("notification relay subscriber started")
		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				var relay notificationRelayMessage
				if err := json.Unmarshal([]byte(msg.Payload), &relay); err != nil {
					r.logger.Error("failed to unmarshal relay message", "error", err)
					continue
				}
				if relay.InstanceID == r.instanceID {
					continue
				}
				r.hub.SendToUser(relay.UserID, relay.Data)
			}
		}
	}()
}
