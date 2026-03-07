package runner

import (
	"runtime"
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
	// Note: InitialPrompt field has been removed - prompt is now passed via LaunchArgs by Backend
	pod := Pod{
		ID:               "pod-1",
		PodKey:           "key-123",
		AgentType:        "claude-code",
		Branch:           "main",
		SandboxPath:      "/workspace/worktrees/pod-1",
		Terminal:         nil,
		StartedAt:        now,
		Status:           PodStatusRunning,
		TicketSlug: "TICKET-123",
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
	if pod.TicketSlug != "TICKET-123" {
		t.Errorf("TicketSlug: got %v, want TICKET-123", pod.TicketSlug)
	}
}

func TestPodAllFields(t *testing.T) {
	now := time.Now()

	// Note: InitialPrompt field has been removed - prompt is now passed via LaunchArgs by Backend
	// Note: OnOutput and OnExit fields have been removed - callbacks are now set via Terminal
	pod := &Pod{
		ID:               "id-1",
		PodKey:           "key-1",
		AgentType:        "claude-code",
		Branch:           "feature/test",
		SandboxPath:      "/workspace/worktrees/test",
		Terminal:         nil,
		StartedAt:        now,
		Status:           PodStatusRunning,
		TicketSlug: "TICKET-123",
	}

	if pod.ID != "id-1" {
		t.Errorf("ID: got %v, want id-1", pod.ID)
	}
	if pod.PodKey != "key-1" {
		t.Errorf("PodKey: got %v, want key-1", pod.PodKey)
	}
	if pod.GetStatus() != PodStatusRunning {
		t.Errorf("Status: got %v, want running", pod.GetStatus())
	}
}

// --- Test Runner Struct ---

// TestNewRunnerRequiresGRPC verifies that New() requires gRPC configuration.
func TestNewRunnerRequiresGRPC(t *testing.T) {
	// Isolate HOME to prevent loading existing gRPC certificates.
	// os.UserHomeDir() checks USERPROFILE first on Windows, HOME on Unix.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmpHome)
	}

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
	if err != nil && !contains(err.Error(), "gRPC configuration is required") {
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
