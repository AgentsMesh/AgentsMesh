package instance

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// LocalOrgProvider returns the set of organization IDs that this server instance
// currently serves. Used by background tasks (LoopScheduler, timeout detection)
// to scope processing to only the orgs whose Runners are connected to this instance.
//
// Architecture: Traefik routes API requests by org slug, so each backend instance
// handles a subset of orgs. Background tasks (cron, timeout) must respect the
// same partitioning. The org set is derived from connected Runners — if an org
// has no Runner on this instance, there's no point scheduling Pods for it.
type LocalOrgProvider interface {
	// GetLocalOrgIDs returns org IDs that have at least one connected Runner
	// on this server instance. Returns nil if all orgs should be processed
	// (single-instance mode or no Runners connected yet).
	GetLocalOrgIDs() []int64
}

// ConnectedRunnerIDsProvider abstracts RunnerConnectionManager to avoid
// importing the runner package (prevents circular dependency).
type ConnectedRunnerIDsProvider interface {
	GetConnectedRunnerIDs() []int64
}

const (
	// redisKeyPrefix is used for storing local org IDs in Redis
	redisKeyPrefix = "instance:local_orgs:"

	// refreshInterval is the periodic refresh interval for the org cache
	refreshInterval = 30 * time.Second

	// redisTTL is the TTL for the Redis key (slightly longer than refresh interval)
	redisTTL = 60 * time.Second
)

// OrgAwarenessService maintains an in-memory cache of organization IDs served
// by this server instance, derived from connected Runners. Optionally syncs
// to Redis for cross-component visibility.
//
// Usage:
//   - Call Start() to begin periodic refresh
//   - Call Refresh() after runner connect/disconnect events
//   - Call GetLocalOrgIDs() from any goroutine (thread-safe)
//   - Call Stop() on shutdown
type OrgAwarenessService struct {
	mu     sync.RWMutex
	orgIDs []int64

	db              *gorm.DB
	runnerConnector ConnectedRunnerIDsProvider
	redisClient     *redis.Client
	logger          *slog.Logger
	instanceID      string // unique ID for this instance (used as Redis key suffix)
	stopCh          chan struct{}
	wg              sync.WaitGroup // tracks the background refresh goroutine
}

// NewOrgAwarenessService creates a new OrgAwarenessService.
//
// Parameters:
//   - db: database connection for querying runners.organization_id
//   - runnerConnector: provides connected runner IDs (from RunnerConnectionManager)
//   - redisClient: optional, for cross-component visibility (nil to disable)
//   - instanceID: unique identifier for this server instance (e.g., hostname:port)
//   - logger: structured logger
func NewOrgAwarenessService(
	db *gorm.DB,
	runnerConnector ConnectedRunnerIDsProvider,
	redisClient *redis.Client,
	instanceID string,
	logger *slog.Logger,
) *OrgAwarenessService {
	return &OrgAwarenessService{
		db:              db,
		runnerConnector: runnerConnector,
		redisClient:     redisClient,
		instanceID:      instanceID,
		logger:          logger.With("component", "org_awareness"),
		stopCh:          make(chan struct{}),
	}
}

// Start performs an initial refresh and begins periodic background refresh.
func (s *OrgAwarenessService) Start() {
	s.Refresh()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-s.stopCh:
				return
			case <-ticker.C:
				s.Refresh()
			}
		}
	}()

	s.logger.Info("org awareness service started", "refresh_interval", refreshInterval)
}

// Stop gracefully stops the periodic refresh.
func (s *OrgAwarenessService) Stop() {
	close(s.stopCh)
	s.wg.Wait() // wait for background goroutine to exit

	// Clean up Redis key on shutdown
	if s.redisClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		s.redisClient.Del(ctx, s.redisKey())
	}

	s.logger.Info("org awareness service stopped")
}

// GetLocalOrgIDs returns the cached list of local org IDs.
// Returns nil if no Runners are connected (process all orgs / single-instance mode).
// Thread-safe for concurrent access.
func (s *OrgAwarenessService) GetLocalOrgIDs() []int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.orgIDs) == 0 {
		return nil
	}

	// Return a copy to prevent callers from mutating the internal slice
	result := make([]int64, len(s.orgIDs))
	copy(result, s.orgIDs)
	return result
}

// Refresh queries connected runners and updates the cached org ID set.
// Call this after runner connect/disconnect events for immediate consistency.
// Thread-safe for concurrent calls.
func (s *OrgAwarenessService) Refresh() {
	runnerIDs := s.runnerConnector.GetConnectedRunnerIDs()

	var orgIDs []int64
	if len(runnerIDs) > 0 {
		s.db.Table("runners").
			Where("id IN ?", runnerIDs).
			Distinct("organization_id").
			Pluck("organization_id", &orgIDs)
	}

	s.mu.Lock()
	if len(orgIDs) == 0 {
		s.orgIDs = nil
	} else {
		s.orgIDs = orgIDs
	}
	s.mu.Unlock()

	// Sync to Redis for cross-component visibility
	if s.redisClient != nil {
		s.syncToRedis(orgIDs)
	}

	if len(orgIDs) > 0 {
		s.logger.Debug("org awareness refreshed",
			"org_count", len(orgIDs),
			"runner_count", len(runnerIDs),
			"org_ids", orgIDs,
		)
	}
}

// syncToRedis stores the current org ID set in Redis with a TTL.
func (s *OrgAwarenessService) syncToRedis(orgIDs []int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	key := s.redisKey()

	if len(orgIDs) == 0 {
		s.redisClient.Del(ctx, key)
		return
	}

	data, err := json.Marshal(orgIDs)
	if err != nil {
		s.logger.Error("failed to marshal org IDs for redis", "error", err)
		return
	}

	if err := s.redisClient.Set(ctx, key, data, redisTTL).Err(); err != nil {
		s.logger.Error("failed to sync org IDs to redis", "error", err)
	}
}

// redisKey returns the Redis key for this instance's org IDs.
func (s *OrgAwarenessService) redisKey() string {
	return redisKeyPrefix + s.instanceID
}
