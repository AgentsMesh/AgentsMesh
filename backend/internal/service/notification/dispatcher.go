package notification

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	notifDomain "github.com/anthropics/agentsmesh/backend/internal/domain/notification"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

// Dispatcher routes notifications to users based on preferences
type Dispatcher struct {
	eventBus         *eventbus.EventBus
	prefStore        *PreferenceStore
	resolvers        map[string]RecipientResolver
	deliveryHandlers map[string]notifDomain.DeliveryHandler
}

// NewDispatcher creates a new notification dispatcher
func NewDispatcher(eventBus *eventbus.EventBus, prefStore *PreferenceStore) *Dispatcher {
	return &Dispatcher{
		eventBus:         eventBus,
		prefStore:        prefStore,
		resolvers:        make(map[string]RecipientResolver),
		deliveryHandlers: make(map[string]notifDomain.DeliveryHandler),
	}
}

// RegisterResolver registers a resolver for a given prefix (e.g. "pod_creator", "channel_members")
func (d *Dispatcher) RegisterResolver(prefix string, resolver RecipientResolver) {
	d.resolvers[prefix] = resolver
}

// RegisterDeliveryHandler registers a server-side delivery handler for a channel.
func (d *Dispatcher) RegisterDeliveryHandler(handler notifDomain.DeliveryHandler) {
	d.deliveryHandlers[handler.Channel()] = handler
}

// Dispatch sends a notification to resolved recipients, filtered by preferences
func (d *Dispatcher) Dispatch(ctx context.Context, req *notifDomain.NotificationRequest) error {
	recipientIDs := req.RecipientUserIDs
	if req.RecipientResolver != "" {
		resolved, err := d.resolve(ctx, req.RecipientResolver)
		if err != nil {
			// Log but continue — a transient DB error should not silently
			// discard all notifications. The caller can retry if needed.
			slog.Error("failed to resolve notification recipients",
				"resolver", req.RecipientResolver,
				"error", err,
			)
			// Fall through with whatever direct RecipientUserIDs exist
		} else {
			recipientIDs = resolved
		}
	}

	if len(recipientIDs) == 0 {
		return nil
	}

	// Filter out excluded users (e.g. message sender should not notify themselves)
	if len(req.ExcludeUserIDs) > 0 {
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
		recipientIDs = filtered
	}

	if len(recipientIDs) == 0 {
		return nil
	}

	priority := req.Priority
	if priority == "" {
		priority = notifDomain.PriorityNormal
	}

	for _, userID := range recipientIDs {
		pref := d.prefStore.GetPreference(ctx, userID, req.Source, req.SourceEntityID)

		// Skip muted users unless high priority
		if pref.IsMuted && priority != notifDomain.PriorityHigh {
			continue
		}

		// Build client-side channels map (only builtin WebSocket channels)
		clientChannels := make(map[string]bool, len(notifDomain.BuiltinClientChannels))
		for ch := range notifDomain.BuiltinClientChannels {
			if priority == notifDomain.PriorityHigh && pref.IsMuted {
				// High-priority notifications bypass mute: force default channels on
				clientChannels[ch] = true
			} else {
				clientChannels[ch] = !pref.IsMuted && pref.IsChannelEnabled(ch)
			}
		}

		payload := eventbus.NotificationPayload{
			Source:   req.Source,
			Title:    req.Title,
			Body:     req.Body,
			Link:     req.Link,
			Priority: priority,
			Channels: clientChannels,
		}

		data, err := json.Marshal(payload)
		if err != nil {
			slog.Error("failed to marshal notification payload", "error", err)
			continue
		}

		uid := userID
		_ = d.eventBus.Publish(ctx, &eventbus.Event{
			Type:           eventbus.EventNotification,
			Category:       eventbus.CategoryNotification,
			OrganizationID: req.OrganizationID,
			TargetUserID:   &uid,
			EntityType:     "notification",
			EntityID:       req.SourceEntityID,
			Data:           data,
		})

		// Fire-and-forget server-side delivery handlers
		for ch, handler := range d.deliveryHandlers {
			if pref.IsChannelEnabled(ch) {
				go func(h notifDomain.DeliveryHandler, uid int64) {
					if err := h.Deliver(ctx, uid, req); err != nil {
						slog.Error("delivery handler failed",
							"channel", h.Channel(),
							"user_id", uid,
							"error", err,
						)
					}
				}(handler, userID)
			}
		}
	}

	return nil
}

// resolve parses "prefix:param" and calls the appropriate resolver
func (d *Dispatcher) resolve(ctx context.Context, resolverStr string) ([]int64, error) {
	parts := strings.SplitN(resolverStr, ":", 2)
	if len(parts) != 2 {
		slog.Warn("invalid resolver string", "resolver", resolverStr)
		return nil, nil
	}

	prefix, param := parts[0], parts[1]
	resolver, ok := d.resolvers[prefix]
	if !ok {
		slog.Warn("no resolver registered for prefix", "prefix", prefix)
		return nil, nil
	}

	return resolver.Resolve(ctx, param)
}
