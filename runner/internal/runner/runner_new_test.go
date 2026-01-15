package runner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/anthropics/agentmesh/runner/internal/client"
	"github.com/anthropics/agentmesh/runner/internal/config"
)

// TestNewRunnerWithMockConnection tests Runner creation with new Connection interface
func TestNewRunnerWithMockConnection(t *testing.T) {
	cfg := &config.Config{
		ServerURL:         "http://localhost:8080",
		NodeID:            "test-runner",
		AuthToken:         "test-token",
		OrgSlug:           "test-org",
		WorkspaceRoot:     t.TempDir(),
		MaxConcurrentPods: 5,
	}

	r, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	if r.conn == nil {
		t.Error("Runner connection should not be nil")
	}

	if r.podStore == nil {
		t.Error("Runner podStore should not be nil")
	}

	if r.messageHandler == nil {
		t.Error("Runner messageHandler should not be nil")
	}
}

// TestRunnerWithConnection tests WithConnection method
func TestRunnerWithConnection(t *testing.T) {
	cfg := &config.Config{
		ServerURL:         "http://localhost:8080",
		NodeID:            "test-runner",
		AuthToken:         "test-token",
		OrgSlug:           "test-org",
		WorkspaceRoot:     t.TempDir(),
		MaxConcurrentPods: 5,
	}

	r, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	mockConn := client.NewMockConnection()
	r.WithConnection(mockConn)

	if r.conn != mockConn {
		t.Error("Connection should be replaced with mock")
	}

	// Verify handler is set on mock connection
	// This is verified by simulating a message
}

// TestRunnerMessageHandlerOnListPods tests the MessageHandler interface
func TestRunnerMessageHandlerOnListPods(t *testing.T) {
	cfg := &config.Config{
		ServerURL:         "http://localhost:8080",
		NodeID:            "test-runner",
		AuthToken:         "test-token",
		OrgSlug:           "test-org",
		WorkspaceRoot:     t.TempDir(),
		MaxConcurrentPods: 5,
	}

	r, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	mockConn := client.NewMockConnection()
	r.WithConnection(mockConn)

	// Get pods (should be empty initially)
	pods := r.messageHandler.OnListPods()
	if len(pods) != 0 {
		t.Errorf("Expected 0 pods, got %d", len(pods))
	}
}

// TestBuildWebSocketBaseURL tests URL conversion (base URL without path)
func TestBuildWebSocketBaseURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"http://localhost:8080", "ws://localhost:8080"},
		{"https://api.example.com", "wss://api.example.com"},
		{"http://localhost:8080/", "ws://localhost:8080/"},
	}

	for _, tt := range tests {
		result := buildWebSocketBaseURL(tt.input)
		if result != tt.expected {
			t.Errorf("buildWebSocketBaseURL(%q) = %q, want %q", tt.input, result, tt.expected)
		}
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
		cfg:          cfg,
		podStore: NewInMemoryPodStore(),
	}
	mockConn := client.NewMockConnection()
	handler := NewRunnerMessageHandler(r, r.podStore, mockConn)

	var _ client.MessageHandler = handler
}

// --- Test Runner.Run ---

func TestRunnerRunMissingTokens(t *testing.T) {
	cfg := &config.Config{
		WorkspaceRoot: t.TempDir(),
		AuthToken:     "",
		RegistrationToken: "",
	}

	r := &Runner{
		cfg:          cfg,
		podStore: NewInMemoryPodStore(),
		stopChan:     make(chan struct{}),
	}

	mockConn := client.NewMockConnection()
	r.conn = mockConn

	ctx := context.Background()
	err := r.Run(ctx)

	if err == nil {
		t.Error("expected error for missing tokens")
	}
	if !contains(err.Error(), "no auth_token or registration_token") {
		t.Errorf("error = %v, want containing 'no auth_token or registration_token'", err)
	}
}

func TestRunnerRunWithAuthToken(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		WorkspaceRoot: tempDir,
		NodeID:        "test-node",
		AuthToken:     "test-token",
	}

	r := &Runner{
		cfg:          cfg,
		podStore: NewInMemoryPodStore(),
		stopChan:     make(chan struct{}),
	}

	mockConn := client.NewMockConnection()
	r.conn = mockConn

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

