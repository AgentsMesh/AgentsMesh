package relay

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockStore implements Store interface for testing.
// Thread-safe via mutex for use in concurrent tests.
type MockStore struct {
	mu           sync.Mutex
	relays       map[string]*RelayInfo
	deleteCalled int // tracks number of DeleteRelay calls
}

func NewMockStore() *MockStore {
	return &MockStore{
		relays: make(map[string]*RelayInfo),
	}
}

func (s *MockStore) SaveRelay(ctx context.Context, relay *RelayInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Store a copy to match RedisStore behavior (serialize/deserialize decouples pointers)
	relayCopy := *relay
	s.relays[relayCopy.ID] = &relayCopy
	return nil
}

func (s *MockStore) GetRelay(ctx context.Context, relayID string) (*RelayInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.relays[relayID]; ok {
		return r, nil
	}
	return nil, nil
}

func (s *MockStore) GetAllRelays(ctx context.Context) ([]*RelayInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]*RelayInfo, 0, len(s.relays))
	for _, r := range s.relays {
		// Return copies to match RedisStore's JSON serialize/deserialize behavior
		relayCopy := *r
		result = append(result, &relayCopy)
	}
	return result, nil
}

func (s *MockStore) DeleteRelay(ctx context.Context, relayID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteCalled++
	delete(s.relays, relayID)
	return nil
}

func (s *MockStore) UpdateRelayHeartbeat(ctx context.Context, relayID string, heartbeat time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.relays[relayID]; ok {
		r.LastHeartbeat = heartbeat
		r.Healthy = true
	}
	return nil
}

// FailingLoadStore is a mock store whose GetAllRelays always fails (tests loadFromStore error path).
type FailingLoadStore struct {
	MockStore
}

func (s *FailingLoadStore) GetAllRelays(ctx context.Context) ([]*RelayInfo, error) {
	return nil, fmt.Errorf("simulated store load failure")
}

// FailingHeartbeatStore is a mock store whose UpdateRelayHeartbeat always fails.
type FailingHeartbeatStore struct {
	MockStore
}

func (s *FailingHeartbeatStore) UpdateRelayHeartbeat(ctx context.Context, relayID string, heartbeat time.Time) error {
	return fmt.Errorf("simulated heartbeat store failure")
}
