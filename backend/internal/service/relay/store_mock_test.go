package relay

import (
	"context"
	"testing"
	"time"
)

// MockStore implements Store interface for testing
type MockStore struct {
	relays map[string]*RelayInfo
}

func NewMockStore() *MockStore {
	return &MockStore{
		relays: make(map[string]*RelayInfo),
	}
}

func (s *MockStore) SaveRelay(ctx context.Context, relay *RelayInfo) error {
	s.relays[relay.ID] = relay
	return nil
}

func (s *MockStore) GetRelay(ctx context.Context, relayID string) (*RelayInfo, error) {
	if r, ok := s.relays[relayID]; ok {
		return r, nil
	}
	return nil, nil
}

func (s *MockStore) GetAllRelays(ctx context.Context) ([]*RelayInfo, error) {
	result := make([]*RelayInfo, 0, len(s.relays))
	for _, r := range s.relays {
		result = append(result, r)
	}
	return result, nil
}

func (s *MockStore) DeleteRelay(ctx context.Context, relayID string) error {
	delete(s.relays, relayID)
	return nil
}

func (s *MockStore) UpdateRelayHeartbeat(ctx context.Context, relayID string, heartbeat time.Time) error {
	if r, ok := s.relays[relayID]; ok {
		r.LastHeartbeat = heartbeat
		r.Healthy = true
	}
	return nil
}

// === Tests for Store interface types ===

func TestNewMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	if store == nil {
		t.Fatal("NewMemoryStore returned nil")
	}
}
