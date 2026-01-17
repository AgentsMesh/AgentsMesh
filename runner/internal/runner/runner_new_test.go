package runner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
)

// TestRunnerWithConnection tests WithConnection method
func TestRunnerWithConnection(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		ServerURL:         "https://localhost:8080",
		NodeID:            "test-runner",
		OrgSlug:           "test-org",
		WorkspaceRoot:     tempDir,
		MaxConcurrentPods: 5,
		// gRPC config will be loaded by the mock
		GRPCEndpoint: "localhost:9443",
		CertFile:     "/tmp/test.crt",
		KeyFile:      "/tmp/test.key",
		CAFile:       "/tmp/ca.crt",
	}

	// Create runner components manually since we're testing with mock
	store := NewInMemoryPodStore()
	r := &Runner{
		cfg:      cfg,
		podStore: store,
		pods:     make(map[string]*Pod),
		stopChan: make(chan struct{}),
	}

	mockConn := client.NewMockConnection()
	r.WithConnection(mockConn)

	if r.conn != mockConn {
		t.Error("Connection should be replaced with mock")
	}
}

// TestRunnerMessageHandlerOnListPods tests the MessageHandler interface
func TestRunnerMessageHandlerOnListPods(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		ServerURL:         "https://localhost:8080",
		NodeID:            "test-runner",
		OrgSlug:           "test-org",
		WorkspaceRoot:     tempDir,
		MaxConcurrentPods: 5,
	}

	store := NewInMemoryPodStore()
	r := &Runner{
		cfg:      cfg,
		podStore: store,
		pods:     make(map[string]*Pod),
	}

	mockConn := client.NewMockConnection()
	r.conn = mockConn
	r.messageHandler = NewRunnerMessageHandler(r, store, mockConn)

	// Get pods (should be empty initially)
	pods := r.messageHandler.OnListPods()
	if len(pods) != 0 {
		t.Errorf("Expected 0 pods, got %d", len(pods))
	}
}

// TestMockConnectionInterface tests that MockConnection implements Connection
func TestMockConnectionInterface(t *testing.T) {
	var _ client.Connection = client.NewMockConnection()
}

// TestRunnerMessageHandlerInterface tests that RunnerMessageHandler implements MessageHandler
func TestRunnerMessageHandlerInterface(t *testing.T) {
	cfg := &config.Config{
		WorkspaceRoot: t.TempDir(),
	}
	r := &Runner{
		cfg:      cfg,
		podStore: NewInMemoryPodStore(),
	}
	mockConn := client.NewMockConnection()
	handler := NewRunnerMessageHandler(r, r.podStore, mockConn)

	var _ client.MessageHandler = handler
}

// TestRunnerRunWithGRPCConnection tests runner with gRPC connection
func TestRunnerRunWithGRPCConnection(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		WorkspaceRoot: tempDir,
		NodeID:        "test-node",
		OrgSlug:       "test-org",
		GRPCEndpoint:  "localhost:9443",
		CertFile:      "/tmp/test.crt",
		KeyFile:       "/tmp/test.key",
		CAFile:        "/tmp/ca.crt",
	}

	store := NewInMemoryPodStore()
	r := &Runner{
		cfg:      cfg,
		podStore: store,
		pods:     make(map[string]*Pod),
		stopChan: make(chan struct{}),
	}

	mockConn := client.NewMockConnection()
	r.conn = mockConn
	r.messageHandler = NewRunnerMessageHandler(r, store, mockConn)

	// Run with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := r.Run(ctx)
	// Should exit cleanly on context cancellation
	if err != nil {
		t.Logf("Run returned: %v", err)
	}

	// Verify connection was started
	if !mockConn.IsStarted() {
		t.Error("connection should be started")
	}
}

func TestRunnerRunStopAllPods(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		WorkspaceRoot: tempDir,
		NodeID:        "test-node",
		OrgSlug:       "test-org",
	}

	store := NewInMemoryPodStore()
	store.Put("pod-1", &Pod{ID: "pod-1", PodKey: "pod-1"})

	r := &Runner{
		cfg:      cfg,
		podStore: store,
		pods:     make(map[string]*Pod),
		stopChan: make(chan struct{}),
	}

	mockConn := client.NewMockConnection()
	r.conn = mockConn
	r.messageHandler = NewRunnerMessageHandler(r, store, mockConn)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	r.Run(ctx)

	// Verify pods were cleaned up
	if store.Count() != 0 {
		t.Errorf("pod count = %d, want 0", store.Count())
	}
}

// --- Test initEnhancedComponents ---

func TestInitEnhancedComponentsWithMCPConfig(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		WorkspaceRoot: tempDir,
		MCPConfigPath: "/nonexistent/mcp.json", // Non-existent file - should log warning but not fail
	}

	r := &Runner{
		cfg:  cfg,
		pods: make(map[string]*Pod),
	}

	// Should not panic
	r.initEnhancedComponents(cfg)

	// MCP manager should still be initialized
	if r.mcpManager == nil {
		t.Error("mcpManager should be initialized")
	}
}

