package notification

import (
	"context"
	"log/slog"
	"strings"

	notifDomain "github.com/anthropics/agentsmesh/backend/internal/domain/notification"
)

// Dispatcher routes notifications to users based on preferences.
// Pushes directly to WebSocket via RealtimePusher, bypassing EventBus.
type Dispatcher struct {
	pusher           notifDomain.RealtimePusher
	prefStore        *PreferenceStore
	resolvers        map[string]RecipientResolver
	deliveryHandlers map[string]notifDomain.DeliveryHandler
}

func NewDispatcher(pusher notifDomain.RealtimePusher, prefStore *PreferenceStore) *Dispatcher {
	return &Dispatcher{
		pusher:           pusher,
		prefStore:        prefStore,
		resolvers:        make(map[string]RecipientResolver),
		deliveryHandlers: make(map[string]notifDomain.DeliveryHandler),
	}
}

func (d *Dispatcher) RegisterResolver(prefix string, resolver RecipientResolver) {
	d.resolvers[prefix] = resolver
}

func (d *Dispatcher) RegisterDeliveryHandler(handler notifDomain.DeliveryHandler) {
	d.deliveryHandlers[handler.Channel()] = handler
}

func (d *Dispatcher) Dispatch(ctx context.Context, req *notifDomain.NotificationRequest) error {
	recipientIDs := d.resolveRecipients(ctx, req)
	if len(recipientIDs) == 0 {
		return nil
	}

	priority := req.Priority
	if priority == "" {
		priority = notifDomain.PriorityNormal
	}

	for _, userID := range recipientIDs {
		pref := d.prefStore.GetPreference(ctx, userID, req.Source, req.SourceEntityID)
		if pref.IsMuted && priority != notifDomain.PriorityHigh {
			continue
		}

		channels := d.buildChannels(pref, priority)
		d.pushToUser(ctx, userID, req, priority, channels)
		d.fireDeliveryHandlers(ctx, userID, pref, req)
	}

	return nil
}

func (d *Dispatcher) resolve(ctx context.Context, resolverStr string) ([]int64, error) {
	parts := strings.SplitN(resolverStr, ":", 2)
	if len(parts) != 2 {
		slog.Warn("invalid resolver string", "resolver", resolverStr)
		return nil, nil
	}
	resolver, ok := d.resolvers[parts[0]]
	if !ok {
		slog.Warn("no resolver registered for prefix", "prefix", parts[0])
		return nil, nil
	}
	return resolver.Resolve(ctx, parts[1])
}
