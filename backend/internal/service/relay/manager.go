package relay

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ErrCapacityLimitReached is returned when the relay count exceeds maxRelayCount.
var ErrCapacityLimitReached = errors.New("relay capacity limit reached")

// storeOpTimeout is the default timeout for persistent store operations.
// Keep short to avoid blocking request-serving goroutines on store failures.
const storeOpTimeout = 2 * time.Second

// Manager manages relay servers
type Manager struct {
	relays map[string]*RelayInfo // relayID -> info (in-memory cache)
	mu     sync.RWMutex

	healthCheckInterval time.Duration

	// Optional persistent store (Redis)
	store Store

	// Lifecycle management
	stopCh  chan struct{}
	stopped bool
	wg      sync.WaitGroup // tracks background goroutines (healthCheckLoop)

	logger *slog.Logger
}

// ManagerOption is a functional option for Manager
type ManagerOption func(*Manager)

// WithStore sets a persistent store for the manager
func WithStore(store Store) ManagerOption {
	return func(m *Manager) {
		m.store = store
	}
}

// WithHealthCheckInterval sets the interval for health checks
func WithHealthCheckInterval(interval time.Duration) ManagerOption {
	return func(m *Manager) {
		m.healthCheckInterval = interval
	}
}

// NewManagerWithOptions creates a new relay manager with options
func NewManagerWithOptions(opts ...ManagerOption) *Manager {
	m := &Manager{
		relays:              make(map[string]*RelayInfo),
		healthCheckInterval: 30 * time.Second,
		stopCh:              make(chan struct{}),
		logger:              slog.With("component", "relay_manager"),
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	// Guard against invalid interval that would panic time.NewTicker
	if m.healthCheckInterval <= 0 {
		m.healthCheckInterval = 30 * time.Second
	}

	// Load from persistent store if available
	if m.store != nil {
		m.loadFromStore()
	}

	// Clean stale relays loaded from store before accepting new traffic.
	// Must run BEFORE starting healthCheckLoop to avoid concurrent doHealthCheck execution.
	if m.store != nil {
		m.doHealthCheck()
	}

	// Start background health check
	m.wg.Add(1)
	go m.healthCheckLoop()

	return m
}

// Stop gracefully stops the manager and waits for background goroutines to exit.
func (m *Manager) Stop() {
	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		return
	}
	m.stopped = true
	m.mu.Unlock()

	close(m.stopCh)
	m.wg.Wait() // wait for healthCheckLoop to exit
	m.logger.Info("Relay manager stopped")
}

// IsStopped returns true if the manager has been stopped
func (m *Manager) IsStopped() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stopped
}

// loadFromStore loads relays from the persistent store
func (m *Manager) loadFromStore() {
	if m.store == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), storeOpTimeout*3) // startup can be slower
	defer cancel()

	relays, err := m.store.GetAllRelays(ctx)
	if err != nil {
		m.logger.Warn("Failed to load relays from store", "error", err)
		return
	}

	m.mu.Lock()
	for _, r := range relays {
		// Defensive copy: prevent aliasing with store-internal pointers
		relayCopy := *r
		m.relays[relayCopy.ID] = &relayCopy
	}
	m.mu.Unlock()
	m.logger.Info("Loaded relays from store", "count", len(relays))
}

// maxRelayCount is the safety cap on concurrent relay registrations.
// Prevents unbounded memory growth from misconfigured or malicious clients.
const maxRelayCount = 1000

// Register registers a new relay or updates existing one.
// The input is copied to prevent data races with the caller retaining the pointer.
// Returns error if persistence fails (when store is configured).
//
// Note: the capacity check uses RLock→check→RUnlock→Lock→insert pattern (TOCTOU).
// maxRelayCount is a soft safety cap, not a hard guarantee; concurrent registrations
// may briefly exceed it by the number of in-flight goroutines, which is acceptable.
func (m *Manager) Register(info *RelayInfo) error {
	if info.ID == "" {
		return fmt.Errorf("relay ID must not be empty")
	}
	if info.URL == "" {
		return fmt.Errorf("relay URL must not be empty")
	}

	// Copy input to prevent data race — caller may retain the original pointer
	infoCopy := *info
	infoCopy.LastHeartbeat = time.Now()
	infoCopy.Healthy = true

	// Check capacity limit (allow re-registration of existing relays)
	m.mu.RLock()
	_, isUpdate := m.relays[infoCopy.ID]
	relayCount := len(m.relays)
	m.mu.RUnlock()

	if !isUpdate && relayCount >= maxRelayCount {
		return ErrCapacityLimitReached
	}

	// Persist to store first (if configured)
	if m.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), storeOpTimeout)
		err := m.store.SaveRelay(ctx, &infoCopy)
		cancel() // release immediately, don't defer past the store call
		if err != nil {
			m.logger.Error("Failed to persist relay to store", "relay_id", infoCopy.ID, "error", err)
			return fmt.Errorf("failed to persist relay: %w", err)
		}
	}

	// Then update memory (store the copy, not the original pointer)
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
