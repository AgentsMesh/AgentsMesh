package runner

import (
	"log/slog"
	"sync"
	"time"

	runnerDomain "github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/redis/go-redis/v9"
)

const (
	// DefaultFlushInterval is the default interval for flushing heartbeats to database
	DefaultFlushInterval = 5 * time.Second

	// DefaultHeartbeatTTL is the Redis TTL for heartbeat status (3x heartbeat interval for safety)
	DefaultHeartbeatTTL = 90 * time.Second

	// DefaultBatchSize is the number of heartbeats to process in a single batch
	DefaultBatchSize = 100

	// HeartbeatOnlineThreshold is the threshold to consider a runner online (same as TTL)
	HeartbeatOnlineThreshold = 90 * time.Second
)

// HeartbeatBatcher aggregates heartbeat updates and flushes them to the database in batches
// This reduces database write pressure from 3,333/sec (100K runners / 30s) to ~667/sec (5s batch)
type HeartbeatBatcher struct {
	redis      *redis.Client
	runnerRepo runnerDomain.RunnerRepository
	logger     *slog.Logger

	buffer   map[int64]*HeartbeatItem
	mu       sync.Mutex
	interval time.Duration

	// Lifecycle management - protected by mu
	stopCh  chan struct{}
	doneCh  chan struct{}
	running bool
}

// HeartbeatItem represents a pending heartbeat update
type HeartbeatItem struct {
	RunnerID    int64
	CurrentPods int
	Status      string
	Version     string
	Timestamp   time.Time
}

// RedisRunnerStatus represents runner status stored in Redis for real-time queries
type RedisRunnerStatus struct {
	LastHeartbeat int64  `json:"last_heartbeat"`
	CurrentPods   int    `json:"current_pods"`
	Status        string `json:"status"`
	Version       string `json:"version,omitempty"`
}

// NewHeartbeatBatcher creates a new heartbeat batcher
func NewHeartbeatBatcher(redisClient *redis.Client, runnerRepo runnerDomain.RunnerRepository, logger *slog.Logger) *HeartbeatBatcher {
	return &HeartbeatBatcher{
		redis:      redisClient,
		runnerRepo: runnerRepo,
		logger:     logger,
		buffer:     make(map[int64]*HeartbeatItem),
		interval:   DefaultFlushInterval,
		// stopCh and doneCh are created in Start() to allow restart
	}
}

// SetInterval sets the flush interval (for testing)
func (b *HeartbeatBatcher) SetInterval(interval time.Duration) {
	b.interval = interval
}

// BufferSize returns the current buffer size (for monitoring)
func (b *HeartbeatBatcher) BufferSize() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.buffer)
}
