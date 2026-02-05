package eventbus

import (
	"context"
	"encoding/json"
)

// Subscribe registers a handler for a specific event type
func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

// SubscribeCategory registers a handler for all events in a category
func (eb *EventBus) SubscribeCategory(category EventCategory, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.categoryHandlers[category] = append(eb.categoryHandlers[category], handler)
}

// SubscribeOrg subscribes to events for a specific organization via Redis
func (eb *EventBus) SubscribeOrg(orgID int64) {
	eb.orgsMu.Lock()
	defer eb.orgsMu.Unlock()

	if eb.subscribedOrgs[orgID] {
		return
	}

	eb.subscribedOrgs[orgID] = true

	// Start a goroutine to subscribe to this org's channel
	go eb.subscribeToOrgChannel(orgID)
}

// UnsubscribeOrg unsubscribes from events for a specific organization
func (eb *EventBus) UnsubscribeOrg(orgID int64) {
	eb.orgsMu.Lock()
	defer eb.orgsMu.Unlock()
	delete(eb.subscribedOrgs, orgID)
}

// subscribeToOrgChannel subscribes to Redis pub/sub for an organization
func (eb *EventBus) subscribeToOrgChannel(orgID int64) {
	if eb.redisClient == nil {
		return
	}

	channel := eb.redisChannel(orgID)
	pubsub := eb.redisClient.Subscribe(eb.ctx, channel)
	defer pubsub.Close()

	eb.logger.Debug("subscribed to Redis channel", "channel", channel, "org_id", orgID)

	ch := pubsub.Channel()
	for {
		select {
		case <-eb.ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}

			// Check if still subscribed
			eb.orgsMu.RLock()
			subscribed := eb.subscribedOrgs[orgID]
			eb.orgsMu.RUnlock()
			if !subscribed {
				return
			}

			// Parse and dispatch event
			var event Event
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				eb.logger.Error("failed to unmarshal event from Redis",
					"error", err,
					"channel", channel,
				)
				continue
			}

			// Skip events from this instance (already dispatched locally)
			if event.SourceInstanceID == eb.instanceID {
				continue
			}

			// Dispatch events from other instances
			eb.dispatchLocal(&event)
		}
	}
}

// StartRedisSubscriber starts listening to all organization channels
// This is used when the server starts to catch up on events
func (eb *EventBus) StartRedisSubscriber(ctx context.Context) {
	if eb.redisClient == nil {
		eb.logger.Warn("Redis client not available, skipping Redis subscriber")
		return
	}

	// Subscribe to pattern for all orgs
	pattern := "events:org:*"
	pubsub := eb.redisClient.PSubscribe(ctx, pattern)

	eb.logger.Info("started Redis pattern subscriber", "pattern", pattern)

	go func() {
		defer pubsub.Close()

		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case <-eb.ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}

				var event Event
				if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
					eb.logger.Error("failed to unmarshal event from Redis",
						"error", err,
						"channel", msg.Channel,
					)
					continue
				}

				// Skip events from this instance (already dispatched locally)
				if event.SourceInstanceID == eb.instanceID {
					continue
				}

				// Dispatch events from other instances
				eb.dispatchLocal(&event)
			}
		}
	}()
}
