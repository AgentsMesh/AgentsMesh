package relay

import (
	"context"
	"fmt"
	"testing"
)

func TestLoadBalancingConfig(t *testing.T) {
	cfg := DefaultLoadBalancingConfig()

	if cfg.ConnectionWeight != 0.4 {
		t.Errorf("ConnectionWeight: got %f, want 0.4", cfg.ConnectionWeight)
	}
	if cfg.CPUWeight != 0.25 {
		t.Errorf("CPUWeight: got %f, want 0.25", cfg.CPUWeight)
	}
	if cfg.MemoryWeight != 0.15 {
		t.Errorf("MemoryWeight: got %f, want 0.15", cfg.MemoryWeight)
	}
	if cfg.LatencyWeight != 0.1 {
		t.Errorf("LatencyWeight: got %f, want 0.1", cfg.LatencyWeight)
	}
	if cfg.RegionBonus != 50 {
		t.Errorf("RegionBonus: got %f, want 50", cfg.RegionBonus)
	}
}

func TestNewManagerWithConfig(t *testing.T) {
	cfg := LoadBalancingConfig{
		ConnectionWeight: 0.5,
		CPUWeight:        0.3,
		MemoryWeight:     0.1,
		LatencyWeight:    0.05,
		RegionBonus:      100,
	}

	m := NewManagerWithConfig(cfg)
	if m == nil {
		t.Fatal("NewManagerWithConfig returned nil")
	}
	if m.loadBalancingConfig.ConnectionWeight != 0.5 {
		t.Errorf("config not applied: ConnectionWeight %f", m.loadBalancingConfig.ConnectionWeight)
	}
}

func TestNewManagerWithOptions(t *testing.T) {
	cfg := LoadBalancingConfig{
		ConnectionWeight: 0.6,
		RegionBonus:      200,
	}

	m := NewManagerWithOptions(WithLoadBalancingConfig(cfg))
	if m == nil {
		t.Fatal("NewManagerWithOptions returned nil")
	}
	if m.loadBalancingConfig.ConnectionWeight != 0.6 {
		t.Errorf("ConnectionWeight: got %f, want 0.6", m.loadBalancingConfig.ConnectionWeight)
	}
	if m.loadBalancingConfig.RegionBonus != 200 {
		t.Errorf("RegionBonus: got %f, want 200", m.loadBalancingConfig.RegionBonus)
	}
}

func TestRegisterPersistenceFailure(t *testing.T) {
	// Create a mock store that always fails
	failStore := &FailingMockStore{}
	m := NewManagerWithOptions(WithStore(failStore))

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
