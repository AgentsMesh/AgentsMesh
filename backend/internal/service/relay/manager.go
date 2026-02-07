package relay

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Manager manages relay servers
type Manager struct {
	relays map[string]*RelayInfo // relayID -> info (in-memory cache)
	mu     sync.RWMutex

	healthCheckInterval time.Duration
	loadBalancingConfig LoadBalancingConfig

	// Optional persistent store (Redis)
	store Store

	// Lifecycle management
	stopCh  chan struct{}
	stopped bool

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

// WithLoadBalancingConfig sets the load balancing configuration
func WithLoadBalancingConfig(cfg LoadBalancingConfig) ManagerOption {
	return func(m *Manager) {
		m.loadBalancingConfig = cfg
	}
}

// WithHealthCheckInterval sets the interval for health checks
func WithHealthCheckInterval(interval time.Duration) ManagerOption {
	return func(m *Manager) {
		m.healthCheckInterval = interval
	}
}

// NewManager creates a new relay manager
func NewManager() *Manager {
	return NewManagerWithOptions()
}

// NewManagerWithConfig creates a new relay manager with custom load balancing config
// Deprecated: Use NewManagerWithOptions instead
func NewManagerWithConfig(lbConfig LoadBalancingConfig) *Manager {
	return NewManagerWithOptions(WithLoadBalancingConfig(lbConfig))
}

// NewManagerWithOptions creates a new relay manager with options
func NewManagerWithOptions(opts ...ManagerOption) *Manager {
	m := &Manager{
		relays:              make(map[string]*RelayInfo),
		healthCheckInterval: 30 * time.Second,
		loadBalancingConfig: DefaultLoadBalancingConfig(),
		stopCh:              make(chan struct{}),
		logger:              slog.With("component", "relay_manager"),
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	// Load from persistent store if available
	if m.store != nil {
		m.loadFromStore()
	}

	// Start background health check
	go m.healthCheckLoop()

	return m
}

// Stop gracefully stops the manager and its background goroutines
func (m *Manager) Stop() {
	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		return
	}
	m.stopped = true
	m.mu.Unlock()

	close(m.stopCh)
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

	ctx := context.Background()

	// Load relays
	relays, err := m.store.GetAllRelays(ctx)
	if err != nil {
		m.logger.Warn("Failed to load relays from store", "error", err)
	} else {
		m.mu.Lock()
		for _, r := range relays {
			m.relays[r.ID] = r
		}
		m.mu.Unlock()
		m.logger.Info("Loaded relays from store", "count", len(relays))
	}
}

// Register registers a new relay or updates existing one
// Returns error if persistence fails (when store is configured)
func (m *Manager) Register(info *RelayInfo) error {
	info.LastHeartbeat = time.Now()
	info.Healthy = true

	// Persist to store first (if configured)
	if m.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.store.SaveRelay(ctx, info); err != nil {
			m.logger.Error("Failed to persist relay to store", "relay_id", info.ID, "error", err)
			return fmt.Errorf("failed to persist relay: %w", err)
		}
	}

	// Then update memory
	m.mu.Lock()
	m.relays[info.ID] = info
	m.mu.Unlock()

	m.logger.Info("Relay registered",
		"relay_id", info.ID,
		"url", info.URL,
		"region", info.Region,
		"capacity", info.Capacity)

	return nil
}

// Heartbeat updates relay health status
func (m *Manager) Heartbeat(relayID string, connections int, cpuUsage, memoryUsage float64) error {
	return m.HeartbeatWithLatency(relayID, connections, cpuUsage, memoryUsage, 0)
}

// HeartbeatWithLatency updates relay health status including latency metric
func (m *Manager) HeartbeatWithLatency(relayID string, connections int, cpuUsage, memoryUsage float64, latencyMs int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	relay, ok := m.relays[relayID]
	if !ok {
		return fmt.Errorf("relay not found: %s", relayID)
	}

	relay.CurrentConnections = connections
	relay.CPUUsage = cpuUsage
	relay.MemoryUsage = memoryUsage
	relay.LastHeartbeat = time.Now()
	relay.Healthy = true

	// Update latency with exponential moving average for smoothing
	if latencyMs > 0 {
		if relay.AvgLatencyMs == 0 {
			relay.AvgLatencyMs = latencyMs
		} else {
			// EMA with alpha = 0.3 for moderate smoothing
			relay.AvgLatencyMs = int(float64(relay.AvgLatencyMs)*0.7 + float64(latencyMs)*0.3)
		}
	}

	return nil
}

// Unregister removes a relay
func (m *Manager) Unregister(relayID string) {
	m.mu.Lock()
	delete(m.relays, relayID)
	m.mu.Unlock()

	// Remove from store
	if m.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.store.DeleteRelay(ctx, relayID); err != nil {
			m.logger.Warn("Failed to delete relay from store", "relay_id", relayID, "error", err)
		}
	}

	m.logger.Info("Relay unregistered", "relay_id", relayID)
}

// ForceUnregister removes a relay
func (m *Manager) ForceUnregister(relayID string) {
	m.mu.Lock()
	delete(m.relays, relayID)
	m.mu.Unlock()

	// Remove from store
	if m.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := m.store.DeleteRelay(ctx, relayID); err != nil {
			m.logger.Warn("Failed to delete relay from store", "relay_id", relayID, "error", err)
		}
	}

	m.logger.Info("Relay force unregistered", "relay_id", relayID)
}

// GracefulUnregister marks a relay as offline (graceful shutdown from relay itself)
func (m *Manager) GracefulUnregister(relayID string, reason string) {
	m.mu.Lock()

	relay, ok := m.relays[relayID]
	if !ok {
		m.mu.Unlock()
		return
	}

	// Mark as unhealthy so no new connections are assigned
	relay.Healthy = false

	// Remove the relay
	delete(m.relays, relayID)
	m.mu.Unlock()

	// Remove from store
	if m.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := m.store.DeleteRelay(ctx, relayID); err != nil {
			m.logger.Warn("Failed to delete relay from store", "relay_id", relayID, "error", err)
		}
	}

	m.logger.Info("Relay gracefully unregistered",
		"relay_id", relayID,
		"reason", reason)
}
