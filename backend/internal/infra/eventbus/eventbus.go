package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

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
	orgCancels     map[int64]context.CancelFunc // per-org cancel functions to stop goroutines
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
		orgCancels:       make(map[int64]context.CancelFunc),
		ctx:              ctx,
		cancel:           cancel,
	}
}

// Close shuts down the event bus
func (eb *EventBus) Close() {
	eb.cancel()
}

// Registry returns the event registry
func (eb *EventBus) Registry() *EventRegistry {
	return eb.registry
}
