package runner

import (
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/config"
)

// --- Test Constants ---

func TestPodStatusConstantsBase(t *testing.T) {
	if PodStatusInitializing != "initializing" {
		t.Errorf("PodStatusInitializing: got %v, want initializing", PodStatusInitializing)
	}
	if PodStatusRunning != "running" {
		t.Errorf("PodStatusRunning: got %v, want running", PodStatusRunning)
	}
	if PodStatusStopped != "stopped" {
		t.Errorf("PodStatusStopped: got %v, want stopped", PodStatusStopped)
	}
	if PodStatusFailed != "failed" {
		t.Errorf("PodStatusFailed: got %v, want failed", PodStatusFailed)
	}
}

// --- Test Pod Struct ---

func TestPodStruct(t *testing.T) {
	now := time.Now()
	pod := Pod{
		ID:               "pod-1",
		PodKey:           "key-123",
		AgentType:        "claude-code",
		Branch:           "main",
		WorktreePath:     "/workspace/worktrees/pod-1",
		InitialPrompt:    "Hello",
		Terminal:         nil,
		StartedAt:        now,
		Status:           PodStatusRunning,
		TicketIdentifier: "TICKET-123",
	}

	if pod.ID != "pod-1" {
		t.Errorf("ID: got %v, want pod-1", pod.ID)
	}
	if pod.PodKey != "key-123" {
		t.Errorf("PodKey: got %v, want key-123", pod.PodKey)
	}
	if pod.AgentType != "claude-code" {
		t.Errorf("AgentType: got %v, want claude-code", pod.AgentType)
	}
	if pod.GetStatus() != PodStatusRunning {
		t.Errorf("Status: got %v, want running", pod.GetStatus())
	}
	if pod.TicketIdentifier != "TICKET-123" {
		t.Errorf("TicketIdentifier: got %v, want TICKET-123", pod.TicketIdentifier)
	}
}

func TestPodAllFields(t *testing.T) {
	now := time.Now()
	forwarder := &PTYForwarder{podKey: "test"}

	pod := &Pod{
		ID:               "id-1",
		PodKey:           "key-1",
		AgentType:        "claude-code",
		Branch:           "feature/test",
		WorktreePath:     "/workspace/worktrees/test",
		InitialPrompt:    "Hello, Claude!",
		Terminal:         nil,
		StartedAt:        now,
		Status:           PodStatusRunning,
		TicketIdentifier: "TICKET-123",
		OnOutput:         func([]byte) {},
		OnExit:           func(int) {},
		Forwarder:        forwarder,
	}

	if pod.OnOutput == nil {
		t.Error("OnOutput should not be nil")
	}
	if pod.OnExit == nil {
		t.Error("OnExit should not be nil")
	}
	if pod.Forwarder == nil {
		t.Error("Forwarder should not be nil")
	}
}

func TestPodWithCallbacks(t *testing.T) {
	outputCalled := false
	exitCalled := false

	pod := &Pod{
		ID:     "pod-1",
		Status: PodStatusRunning,
		OnOutput: func(data []byte) {
			outputCalled = true
		},
		OnExit: func(exitCode int) {
			exitCalled = true
		},
	}

	if pod.OnOutput != nil {
		pod.OnOutput([]byte("test"))
	}
	if pod.OnExit != nil {
		pod.OnExit(0)
	}

	if !outputCalled {
		t.Error("OnOutput should be called")
	}
	if !exitCalled {
		t.Error("OnExit should be called")
	}
}

// --- Test Runner Struct ---

// TestNewRunnerRequiresGRPC verifies that New() requires gRPC configuration.
func TestNewRunnerRequiresGRPC(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		ServerURL:         "http://localhost:8080",
		NodeID:            "test-runner",
		OrgSlug:           "test-org",
		WorkspaceRoot:     tempDir,
		MaxConcurrentPods: 10,
		// No gRPC config - should fail
	}

	r, err := New(cfg)
	if err == nil {
		t.Error("New should return error when gRPC config is missing")
	}
	if r != nil {
		t.Error("Runner should be nil when gRPC config is missing")
	}
	if !contains(err.Error(), "gRPC configuration is required") {
		t.Errorf("Error should mention gRPC configuration, got: %v", err)
	}
}

// TestRunnerConfigFields tests runner configuration fields using direct struct creation.
// Note: Full runner creation requires gRPC certificates, so we test config fields directly.
func TestRunnerConfigFields(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		ServerURL:         "https://localhost:8080",
		NodeID:            "test-runner",
		OrgSlug:           "test-org",
		WorkspaceRoot:     tempDir,
		MaxConcurrentPods: 5,
		GRPCEndpoint:      "localhost:9443",
		CertFile:          "/tmp/test.crt",
		KeyFile:           "/tmp/test.key",
		CAFile:            "/tmp/ca.crt",
		AgentEnvVars: map[string]string{
			"API_KEY": "test-key",
		},
	}

	// Create runner components directly for testing
	store := NewInMemoryPodStore()
	r := &Runner{
		cfg:      cfg,
		podStore: store,
		pods:     make(map[string]*Pod),
		stopChan: make(chan struct{}),
	}

	if r.cfg.WorkspaceRoot != tempDir {
		t.Errorf("WorkspaceRoot: got %v, want %v", r.cfg.WorkspaceRoot, tempDir)
	}
	if r.cfg.MaxConcurrentPods != 5 {
		t.Errorf("MaxConcurrentPods: got %v, want 5", r.cfg.MaxConcurrentPods)
	}
	if r.cfg.AgentEnvVars["API_KEY"] != "test-key" {
		t.Errorf("AgentEnvVars[API_KEY]: got %v, want test-key", r.cfg.AgentEnvVars["API_KEY"])
	}
	if r.cfg.GRPCEndpoint != "localhost:9443" {
		t.Errorf("GRPCEndpoint: got %v, want localhost:9443", r.cfg.GRPCEndpoint)
	}
}

// Note: InMemoryPodStore tests are in pod_store_test.go
