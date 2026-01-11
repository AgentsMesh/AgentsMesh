package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// EventBus is the central event publishing and subscription system
type EventBus struct {
	registry    *EventRegistry
	redisClient *redis.Client
	logger      *slog.Logger

	// instanceID uniquely identifies this server instance
	// Used to prevent duplicate event dispatch from Redis
	instanceID string

	// Local handlers by event type
	handlers map[EventType][]EventHandler
	// Category handlers (handle all events in a category)
	categoryHandlers map[EventCategory][]EventHandler

	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc

	// subscribedOrgs tracks which organizations are subscribed for Redis pub/sub
	subscribedOrgs map[int64]bool
	orgsMu         sync.RWMutex
}

// NewEventBus creates a new EventBus instance
func NewEventBus(redisClient *redis.Client, logger *slog.Logger) *EventBus {
	ctx, cancel := context.WithCancel(context.Background())

	if logger == nil {
		logger = slog.Default()
	}

	// Generate unique instance ID for this server instance
	// Format: hostname-uuid (for easier debugging)
	hostname, _ := os.Hostname()
	instanceID := fmt.Sprintf("%s-%s", hostname, uuid.New().String()[:8])

	return &EventBus{
		registry:         DefaultRegistry,
		redisClient:      redisClient,
		logger:           logger.With("component", "eventbus", "instance_id", instanceID),
		instanceID:       instanceID,
		handlers:         make(map[EventType][]EventHandler),
		categoryHandlers: make(map[EventCategory][]EventHandler),
		subscribedOrgs:   make(map[int64]bool),
		ctx:              ctx,
		cancel:           cancel,
	}
}

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
		"target_user_id", event.TargetUserID,
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

// Close shuts down the event bus
func (eb *EventBus) Close() {
	eb.cancel()
}

// Registry returns the event registry
func (eb *EventBus) Registry() *EventRegistry {
	return eb.registry
}

// Helper functions for creating events

// NewEntityEvent creates a new entity event
func NewEntityEvent(eventType EventType, orgID int64, entityType, entityID string, data interface{}) (*Event, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event data: %w", err)
	}

	return &Event{
		Type:           eventType,
		Category:       CategoryEntity,
		OrganizationID: orgID,
		EntityType:     entityType,
		EntityID:       entityID,
		Data:           jsonData,
		Timestamp:      time.Now().UnixMilli(),
	}, nil
}

// NewNotificationEvent creates a new notification event targeted to specific users
func NewNotificationEvent(eventType EventType, orgID int64, targetUserID *int64, targetUserIDs []int64, entityType, entityID string, data interface{}) (*Event, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event data: %w", err)
	}

	return &Event{
		Type:           eventType,
		Category:       CategoryNotification,
		OrganizationID: orgID,
		TargetUserID:   targetUserID,
		TargetUserIDs:  targetUserIDs,
		EntityType:     entityType,
		EntityID:       entityID,
		Data:           jsonData,
		Timestamp:      time.Now().UnixMilli(),
	}, nil
}
