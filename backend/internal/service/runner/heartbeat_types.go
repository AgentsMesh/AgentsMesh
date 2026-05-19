package runner

import (
	"log/slog"
	"sync"
	"time"

	runnerDomain "github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/redis/go-redis/v9"
)

const (
	DefaultFlushInterval = 5 * time.Second

	DefaultHeartbeatTTL = 90 * time.Second

	DefaultBatchSize = 100

	HeartbeatOnlineThreshold = 90 * time.Second
)

// HeartbeatBatcher flushes heartbeats every 5s — cuts DB write rate from
// 100k runners × (1/30s) ≈ 3.3k/s to ~670/s. NOT thread-safe to swap interval at runtime.
type HeartbeatBatcher struct {
	redis      *redis.Client
	runnerRepo runnerDomain.RunnerRepository
	logger     *slog.Logger

	buffer   map[int64]*HeartbeatItem
	mu       sync.Mutex
	interval time.Duration

	stopCh  chan struct{}
	doneCh  chan struct{}
	running bool
}

type HeartbeatItem struct {
	RunnerID    int64
	CurrentPods int
	Status      string
	Version     string
	Timestamp   time.Time
}

type RedisRunnerStatus struct {
	LastHeartbeat int64  `json:"last_heartbeat"`
	CurrentPods   int    `json:"current_pods"`
	Status        string `json:"status"`
	Version       string `json:"version,omitempty"`
}

func NewHeartbeatBatcher(redisClient *redis.Client, runnerRepo runnerDomain.RunnerRepository, logger *slog.Logger) *HeartbeatBatcher {
	return &HeartbeatBatcher{
		redis:      redisClient,
		runnerRepo: runnerRepo,
		logger:     logger,
		buffer:     make(map[int64]*HeartbeatItem),
		interval:   DefaultFlushInterval,
	}
}

func (b *HeartbeatBatcher) SetInterval(interval time.Duration) {
	b.interval = interval
}

func (b *HeartbeatBatcher) BufferSize() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.buffer)
}
