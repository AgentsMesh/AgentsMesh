package relay

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

var ErrCapacityLimitReached = errors.New("relay capacity limit reached")

const storeOpTimeout = 2 * time.Second

type Manager struct {
	relays map[string]*RelayInfo
	mu     sync.RWMutex

	healthCheckInterval time.Duration

	store Store

	stopCh  chan struct{}
	stopped bool
	wg      sync.WaitGroup

	logger *slog.Logger
}

type ManagerOption func(*Manager)

func WithStore(store Store) ManagerOption {
	return func(m *Manager) {
		m.store = store
	}
}

func WithHealthCheckInterval(interval time.Duration) ManagerOption {
	return func(m *Manager) {
		m.healthCheckInterval = interval
	}
}

func NewManagerWithOptions(opts ...ManagerOption) *Manager {
	m := &Manager{
		relays:              make(map[string]*RelayInfo),
		healthCheckInterval: 30 * time.Second,
		stopCh:              make(chan struct{}),
		logger:              slog.With("component", "relay_manager"),
	}

	for _, opt := range opts {
		opt(m)
	}

	if m.healthCheckInterval <= 0 {
		m.healthCheckInterval = 30 * time.Second
	}

	if m.store != nil {
		m.loadFromStore()
	}

	// Must run BEFORE healthCheckLoop starts — concurrent doHealthCheck would race.
	if m.store != nil {
		m.doHealthCheck()
	}

	m.wg.Add(1)
	go m.healthCheckLoop()

	return m
}

func (m *Manager) Stop() {
	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		return
	}
	m.stopped = true
	m.mu.Unlock()

	close(m.stopCh)
	m.wg.Wait()
	m.logger.Info("Relay manager stopped")
}

func (m *Manager) IsStopped() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stopped
}

func (m *Manager) loadFromStore() {
	if m.store == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), storeOpTimeout*3)
	defer cancel()

	relays, err := m.store.GetAllRelays(ctx)
	if err != nil {
		m.logger.Warn("Failed to load relays from store", "error", err)
		return
	}

	m.mu.Lock()
	for _, r := range relays {
		relayCopy := *r
		m.relays[relayCopy.ID] = &relayCopy
	}
	m.mu.Unlock()
	m.logger.Info("Loaded relays from store", "count", len(relays))
}

const maxRelayCount = 1000

func (m *Manager) Register(info *RelayInfo) error {
	if info.ID == "" {
		return fmt.Errorf("relay ID must not be empty")
	}
	if info.URL == "" {
		return fmt.Errorf("relay URL must not be empty")
	}

	infoCopy := *info
	infoCopy.LastHeartbeat = time.Now()
	infoCopy.Healthy = true

	m.mu.RLock()
	_, isUpdate := m.relays[infoCopy.ID]
	relayCount := len(m.relays)
	m.mu.RUnlock()

	if !isUpdate && relayCount >= maxRelayCount {
		return ErrCapacityLimitReached
	}

	if m.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), storeOpTimeout)
		err := m.store.SaveRelay(ctx, &infoCopy)
		cancel() // release immediately, don't defer past the store call
		if err != nil {
			m.logger.Error("Failed to persist relay to store", "relay_id", infoCopy.ID, "error", err)
			return fmt.Errorf("failed to persist relay: %w", err)
		}
	}

	m.mu.Lock()
	m.relays[infoCopy.ID] = &infoCopy
	m.mu.Unlock()

	m.logger.Info("Relay registered",
		"relay_id", infoCopy.ID,
		"url", infoCopy.URL,
		"region", infoCopy.Region,
		"capacity", infoCopy.Capacity)

	return nil
}
