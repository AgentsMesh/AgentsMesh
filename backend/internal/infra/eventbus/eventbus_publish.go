package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func (eb *EventBus) Publish(ctx context.Context, event *Event) error {
	ctx, span := otel.Tracer("agentsmesh-backend").Start(ctx, "eventbus.publish",
		trace.WithAttributes(attribute.String("event.type", string(event.Type))),
	)
	defer span.End()

	if event.Timestamp == 0 {
		event.Timestamp = time.Now().UnixMilli()
	}

	if event.Category == "" {
		event.Category = eb.registry.GetCategory(event.Type)
	}

	event.SourceInstanceID = eb.instanceID

	eb.logger.Debug("publishing event",
		"type", event.Type,
		"category", event.Category,
		"org_id", event.OrganizationID,
		"entity_type", event.EntityType,
		"entity_id", event.EntityID,
	)

	eb.dispatchLocal(event)

	if eb.redisClient != nil {
		if err := eb.publishToRedis(ctx, event); err != nil {
			eb.logger.Error("failed to publish event to Redis",
				"error", err,
				"type", event.Type,
				"org_id", event.OrganizationID,
			)
		}
	}

	return nil
}

func (eb *EventBus) dispatchLocal(event *Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if handlers, ok := eb.handlers[event.Type]; ok {
		for _, handler := range handlers {
			go eb.safeCallHandler(handler, event)
		}
	}

	if handlers, ok := eb.categoryHandlers[event.Category]; ok {
		for _, handler := range handlers {
			go eb.safeCallHandler(handler, event)
		}
	}
}

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

func (eb *EventBus) publishToRedis(ctx context.Context, event *Event) error {
	channel := eb.redisChannel(event.OrganizationID)

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return eb.redisClient.Publish(ctx, channel, data).Err()
}

func (eb *EventBus) redisChannel(orgID int64) string {
	return fmt.Sprintf("events:org:%d", orgID)
}
