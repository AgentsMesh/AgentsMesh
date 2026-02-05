package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// SaveSession saves session info to Redis
func (s *RedisStore) SaveSession(ctx context.Context, session *ActiveSession) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Calculate TTL from expiry
	ttl := time.Until(session.ExpireAt)
	if ttl <= 0 {
		ttl = sessionDefaultTTL
	}

	// Save session data with TTL
	key := s.key(sessionKeyPrefix, session.PodKey)
	if err := s.cache.Client().Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	// Add to session list set
	if err := s.cache.Client().SAdd(ctx, s.key(sessionListKey), session.PodKey).Err(); err != nil {
		return fmt.Errorf("failed to add session to list: %w", err)
	}

	// Add to relay-specific session set
	relaySessionKey := s.key(sessionByRelayPrefix, session.RelayID)
	if err := s.cache.Client().SAdd(ctx, relaySessionKey, session.PodKey).Err(); err != nil {
		return fmt.Errorf("failed to add session to relay set: %w", err)
	}

	return nil
}

// GetSession retrieves session info from Redis
func (s *RedisStore) GetSession(ctx context.Context, podKey string) (*ActiveSession, error) {
	key := s.key(sessionKeyPrefix, podKey)
	data, err := s.cache.Client().Get(ctx, key).Bytes()
	if err != nil {
		if err.Error() == "redis: nil" {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session ActiveSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// GetAllSessions retrieves all sessions from Redis
func (s *RedisStore) GetAllSessions(ctx context.Context) ([]*ActiveSession, error) {
	// Get all session pod keys from the set
	podKeys, err := s.cache.Client().SMembers(ctx, s.key(sessionListKey)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get session list: %w", err)
	}

	sessions := make([]*ActiveSession, 0, len(podKeys))
	for _, pk := range podKeys {
		session, err := s.GetSession(ctx, pk)
		if err != nil {
			continue
		}
		if session != nil {
			sessions = append(sessions, session)
		} else {
			// Session expired, remove from set
			s.cache.Client().SRem(ctx, s.key(sessionListKey), pk)
		}
	}

	return sessions, nil
}

// GetSessionsByRelay retrieves all sessions for a specific relay
func (s *RedisStore) GetSessionsByRelay(ctx context.Context, relayID string) ([]*ActiveSession, error) {
	relaySessionKey := s.key(sessionByRelayPrefix, relayID)
	podKeys, err := s.cache.Client().SMembers(ctx, relaySessionKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get relay sessions: %w", err)
	}

	sessions := make([]*ActiveSession, 0, len(podKeys))
	for _, pk := range podKeys {
		session, err := s.GetSession(ctx, pk)
		if err != nil {
			continue
		}
		if session != nil {
			sessions = append(sessions, session)
		} else {
			// Session expired, remove from set
			s.cache.Client().SRem(ctx, relaySessionKey, pk)
		}
	}

	return sessions, nil
}

// DeleteSession removes session from Redis
func (s *RedisStore) DeleteSession(ctx context.Context, podKey string) error {
	// Get session first to find relay ID
	session, _ := s.GetSession(ctx, podKey)

	// Delete session data
	key := s.key(sessionKeyPrefix, podKey)
	if err := s.cache.Client().Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Remove from session list
	s.cache.Client().SRem(ctx, s.key(sessionListKey), podKey)

	// Remove from relay-specific set if we know the relay
	if session != nil {
		relaySessionKey := s.key(sessionByRelayPrefix, session.RelayID)
		s.cache.Client().SRem(ctx, relaySessionKey, podKey)
	}

	return nil
}

// UpdateSessionExpiry updates the expiry time for a session
func (s *RedisStore) UpdateSessionExpiry(ctx context.Context, podKey string, expiry time.Time) error {
	session, err := s.GetSession(ctx, podKey)
	if err != nil || session == nil {
		return nil
	}

	session.ExpireAt = expiry
	return s.SaveSession(ctx, session)
}
