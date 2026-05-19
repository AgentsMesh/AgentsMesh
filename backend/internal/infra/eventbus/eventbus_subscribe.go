package eventbus

import (
	"context"
	"encoding/json"
)

func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

func (eb *EventBus) SubscribeCategory(category EventCategory, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.categoryHandlers[category] = append(eb.categoryHandlers[category], handler)
}

func (eb *EventBus) SubscribeOrg(orgID int64) {
	eb.orgsMu.Lock()
	defer eb.orgsMu.Unlock()

	if eb.subscribedOrgs[orgID] {
		return
	}

	eb.subscribedOrgs[orgID] = true

	orgCtx, orgCancel := context.WithCancel(eb.ctx)
	eb.orgCancels[orgID] = orgCancel

	go eb.subscribeToOrgChannel(orgCtx, orgID)
}

func (eb *EventBus) UnsubscribeOrg(orgID int64) {
	eb.orgsMu.Lock()
	defer eb.orgsMu.Unlock()
	delete(eb.subscribedOrgs, orgID)

	if cancel, ok := eb.orgCancels[orgID]; ok {
		cancel()
		delete(eb.orgCancels, orgID)
	}
}

func (eb *EventBus) subscribeToOrgChannel(ctx context.Context, orgID int64) {
	if eb.redisClient == nil {
		return
	}

	channel := eb.redisChannel(orgID)
	pubsub := eb.redisClient.Subscribe(ctx, channel)
	defer pubsub.Close()

	eb.logger.Debug("subscribed to Redis channel", "channel", channel, "org_id", orgID)

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}

			var event Event
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				eb.logger.Error("failed to unmarshal event from Redis",
					"error", err,
					"channel", channel,
				)
				continue
			}

			if event.SourceInstanceID == eb.instanceID {
				continue
			}

			eb.dispatchLocal(&event)
		}
	}
}

func (eb *EventBus) StartRedisSubscriber(ctx context.Context) {
	if eb.redisClient == nil {
		eb.logger.Warn("Redis client not available, skipping Redis subscriber")
		return
	}

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

				if event.SourceInstanceID == eb.instanceID {
					continue
				}

				eb.dispatchLocal(&event)
			}
		}
	}()
}