func TestInitEnhancedComponentsDefaultShell(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		WorkspaceRoot: tempDir,
		DefaultShell:  "", // Empty - should default to /bin/sh
	}

	r := &Runner{
		cfg:  cfg,
		pods: make(map[string]*Pod),
	}

	r.initEnhancedComponents(cfg)

	if r.termManager == nil {
		t.Error("termManager should be initialized")
	}
}

func TestInitEnhancedComponentsCustomShell(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		WorkspaceRoot: tempDir,
		DefaultShell:  "/bin/bash",
	}

	r := &Runner{
		cfg:  cfg,
		pods: make(map[string]*Pod),
	}

	r.initEnhancedComponents(cfg)

	if r.termManager == nil {
		t.Error("termManager should be initialized")
	}
}

// --- Test stopAllPods ---

func TestStopAllPodsWithTerminals(t *testing.T) {
	store := NewInMemoryPodStore()
	store.Put("pod-1", &Pod{
		ID:       "pod-1",
		PodKey:   "pod-1",
		Terminal: nil,
	})
	store.Put("pod-2", &Pod{
		ID:       "pod-2",
		PodKey:   "pod-2",
		Terminal: nil,
	})

	r := &Runner{
		cfg:      &config.Config{},
		podStore: store,
	}

	r.stopAllPods()

	if store.Count() != 0 {
		t.Errorf("pod count = %d, want 0", store.Count())
	}
}

// --- Test MockConnection helpers ---

func TestMockConnectionSimulateCreatePod(t *testing.T) {
	mockConn := client.NewMockConnection()
	store := NewInMemoryPodStore()

	tempDir := t.TempDir()
	r := &Runner{
		cfg: &config.Config{
			WorkspaceRoot:     tempDir,
			MaxConcurrentPods: 10,
		},
		podStore: store,
	}

	handler := NewRunnerMessageHandler(r, store, mockConn)
	mockConn.SetHandler(handler)

	req := client.CreatePodRequest{
		PodKey:        "mock-pod",
		LaunchCommand: "echo",
	}

	err := mockConn.SimulateCreatePod(req)
	if err != nil {
		t.Logf("SimulateCreatePod: %v", err)
	}

	// Clean up
	pod, ok := store.Get("mock-pod")
	if ok && pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

func TestMockConnectionSimulateTerminatePod(t *testing.T) {
	mockConn := client.NewMockConnection()
	store := NewInMemoryPodStore()

	r := &Runner{
		cfg: &config.Config{},
	}

	store.Put("terminate-mock", &Pod{
		ID:       "terminate-mock",
		PodKey:   "terminate-mock",
		Terminal: nil,
	})

	handler := NewRunnerMessageHandler(r, store, mockConn)
	mockConn.SetHandler(handler)

	req := client.TerminatePodRequest{
		PodKey: "terminate-mock",
	}

	err := mockConn.SimulateTerminatePod(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	_, exists := store.Get("terminate-mock")
	if exists {
		t.Error("pod should be removed")
	}
}

func TestMockConnectionGetPods(t *testing.T) {
	mockConn := client.NewMockConnection()
	store := NewInMemoryPodStore()

	r := &Runner{cfg: &config.Config{}}

	store.Put("list-pod", &Pod{
		ID:     "list-pod",
		PodKey: "list-pod",
		Status: PodStatusRunning,
	})

	handler := NewRunnerMessageHandler(r, store, mockConn)
	mockConn.SetHandler(handler)

	pods := mockConn.GetPods()
	if len(pods) != 1 {
		t.Errorf("pods count = %d, want 1", len(pods))
	}
}

func TestMockConnectionReset(t *testing.T) {
	mockConn := client.NewMockConnection()

	// Send some events using new methods
	mockConn.SendPodCreated("test-pod", 123)
	mockConn.Start()

	// Verify state
	if len(mockConn.GetEvents()) == 0 {
		t.Error("should have events before reset")
	}

	// Reset
	mockConn.Reset()

	// Verify state is cleared
	if len(mockConn.GetEvents()) != 0 {
		t.Errorf("events count after reset = %d, want 0", len(mockConn.GetEvents()))
	}
	if mockConn.IsStarted() {
		t.Error("should not be started after reset")
	}
}

func TestMockConnectionConnectError(t *testing.T) {
	mockConn := client.NewMockConnection()
	mockConn.ConnectErr = errors.New("connection refused")

	err := mockConn.Connect()
	if err == nil {
		t.Error("expected error for ConnectErr")
	}
	if !contains(err.Error(), "connection refused") {
		t.Errorf("error = %v, want containing 'connection refused'", err)
	}
}

func TestMockConnectionQueueLength(t *testing.T) {
	mockConn := client.NewMockConnection()

	if mockConn.QueueLength() != 0 {
		t.Errorf("initial queue length = %d, want 0", mockConn.QueueLength())
	}

	mockConn.SendPodCreated("test1", 1)
	mockConn.SendPodCreated("test2", 2)

	if mockConn.QueueLength() != 2 {
		t.Errorf("queue length = %d, want 2", mockConn.QueueLength())
	}
}

func TestMockConnectionQueueCapacity(t *testing.T) {
	mockConn := client.NewMockConnection()

	if mockConn.QueueCapacity() != 100 {
		t.Errorf("queue capacity = %d, want 100", mockConn.QueueCapacity())
	}
}

// Note: contains helper is defined in mocks_test.go
