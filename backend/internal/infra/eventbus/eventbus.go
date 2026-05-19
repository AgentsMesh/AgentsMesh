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

type EventBus struct {
	registry    *EventRegistry
	redisClient *redis.Client
	logger      *slog.Logger

	instanceID string

	handlers map[EventType][]EventHandler
	categoryHandlers map[EventCategory][]EventHandler

	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc

	subscribedOrgs map[int64]bool
	orgCancels     map[int64]context.CancelFunc // per-org cancel functions to stop goroutines
	orgsMu         sync.RWMutex
}

func NewEventBus(redisClient *redis.Client, logger *slog.Logger) *EventBus {
	ctx, cancel := context.WithCancel(context.Background())

	if logger == nil {
		logger = slog.Default()
	}

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

func (eb *EventBus) Close() {
	eb.cancel()
}

func (eb *EventBus) Registry() *EventRegistry {
	return eb.registry
}
