package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Publish publishes an event locally and to Redis for multi-instance sync
func (eb *EventBus) Publish(ctx context.Context, event *Event) error {
	// Set timestamp if not set
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().UnixMilli()
	}

	// Set category from registry if not set
	if event.Category == "" {
		event.Category = eb.registry.GetCategory(event.Type)
	}

	// Set source instance ID to prevent duplicate dispatch from Redis
	event.SourceInstanceID = eb.instanceID

	eb.logger.Debug("publishing event",
		"type", event.Type,
		"category", event.Category,
		"org_id", event.OrganizationID,
		"entity_type", event.EntityType,
		"entity_id", event.EntityID,
	)

	// Dispatch locally
	eb.dispatchLocal(event)

	// Publish to Redis for multi-instance sync
	if eb.redisClient != nil {
		if err := eb.publishToRedis(ctx, event); err != nil {
			eb.logger.Error("failed to publish event to Redis",
				"error", err,
				"type", event.Type,
				"org_id", event.OrganizationID,
			)
			// Don't return error - local dispatch already succeeded
		}
	}

	return nil
}

// dispatchLocal dispatches an event to local handlers
func (eb *EventBus) dispatchLocal(event *Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	// Call type-specific handlers
	if handlers, ok := eb.handlers[event.Type]; ok {
		for _, handler := range handlers {
			go eb.safeCallHandler(handler, event)
		}
	}

	// Call category handlers
	if handlers, ok := eb.categoryHandlers[event.Category]; ok {
		for _, handler := range handlers {
			go eb.safeCallHandler(handler, event)
		}
	}
}

// safeCallHandler calls a handler with panic recovery
func (eb *EventBus) safeCallHandler(handler EventHandler, event *Event) {
	defer func() {
		if r := recover(); r != nil {
			eb.logger.Error("event handler panic recovered",
				"error", r,
				"event_type", event.Type,
				"event_category", event.Category,
				"entity_type", event.EntityType,
				"entity_id", event.EntityID,
			)
		}
	}()
	handler(event)
}

// publishToRedis publishes an event to Redis pub/sub
func (eb *EventBus) publishToRedis(ctx context.Context, event *Event) error {
	channel := eb.redisChannel(event.OrganizationID)

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return eb.redisClient.Publish(ctx, channel, data).Err()
}

// redisChannel returns the Redis pub/sub channel name for an organization
func (eb *EventBus) redisChannel(orgID int64) string {
	return fmt.Sprintf("events:org:%d", orgID)
}
