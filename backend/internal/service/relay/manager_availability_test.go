package relay

import (
	"testing"
)

func TestIsRelayAvailable(t *testing.T) {
	tests := []struct {
		name     string
		relay    RelayInfo
		expected bool
	}{
		{"healthy", RelayInfo{Healthy: true, Capacity: 100}, true},
		{"unhealthy", RelayInfo{Healthy: false, Capacity: 100}, false},
		{"at capacity", RelayInfo{Healthy: true, Capacity: 100, CurrentConnections: 100}, false},
		{"at CPU threshold", RelayInfo{Healthy: true, Capacity: 100, CPUUsage: strictCPUThreshold}, true},
		{"high CPU", RelayInfo{Healthy: true, Capacity: 100, CPUUsage: strictCPUThreshold + 1}, false},
		{"at memory threshold", RelayInfo{Healthy: true, Capacity: 100, MemoryUsage: strictMemThreshold}, true},
		{"high memory", RelayInfo{Healthy: true, Capacity: 100, MemoryUsage: strictMemThreshold + 1}, false},
		{"no capacity limit", RelayInfo{Healthy: true, Capacity: 0, CurrentConnections: 999}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRelayAvailable(&tt.relay); got != tt.expected {
				t.Errorf("isRelayAvailable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsRelayReachable(t *testing.T) {
	tests := []struct {
		name     string
		relay    RelayInfo
		expected bool
	}{
		{"healthy normal", RelayInfo{Healthy: true, Capacity: 100}, true},
		{"unhealthy", RelayInfo{Healthy: false, Capacity: 100}, false},
		{"at capacity", RelayInfo{Healthy: true, Capacity: 100, CurrentConnections: 100}, false},
		{"high CPU still reachable", RelayInfo{Healthy: true, Capacity: 100, CPUUsage: 95}, true},
		{"high memory still reachable", RelayInfo{Healthy: true, Capacity: 100, MemoryUsage: 95}, true},
		{"no capacity limit", RelayInfo{Healthy: true, Capacity: 0, CurrentConnections: 999}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRelayReachable(&tt.relay); got != tt.expected {
				t.Errorf("isRelayReachable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSelectRelayWithAffinity_OverloadedFallback(t *testing.T) {
	m := newTestManager(t)
	_ = m.Register(&RelayInfo{
		ID: "relay-hot", URL: "wss://hot.com",
		Healthy: true, Capacity: 100, CPUUsage: 95,
	})

	result := m.SelectRelayWithAffinity("test-org")
	if result == nil {
		t.Fatal("overloaded relay should be selected via lenient fallback")
	}
	if result.ID != "relay-hot" {
		t.Errorf("expected relay-hot, got %q", result.ID)
	}
}
