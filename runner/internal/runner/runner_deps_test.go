package runner

import (
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
)

// --- New() validation tests ---

func TestNewRequiresConfig(t *testing.T) {
	_, err := New(RunnerDeps{
		Connection: client.NewMockConnection(),
	})
	if err == nil {
		t.Error("New() should fail without Config")
	}
}

func TestNewRequiresConnection(t *testing.T) {
	_, err := New(RunnerDeps{
		Config: &config.Config{WorkspaceRoot: t.TempDir()},
	})
	if err == nil {
		t.Error("New() should fail without Connection")
	}
}

func TestNewDefaultsPodStore(t *testing.T) {
	mockConn := client.NewMockConnection()
	r, err := New(RunnerDeps{
		Config:     &config.Config{WorkspaceRoot: t.TempDir()},
		Connection: mockConn,
	})
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	if r.podStore == nil {
		t.Error("podStore should default to InMemoryPodStore")
	}
	if r.podStore.Count() != 0 {
		t.Errorf("default podStore should be empty, got count=%d", r.podStore.Count())
	}
}

func TestNewInitializesAllComponents(t *testing.T) {
	r, _ := NewTestRunner(t)

	if r.autopilotStore == nil {
		t.Error("autopilotStore should be initialized")
	}
	if r.upgradeCoord == nil {
		t.Error("upgradeCoord should be initialized")
	}
	if r.sidecars == nil {
		t.Error("sidecars should be initialized")
	}
	if r.messageHandler == nil {
		t.Error("messageHandler should be initialized")
	}
	if r.stopChan == nil {
		t.Error("stopChan should be initialized")
	}
}

func TestNewWithCustomPodStore(t *testing.T) {
	customStore := NewInMemoryPodStore()
	customStore.Put("existing-pod", &Pod{PodKey: "existing-pod"})

	r, _ := NewTestRunner(t, WithTestPodStore(customStore))

	if r.podStore.Count() != 1 {
		t.Errorf("custom podStore should have 1 pod, got %d", r.podStore.Count())
	}
}

func TestNewTestRunnerDefaults(t *testing.T) {
	r, mc := NewTestRunner(t)

	if r.conn != mc {
		t.Error("connection should match MockConnection")
	}
	if r.cfg.NodeID != "test-node" {
		t.Errorf("NodeID = %q, want test-node", r.cfg.NodeID)
	}
	if r.cfg.MaxConcurrentPods != 10 {
		t.Errorf("MaxConcurrentPods = %d, want 10", r.cfg.MaxConcurrentPods)
	}
}

func TestNewTestRunnerWithCustomConfig(t *testing.T) {
	customCfg := &config.Config{
		WorkspaceRoot:     t.TempDir(),
		MaxConcurrentPods: 3,
		NodeID:            "custom-node",
	}
	r, _ := NewTestRunner(t, WithTestConfig(customCfg))

	if r.cfg.MaxConcurrentPods != 3 {
		t.Errorf("MaxConcurrentPods = %d, want 3", r.cfg.MaxConcurrentPods)
	}
	if r.cfg.NodeID != "custom-node" {
		t.Errorf("NodeID = %q, want custom-node", r.cfg.NodeID)
	}
}
