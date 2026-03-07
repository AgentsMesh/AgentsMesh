package relay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/infra/cache"
	"github.com/redis/go-redis/v9"
)

// Redis key prefixes and TTLs for relay persistence.
const (
	relayKeyPrefix       = "relay:info:"
	relayHeartbeatPrefix = "relay:heartbeat:"
	relayListKey         = "relay:list"
	relayHeartbeatTTL    = 60 * time.Second // Relay heartbeat expires after 60s
)

// RedisStore implements Store interface using Redis
type RedisStore struct {
	cache  *cache.Cache
	prefix string // Optional key prefix for multi-tenant scenarios
}

// NewRedisStore creates a new Redis-backed store
func NewRedisStore(c *cache.Cache, prefix string) *RedisStore {
	return &RedisStore{
		cache:  c,
		prefix: prefix,
	}
}

// key returns a prefixed key
func (s *RedisStore) key(parts ...string) string {
	result := s.prefix
	for _, p := range parts {
		result += p
	}
	return result
}

// SaveRelay saves relay info to Redis using pipeline (1 RTT for 3 commands).
func (s *RedisStore) SaveRelay(ctx context.Context, relay *RelayInfo) error {
	data, err := json.Marshal(relay)
	if err != nil {
		return fmt.Errorf("failed to marshal relay: %w", err)
	}

	pipe := s.cache.Client().Pipeline()
	pipe.Set(ctx, s.key(relayKeyPrefix, relay.ID), data, 0)
	pipe.SAdd(ctx, s.key(relayListKey), relay.ID)
	// Use relay.LastHeartbeat (set by Manager.Register) for consistency with in-memory state
	pipe.Set(ctx, s.key(relayHeartbeatPrefix, relay.ID), relay.LastHeartbeat.Unix(), relayHeartbeatTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to save relay: %w", err)
	}
	return nil
}

// GetRelay retrieves relay info from Redis
func (s *RedisStore) GetRelay(ctx context.Context, relayID string) (*RelayInfo, error) {
	key := s.key(relayKeyPrefix, relayID)
	data, err := s.cache.Client().Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to get relay: %w", err)
	}

	var relay RelayInfo
	if err := json.Unmarshal(data, &relay); err != nil {
		return nil, fmt.Errorf("failed to unmarshal relay: %w", err)
	}

	// Check heartbeat to determine health (treat errors as unhealthy)
	heartbeatKey := s.key(relayHeartbeatPrefix, relayID)
	exists, err := s.cache.Exists(ctx, heartbeatKey)
	if err != nil {
		// Log but don't fail — stale health is better than no data
		relay.Healthy = false
	} else {
		relay.Healthy = exists
	}

	return &relay, nil
}

// GetAllRelays retrieves all relay infos from Redis using pipeline to avoid N+1 queries.
func (s *RedisStore) GetAllRelays(ctx context.Context) ([]*RelayInfo, error) {
	// Get all relay IDs from the set
	relayIDs, err := s.cache.Client().SMembers(ctx, s.key(relayListKey)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get relay list: %w", err)
	}

	if len(relayIDs) == 0 {
		return nil, nil
	}

	// Pipeline: batch GET relay data + EXISTS heartbeat in one round-trip
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

	// Execute pipeline: 1 round-trip instead of 2N+1.
	// Pipeline Exec returns redis.Nil when some keys are missing — this is expected
	// for partially-deleted relays; only treat non-Nil errors as failures.
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("failed to execute pipeline: %w", err)
	}

	relays := make([]*RelayInfo, 0, len(relayIDs))
	var orphanIDs []string
	for _, cmd := range cmds {
		data, err := cmd.dataCmd.Bytes()
		if err != nil {
			// relay:info key is gone but ID still in relay:list — orphan
			orphanIDs = append(orphanIDs, cmd.id)
			continue
		}
		var relay RelayInfo
		if err := json.Unmarshal(data, &relay); err != nil {
			orphanIDs = append(orphanIDs, cmd.id)
			continue
		}
		// Exists returns count; >=1 means heartbeat key is present
		relay.Healthy = cmd.aliveCmd.Val() >= 1
		// relay:info JSON stores the original registration time, but doHealthCheck
		// uses LastHeartbeat to determine staleness. Adjust it based on heartbeat key:
		if relay.Healthy {
			// Heartbeat key alive → treat as just-heartbeated to prevent stale eviction
			relay.LastHeartbeat = time.Now()
		} else {
			// Heartbeat key expired → relay has been offline for at least relayHeartbeatTTL.
			// Use a conservatively large offset so doHealthCheck can auto-remove truly stale
			// relays. Without this, every backend restart would reset offline relays to
			// "just 60s ago", preventing stale eviction indefinitely.
			relay.LastHeartbeat = time.Now().Add(-relayHeartbeatTTL * time.Duration(staleRelayMultiplier+1))
		}
		relays = append(relays, &relay)
	}

	// Clean up orphan IDs from relay:list (best-effort).
	// Re-verify each candidate: between our pipeline read and this cleanup,
	// another instance might have re-registered the relay via SaveRelay.
	if len(orphanIDs) > 0 {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), storeOpTimeout)
		defer cleanCancel()

		// Re-verify: only remove IDs whose relay:info key is still missing
		verifyPipe := s.cache.Client().Pipeline()
		verifyCmds := make([]*redis.StringCmd, len(orphanIDs))
		for i, id := range orphanIDs {
			verifyCmds[i] = verifyPipe.Get(cleanCtx, s.key(relayKeyPrefix, id))
		}
		_, _ = verifyPipe.Exec(cleanCtx) // errors expected for missing keys

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

// DeleteRelay removes relay from Redis using pipeline (1 RTT for 3 commands).
func (s *RedisStore) DeleteRelay(ctx context.Context, relayID string) error {
	pipe := s.cache.Client().Pipeline()
	pipe.Del(ctx, s.key(relayKeyPrefix, relayID))
	pipe.SRem(ctx, s.key(relayListKey), relayID)
	pipe.Del(ctx, s.key(relayHeartbeatPrefix, relayID))

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete relay: %w", err)
	}
	return nil
}

// UpdateRelayHeartbeat refreshes the heartbeat TTL key for a relay.
// Only touches the heartbeat key — does NOT read-modify-write relay:info,
// because health status is derived from heartbeat key existence in GetRelay/GetAllRelays.
func (s *RedisStore) UpdateRelayHeartbeat(ctx context.Context, relayID string, heartbeat time.Time) error {
	key := s.key(relayHeartbeatPrefix, relayID)
	if err := s.cache.Client().Set(ctx, key, heartbeat.Unix(), relayHeartbeatTTL).Err(); err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}
	return nil
}
