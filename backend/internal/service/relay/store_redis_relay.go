package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/infra/cache"
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

// SaveRelay saves relay info to Redis
func (s *RedisStore) SaveRelay(ctx context.Context, relay *RelayInfo) error {
	data, err := json.Marshal(relay)
	if err != nil {
		return fmt.Errorf("failed to marshal relay: %w", err)
	}

	// Save relay data (no expiration, managed by heartbeat)
	key := s.key(relayKeyPrefix, relay.ID)
	if err := s.cache.Client().Set(ctx, key, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to save relay: %w", err)
	}

	// Add to relay list set
	if err := s.cache.Client().SAdd(ctx, s.key(relayListKey), relay.ID).Err(); err != nil {
		return fmt.Errorf("failed to add relay to list: %w", err)
	}

	// Update heartbeat with TTL
	heartbeatKey := s.key(relayHeartbeatPrefix, relay.ID)
	if err := s.cache.Client().Set(ctx, heartbeatKey, time.Now().Unix(), relayHeartbeatTTL).Err(); err != nil {
		return fmt.Errorf("failed to set heartbeat: %w", err)
	}

	return nil
}

// GetRelay retrieves relay info from Redis
func (s *RedisStore) GetRelay(ctx context.Context, relayID string) (*RelayInfo, error) {
	key := s.key(relayKeyPrefix, relayID)
	data, err := s.cache.Client().Get(ctx, key).Bytes()
	if err != nil {
		if err.Error() == "redis: nil" {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to get relay: %w", err)
	}

	var relay RelayInfo
	if err := json.Unmarshal(data, &relay); err != nil {
		return nil, fmt.Errorf("failed to unmarshal relay: %w", err)
	}

	// Check heartbeat to determine health
	heartbeatKey := s.key(relayHeartbeatPrefix, relayID)
	exists, _ := s.cache.Exists(ctx, heartbeatKey)
	relay.Healthy = exists

	return &relay, nil
}

// GetAllRelays retrieves all relay infos from Redis
func (s *RedisStore) GetAllRelays(ctx context.Context) ([]*RelayInfo, error) {
	// Get all relay IDs from the set
	relayIDs, err := s.cache.Client().SMembers(ctx, s.key(relayListKey)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get relay list: %w", err)
	}

	relays := make([]*RelayInfo, 0, len(relayIDs))
	for _, id := range relayIDs {
		relay, err := s.GetRelay(ctx, id)
		if err != nil {
			continue // Skip errors for individual relays
		}
		if relay != nil {
			relays = append(relays, relay)
		}
	}

	return relays, nil
}

// DeleteRelay removes relay from Redis
func (s *RedisStore) DeleteRelay(ctx context.Context, relayID string) error {
	// Delete relay data
	key := s.key(relayKeyPrefix, relayID)
	if err := s.cache.Client().Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete relay: %w", err)
	}

	// Remove from relay list
	if err := s.cache.Client().SRem(ctx, s.key(relayListKey), relayID).Err(); err != nil {
		return fmt.Errorf("failed to remove relay from list: %w", err)
	}

	// Delete heartbeat
	heartbeatKey := s.key(relayHeartbeatPrefix, relayID)
	s.cache.Client().Del(ctx, heartbeatKey)

	return nil
}

// UpdateRelayHeartbeat updates the heartbeat timestamp for a relay
func (s *RedisStore) UpdateRelayHeartbeat(ctx context.Context, relayID string, heartbeat time.Time) error {
	// Update the heartbeat key with TTL
	heartbeatKey := s.key(relayHeartbeatPrefix, relayID)
	if err := s.cache.Client().Set(ctx, heartbeatKey, heartbeat.Unix(), relayHeartbeatTTL).Err(); err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}

	// Also update the LastHeartbeat field in relay data
	relay, err := s.GetRelay(ctx, relayID)
	if err != nil || relay == nil {
		return nil // Relay not found, skip
	}

	relay.LastHeartbeat = heartbeat
	relay.Healthy = true

	data, _ := json.Marshal(relay)
	key := s.key(relayKeyPrefix, relayID)
	return s.cache.Client().Set(ctx, key, data, 0).Err()
}
