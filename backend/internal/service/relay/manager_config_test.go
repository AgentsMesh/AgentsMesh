package relay

import (
	"context"
	"fmt"
	"testing"
)

func TestRegisterPersistenceFailure(t *testing.T) {
	failStore := &FailingMockStore{}
	m := newTestManager(t, WithStore(failStore))

	relay := &RelayInfo{ID: "relay-1", URL: "wss://r1.com"}
	err := m.Register(relay)

	if err == nil {
		t.Error("Register should return error when persistence fails")
	}

	// Verify relay was NOT added to memory (persistence-first pattern)
	if m.GetRelayByID("relay-1") != nil {
		t.Error("relay should not be in memory when persistence fails")
	}
}

// FailingMockStore is a mock store that always fails on SaveRelay
type FailingMockStore struct {
	MockStore
}

func (s *FailingMockStore) SaveRelay(ctx context.Context, relay *RelayInfo) error {
	return fmt.Errorf("simulated persistence failure")
}
