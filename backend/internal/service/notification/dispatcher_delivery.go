package notification

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	notifDomain "github.com/anthropics/agentsmesh/backend/internal/domain/notification"
)

// notificationWireEvent mirrors the frontend RealtimeEvent interface for backward compatibility.
type notificationWireEvent struct {
	Type           string          `json:"type"`
	Category       string          `json:"category"`
	OrganizationID int64           `json:"organization_id"`
	TargetUserID   *int64          `json:"target_user_id,omitempty"`
	EntityType     string          `json:"entity_type,omitempty"`
	EntityID       string          `json:"entity_id,omitempty"`
	Data           json.RawMessage `json:"data"`
	Timestamp      int64           `json:"timestamp"`
}

type notificationPayload struct {
	Source   string          `json:"source"`
	Title    string          `json:"title"`
	Body     string          `json:"body"`
	Link     string          `json:"link,omitempty"`
	Priority string          `json:"priority"`
	Channels map[string]bool `json:"channels"`
}

func (d *Dispatcher) resolveRecipients(ctx context.Context, req *notifDomain.NotificationRequest) []int64 {
	recipientIDs := req.RecipientUserIDs
	if req.RecipientResolver != "" {
		resolved, err := d.resolve(ctx, req.RecipientResolver)
		if err != nil {
			slog.Error("failed to resolve notification recipients", "resolver", req.RecipientResolver, "error", err)
		} else {
			recipientIDs = resolved
		}
	}

	if len(recipientIDs) == 0 || len(req.ExcludeUserIDs) == 0 {
		return recipientIDs
	}

	excluded := make(map[int64]bool, len(req.ExcludeUserIDs))
	for _, id := range req.ExcludeUserIDs {
		excluded[id] = true
	}
	filtered := recipientIDs[:0]
	for _, id := range recipientIDs {
		if !excluded[id] {
			filtered = append(filtered, id)
		}
	}
	return filtered
}

func (d *Dispatcher) buildChannels(pref *notifDomain.Preference, priority string) map[string]bool {
	channels := make(map[string]bool, len(notifDomain.BuiltinClientChannels))
	for ch := range notifDomain.BuiltinClientChannels {
		if priority == notifDomain.PriorityHigh && pref.IsMuted {
			channels[ch] = true
		} else {
			channels[ch] = !pref.IsMuted && pref.IsChannelEnabled(ch)
		}
	}
	return channels
}

func (d *Dispatcher) pushToUser(ctx context.Context, userID int64, req *notifDomain.NotificationRequest, priority string, channels map[string]bool) {
	payload := notificationPayload{
		Source:   req.Source,
		Title:    req.Title,
		Body:     req.Body,
		Link:     req.Link,
		Priority: priority,
		Channels: channels,
	}
	payloadData, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal notification payload", "error", err)
		return
	}

	uid := userID
	wireEvent := notificationWireEvent{
		Type:           "notification",
		Category:       "notification",
		OrganizationID: req.OrganizationID,
		TargetUserID:   &uid,
		EntityType:     "notification",
		EntityID:       req.SourceEntityID,
		Data:           payloadData,
		Timestamp:      time.Now().UnixMilli(),
	}
	data, err := json.Marshal(wireEvent)
	if err != nil {
		slog.Error("failed to marshal notification wire event", "error", err)
		return
	}

	if err := d.pusher.PushToUser(ctx, userID, data); err != nil {
		slog.Error("failed to push notification", "user_id", userID, "error", err)
	}
}

func (d *Dispatcher) fireDeliveryHandlers(ctx context.Context, userID int64, pref *notifDomain.Preference, req *notifDomain.NotificationRequest) {
	for ch, handler := range d.deliveryHandlers {
		if pref.IsChannelEnabled(ch) {
			go func(h notifDomain.DeliveryHandler, uid int64) {
				if err := h.Deliver(ctx, uid, req); err != nil {
					slog.Error("delivery handler failed", "channel", h.Channel(), "user_id", uid, "error", err)
				}
			}(handler, userID)
		}
	}
}