func TestRunnerRunWithRegistrationTokenError(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		WorkspaceRoot:     tempDir,
		NodeID:            "test-node",
		AuthToken:         "",
		RegistrationToken: "test-reg-token",
		ServerURL:         "http://localhost:9999", // Non-existent server
	}

	r := &Runner{
		cfg:          cfg,
		podStore: NewInMemoryPodStore(),
		stopChan:     make(chan struct{}),
	}

	mockConn := client.NewMockConnection()
	r.conn = mockConn

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := r.Run(ctx)
	// Should fail due to registration error
	if err == nil {
		t.Error("expected error for registration failure")
	}
	if !contains(err.Error(), "registration failed") {
		t.Errorf("error = %v, want containing 'registration failed'", err)
	}
}

func TestRunnerRunStopAllPods(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		WorkspaceRoot: tempDir,
		NodeID:        "test-node",
		AuthToken:     "test-token",
	}

	store := NewInMemoryPodStore()
	store.Put("pod-1", &Pod{ID: "pod-1", PodKey: "pod-1"})

	r := &Runner{
		cfg:          cfg,
		podStore: store,
		stopChan:     make(chan struct{}),
	}

	mockConn := client.NewMockConnection()
	r.conn = mockConn

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	r.Run(ctx)

	// Verify pods were cleaned up
	if store.Count() != 0 {
		t.Errorf("pod count = %d, want 0", store.Count())
	}
}

// --- Test initEnhancedComponents ---

// Note: TestInitEnhancedComponentsWithWorktree removed - worktree functionality
// is now handled by PodBuilder.setupWorkDir based on WorkDirConfig from Backend.

func TestInitEnhancedComponentsWithMCPConfig(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		WorkspaceRoot: tempDir,
		MCPConfigPath: "/nonexistent/mcp.json", // Non-existent file - should log warning but not fail
	}

	r := &Runner{
		cfg:      cfg,
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
		cfg:      cfg,
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
		cfg:      cfg,
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
		ID:         "pod-1",
		PodKey: "pod-1",
		Terminal:   nil,
	})
	store.Put("pod-2", &Pod{
		ID:         "pod-2",
		PodKey: "pod-2",
		Terminal:   nil,
	})

	r := &Runner{
		cfg:          &config.Config{},
		podStore: store,
	}

	r.stopAllPods()

	if store.Count() != 0 {
		t.Errorf("pod count = %d, want 0", store.Count())
	}
}

// --- Test buildWebSocketBaseURL edge cases ---

func TestBuildWebSocketBaseURLPlainURL(t *testing.T) {
	result := buildWebSocketBaseURL("localhost:8080")
	expected := "localhost:8080"
	if result != expected {
		t.Errorf("buildWebSocketBaseURL(plain) = %s, want %s", result, expected)
	}
}

// --- Test MockConnection helpers ---

func TestMockConnectionSimulateCreatePod(t *testing.T) {
	mockConn := client.NewMockConnection()
	store := NewInMemoryPodStore()

	tempDir := t.TempDir()
	r := &Runner{
		cfg: &config.Config{
			WorkspaceRoot:         tempDir,
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
		ID:         "terminate-mock",
		PodKey: "terminate-mock",
		Terminal:   nil,
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
		ID:         "list-pod",
		PodKey: "list-pod",
		Status:     PodStatusRunning,
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

	// Send some events
	mockConn.SendEvent(client.MsgTypePodCreated, map[string]string{"test": "data"})
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

func TestMockConnectionSendWithBackpressureWhenStopped(t *testing.T) {
	mockConn := client.NewMockConnection()
	mockConn.Stop()

	msg := client.ProtocolMessage{Type: "test"}
	ok := mockConn.SendWithBackpressure(msg)
	if ok {
		t.Error("SendWithBackpressure should return false when stopped")
	}
}

func TestMockConnectionQueueLength(t *testing.T) {
	mockConn := client.NewMockConnection()

	if mockConn.QueueLength() != 0 {
		t.Errorf("initial queue length = %d, want 0", mockConn.QueueLength())
	}

	mockConn.Send(client.ProtocolMessage{Type: "test1"})
	mockConn.Send(client.ProtocolMessage{Type: "test2"})

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
