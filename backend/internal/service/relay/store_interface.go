package relay

import (
	"context"
	"time"
)

// Store defines the interface for relay data persistence
type Store interface {
	// Relay operations
	SaveRelay(ctx context.Context, relay *RelayInfo) error
	GetRelay(ctx context.Context, relayID string) (*RelayInfo, error)
	GetAllRelays(ctx context.Context) ([]*RelayInfo, error)
	DeleteRelay(ctx context.Context, relayID string) error
	UpdateRelayHeartbeat(ctx context.Context, relayID string, heartbeat time.Time) error
}

const (
	// Redis key prefixes
	relayKeyPrefix       = "relay:info:"
	relayHeartbeatPrefix = "relay:heartbeat:"
	relayListKey         = "relay:list"

	// Default TTLs
	relayHeartbeatTTL = 60 * time.Second // Relay heartbeat expires after 60s
)

// MemoryStore implements Store interface using in-memory maps
// This is the current implementation (for backward compatibility)
type MemoryStore struct {
	// No state - the manager holds the data
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}
