package relay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/infra/cache"
	"github.com/redis/go-redis/v9"
)

const (
	relayKeyPrefix       = "relay:info:"
	relayHeartbeatPrefix = "relay:heartbeat:"
	relayListKey         = "relay:list"
	relayHeartbeatTTL    = 60 * time.Second
)

type RedisStore struct {
	cache  *cache.Cache
	prefix string
}

func NewRedisStore(c *cache.Cache, prefix string) *RedisStore {
	return &RedisStore{
		cache:  c,
		prefix: prefix,
	}
}

func (s *RedisStore) key(parts ...string) string {
	result := s.prefix
	for _, p := range parts {
		result += p
	}
	return result
}

func (s *RedisStore) SaveRelay(ctx context.Context, relay *RelayInfo) error {
	data, err := json.Marshal(relay)
	if err != nil {
		return fmt.Errorf("failed to marshal relay: %w", err)
	}

	pipe := s.cache.Client().Pipeline()
	pipe.Set(ctx, s.key(relayKeyPrefix, relay.ID), data, 0)
	pipe.SAdd(ctx, s.key(relayListKey), relay.ID)
	pipe.Set(ctx, s.key(relayHeartbeatPrefix, relay.ID), relay.LastHeartbeat.Unix(), relayHeartbeatTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		slog.ErrorContext(ctx, "failed to save relay", "relay_id", relay.ID, "error", err)
		return fmt.Errorf("failed to save relay: %w", err)
	}
	return nil
}

func (s *RedisStore) GetRelay(ctx context.Context, relayID string) (*RelayInfo, error) {
	key := s.key(relayKeyPrefix, relayID)
	data, err := s.cache.Client().Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get relay: %w", err)
	}

	var relay RelayInfo
	if err := json.Unmarshal(data, &relay); err != nil {
		return nil, fmt.Errorf("failed to unmarshal relay: %w", err)
	}

	heartbeatKey := s.key(relayHeartbeatPrefix, relayID)
	exists, err := s.cache.Exists(ctx, heartbeatKey)
	if err != nil {
		relay.Healthy = false
	} else {
		relay.Healthy = exists
	}

	return &relay, nil
}

func (s *RedisStore) GetAllRelays(ctx context.Context) ([]*RelayInfo, error) {
	relayIDs, err := s.cache.Client().SMembers(ctx, s.key(relayListKey)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get relay list: %w", err)
	}

	if len(relayIDs) == 0 {
		return nil, nil
	}

	pipe := s.cache.Client().Pipeline()

	type relayCmd struct {
		id       string
		dataCmd  *redis.StringCmd
		aliveCmd *redis.IntCmd
	}
	cmds := make([]relayCmd, len(relayIDs))
	for i, id := range relayIDs {
		cmds[i] = relayCmd{
			id:       id,
			dataCmd:  pipe.Get(ctx, s.key(relayKeyPrefix, id)),
			aliveCmd: pipe.Exists(ctx, s.key(relayHeartbeatPrefix, id)),
		}
	}

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("failed to execute pipeline: %w", err)
	}

	relays := make([]*RelayInfo, 0, len(relayIDs))
	var orphanIDs []string
	for _, cmd := range cmds {
		data, err := cmd.dataCmd.Bytes()
		if err != nil {
			orphanIDs = append(orphanIDs, cmd.id)
			continue
		}
		var relay RelayInfo
		if err := json.Unmarshal(data, &relay); err != nil {
			orphanIDs = append(orphanIDs, cmd.id)
			continue
		}
		relay.Healthy = cmd.aliveCmd.Val() >= 1
		if relay.Healthy {
			relay.LastHeartbeat = time.Now()
		} else {
			// Pre-date LastHeartbeat past staleRelayMultiplier × TTL so doHealthCheck can
			// evict; otherwise every backend restart would reset offline relays to "60s ago" forever.
			relay.LastHeartbeat = time.Now().Add(-relayHeartbeatTTL * time.Duration(staleRelayMultiplier+1))
		}
		relays = append(relays, &relay)
	}

	if len(orphanIDs) > 0 {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), storeOpTimeout)
		defer cleanCancel()

		verifyPipe := s.cache.Client().Pipeline()
		verifyCmds := make([]*redis.StringCmd, len(orphanIDs))
		for i, id := range orphanIDs {
			verifyCmds[i] = verifyPipe.Get(cleanCtx, s.key(relayKeyPrefix, id))
		}
		_, _ = verifyPipe.Exec(cleanCtx)

		var confirmed []interface{}
		for i, id := range orphanIDs {
			if _, err := verifyCmds[i].Result(); errors.Is(err, redis.Nil) {
				confirmed = append(confirmed, id)
			}
		}
		if len(confirmed) > 0 {
			_ = s.cache.Client().SRem(cleanCtx, s.key(relayListKey), confirmed...).Err()
		}
	}

	return relays, nil
}

func (s *RedisStore) DeleteRelay(ctx context.Context, relayID string) error {
	pipe := s.cache.Client().Pipeline()
	pipe.Del(ctx, s.key(relayKeyPrefix, relayID))
	pipe.SRem(ctx, s.key(relayListKey), relayID)
	pipe.Del(ctx, s.key(relayHeartbeatPrefix, relayID))

	if _, err := pipe.Exec(ctx); err != nil {
		slog.ErrorContext(ctx, "failed to delete relay", "relay_id", relayID, "error", err)
		return fmt.Errorf("failed to delete relay: %w", err)
	}
	return nil
}

// UpdateRelayHeartbeat only touches the heartbeat key — health is derived from
// heartbeat existence in GetRelay/GetAllRelays, never from relay:info.
func (s *RedisStore) UpdateRelayHeartbeat(ctx context.Context, relayID string, heartbeat time.Time) error {
	key := s.key(relayHeartbeatPrefix, relayID)
	if err := s.cache.Client().Set(ctx, key, heartbeat.Unix(), relayHeartbeatTTL).Err(); err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}
	return nil
}
