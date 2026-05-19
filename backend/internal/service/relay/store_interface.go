package relay

import (
	"context"
	"time"
)

type Store interface {
	SaveRelay(ctx context.Context, relay *RelayInfo) error
	GetRelay(ctx context.Context, relayID string) (*RelayInfo, error)
	GetAllRelays(ctx context.Context) ([]*RelayInfo, error)
	DeleteRelay(ctx context.Context, relayID string) error
	UpdateRelayHeartbeat(ctx context.Context, relayID string, heartbeat time.Time) error
}
