package instance

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type LocalOrgProvider interface {
	GetLocalOrgIDs() []int64
}

type ConnectedRunnerIDsProvider interface {
	GetConnectedRunnerIDs() []int64
}

type RunnerOrgQuerier interface {
	GetOrgIDsByRunnerIDs(ctx context.Context, runnerIDs []int64) ([]int64, error)
}

const (
	redisKeyPrefix = "instance:local_orgs:"

	refreshInterval = 30 * time.Second

	redisTTL = 60 * time.Second
)

// OrgAwarenessService GetLocalOrgIDs is read by tasks/loop runners to filter
// work to this instance's connected orgs; refresh runs every 30s + on
// runner connect/disconnect events; Redis mirror has 60s TTL.
type OrgAwarenessService struct {
	mu     sync.RWMutex
	orgIDs []int64

	orgQuerier      RunnerOrgQuerier
	runnerConnector ConnectedRunnerIDsProvider
	redisClient     *redis.Client
	logger          *slog.Logger
	instanceID      string
	stopCh          chan struct{}
	wg              sync.WaitGroup
}

func NewOrgAwarenessService(
	orgQuerier RunnerOrgQuerier,
	runnerConnector ConnectedRunnerIDsProvider,
	redisClient *redis.Client,
	instanceID string,
	logger *slog.Logger,
) *OrgAwarenessService {
	return &OrgAwarenessService{
		orgQuerier:      orgQuerier,
		runnerConnector: runnerConnector,
		redisClient:     redisClient,
		instanceID:      instanceID,
		logger:          logger.With("component", "org_awareness"),
		stopCh:          make(chan struct{}),
	}
}

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

func (s *OrgAwarenessService) Stop() {
	close(s.stopCh)
	s.wg.Wait()

	if s.redisClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		s.redisClient.Del(ctx, s.redisKey())
	}

	s.logger.Info("org awareness service stopped")
}

// GetLocalOrgIDs returns nil when no Runners connected (single-instance mode
// or empty); callers treat nil as "process all orgs".
func (s *OrgAwarenessService) GetLocalOrgIDs() []int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.orgIDs) == 0 {
		return nil
	}

	result := make([]int64, len(s.orgIDs))
	copy(result, s.orgIDs)
	return result
}

func (s *OrgAwarenessService) Refresh() {
	runnerIDs := s.runnerConnector.GetConnectedRunnerIDs()

	var orgIDs []int64
	if len(runnerIDs) > 0 {
		var err error
		orgIDs, err = s.orgQuerier.GetOrgIDsByRunnerIDs(context.Background(), runnerIDs)
		if err != nil {
			s.logger.Error("failed to query org IDs", "error", err)
		}
	}

	s.mu.Lock()
	if len(orgIDs) == 0 {
		s.orgIDs = nil
	} else {
		s.orgIDs = orgIDs
	}
	s.mu.Unlock()

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

func (s *OrgAwarenessService) redisKey() string {
	return redisKeyPrefix + s.instanceID
}
