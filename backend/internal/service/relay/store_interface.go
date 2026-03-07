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
