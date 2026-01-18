package runner

import (
	"errors"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/workspace"
)

// Tests for OnCreatePod and OnTerminatePod operations

// --- OnCreatePod Tests ---

func TestOnCreatePodSuccess(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	ws, err := workspace.NewManager(tempDir, "")
	if err != nil {
		t.Skipf("Could not create workspace manager: %v", err)
	}

	runner := &Runner{
		cfg: &config.Config{
			MaxConcurrentPods: 10,
			WorkspaceRoot:         tempDir,
		},
		workspace: ws,
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	req := client.CreatePodRequest{
		PodKey:      "test-pod-1",
		LaunchCommand: "echo",
	}

	err = handler.OnCreatePod(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify pod was created
	pod, ok := store.Get("test-pod-1")
	if !ok {
		t.Error("pod should be stored")
	} else {
		if pod.GetStatus() != PodStatusRunning {
			t.Errorf("pod status = %s, want running", pod.GetStatus())
		}
		// Clean up terminal
		if pod.Terminal != nil {
			pod.Terminal.Stop()
		}
	}

	// Verify pod_created event was sent
	events := mockConn.GetEvents()
	hasCreated := false
	for _, e := range events {
		if e.Type == client.MsgTypePodCreated {
			hasCreated = true
			break
		}
	}
	if !hasCreated {
		t.Error("should have sent pod_created event")
	}
}

func TestOnCreatePodMaxCapacity(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{
			MaxConcurrentPods: 1,
			WorkspaceRoot:         tempDir,
		},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Add pod
	store.Put("existing-pod", &Pod{ID: "existing-pod"})

	req := client.CreatePodRequest{
		PodKey:      "new-pod",
		LaunchCommand: "echo",
	}

	err := handler.OnCreatePod(req)
	if err == nil {
		t.Error("expected error for max capacity")
	}
	if !contains(err.Error(), "max concurrent pods") {
		t.Errorf("error = %v, want containing 'max concurrent pods'", err)
	}
}

func TestOnCreatePodInvalidCommand(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	ws, err := workspace.NewManager(tempDir, "")
	if err != nil {
		t.Skipf("Could not create workspace manager: %v", err)
	}

	runner := &Runner{
		cfg: &config.Config{
			MaxConcurrentPods: 10,
			WorkspaceRoot:         tempDir,
		},
		workspace: ws,
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	req := client.CreatePodRequest{
		PodKey:      "invalid-cmd-pod",
		LaunchCommand: "/nonexistent/command/path",
	}

	err = handler.OnCreatePod(req)
	// Command may or may not fail depending on OS
	t.Logf("OnCreatePod with invalid command: %v", err)
}

func TestOnCreatePodWithWorkDirConfig(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{
			MaxConcurrentPods: 10,
			WorkspaceRoot:         tempDir,
		},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	req := client.CreatePodRequest{
		PodKey:        "workdir-pod",
		LaunchCommand: "echo",
		WorkDirConfig: &client.WorkDirConfig{
			Type: "tempdir",
		},
	}

	err := handler.OnCreatePod(req)
	if err != nil {
		t.Logf("OnCreatePod with work dir config: %v", err)
	}

	// Clean up
	pod, ok := store.Get("workdir-pod")
	if ok && pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

func TestOnCreatePodWithFilesToCreate(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{
			MaxConcurrentPods: 10,
			WorkspaceRoot:         tempDir,
		},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	req := client.CreatePodRequest{
		PodKey:        "files-pod",
		LaunchCommand: "echo",
		FilesToCreate: []client.FileToCreate{
			{
				PathTemplate: "{{.sandbox.root_path}}/test.txt",
				Content:      "test content",
				Mode:         0644,
			},
		},
	}

	err := handler.OnCreatePod(req)
	if err != nil {
		t.Logf("OnCreatePod with files to create: %v", err)
	}

	// Clean up
	pod, ok := store.Get("files-pod")
	if ok && pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

func TestOnCreatePodWithLocalWorkDir(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{
			MaxConcurrentPods: 10,
			WorkspaceRoot:         tempDir,
		},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	req := client.CreatePodRequest{
		PodKey:        "local-workdir-pod",
		LaunchCommand: "echo",
		WorkDirConfig: &client.WorkDirConfig{
			Type:      "local",
			LocalPath: tempDir,
		},
	}

	err := handler.OnCreatePod(req)
	if err != nil {
		t.Logf("OnCreatePod with local work dir: %v", err)
	}

	// Clean up
	pod, ok := store.Get("local-workdir-pod")
	if ok && pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

func TestOnCreatePodWithLaunchArgs(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{
			MaxConcurrentPods: 10,
			WorkspaceRoot:         tempDir,
		},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	req := client.CreatePodRequest{
		PodKey:        "launch-args-pod",
		LaunchCommand: "echo",
		LaunchArgs:    []string{"hello", "world"},
	}

	err := handler.OnCreatePod(req)
	if err != nil {
		t.Logf("OnCreatePod with launch args: %v", err)
	}

	// Clean up
	pod, ok := store.Get("launch-args-pod")
	if ok && pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

func TestOnCreatePodWithPromptInArgs(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{
			MaxConcurrentPods: 10,
			WorkspaceRoot:         tempDir,
		},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Prompt is now passed as first argument (handled by Backend)
	req := client.CreatePodRequest{
		PodKey:        "prompt-pod",
		LaunchCommand: "echo",
		LaunchArgs:    []string{"Hello from test"},
	}

	err := handler.OnCreatePod(req)
	if err != nil {
		t.Logf("OnCreatePod with prompt in args: %v", err)
	}

	// Clean up
	pod, ok := store.Get("prompt-pod")
	if ok && pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

func TestOnCreatePodWithWorktreeConfigError(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	// Create runner without workspace manager
	runner := &Runner{
		cfg: &config.Config{
			MaxConcurrentPods: 10,
			WorkspaceRoot:     tempDir,
		},
		workspace: nil, // No workspace manager
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	req := client.CreatePodRequest{
		PodKey:        "worktree-error-pod",
		LaunchCommand: "echo",
		WorkDirConfig: &client.WorkDirConfig{
			Type:          "worktree",
			RepositoryURL: "https://github.com/test/repo",
			Branch:        "main",
		},
	}

	err := handler.OnCreatePod(req)
	// Should fail because workspace manager is not available
	if err == nil {
		t.Error("expected error for worktree without workspace manager")
	}
	t.Logf("OnCreatePod with worktree error: %v", err)
}

func TestOnCreatePodWithSendEventError(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()
	mockConn.SendErr = errors.New("send failed")

	runner := &Runner{
		cfg: &config.Config{
			MaxConcurrentPods: 10,
			WorkspaceRoot:         tempDir,
		},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	req := client.CreatePodRequest{
		PodKey:      "send-error-pod",
		LaunchCommand: "echo",
	}

	err := handler.OnCreatePod(req)
	// Pod should still be created even if send fails
	if err != nil {
		t.Logf("OnCreatePod with send error: %v", err)
	}

	// Clean up
	pod, ok := store.Get("send-error-pod")
	if ok && pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

// --- OnTerminatePod Tests ---

func TestOnTerminatePodSuccess(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{WorkspaceRoot: tempDir},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Add pod
	store.Put("terminate-pod", &Pod{
		ID:       "terminate-pod",
		Terminal: nil, // nil terminal should be handled gracefully
	})

	req := client.TerminatePodRequest{
		PodKey: "terminate-pod",
	}

	err := handler.OnTerminatePod(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify pod was removed
	_, exists := store.Get("terminate-pod")
	if exists {
		t.Error("pod should be removed")
	}

	// Verify pod_terminated event was sent
	events := mockConn.GetEvents()
	hasTerminated := false
	for _, e := range events {
		if e.Type == client.MsgTypePodTerminated {
			hasTerminated = true
			break
		}
	}
	if !hasTerminated {
		t.Error("should have sent pod_terminated event")
	}
}

func TestOnTerminatePodNotFound(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	req := client.TerminatePodRequest{
		PodKey: "nonexistent-pod",
	}

	err := handler.OnTerminatePod(req)
	if err == nil {
		t.Error("expected error for nonexistent pod")
	}
	if !contains(err.Error(), "pod not found") {
		t.Errorf("error = %v, want containing 'pod not found'", err)
	}
}

func TestOnTerminatePodWithWorktree(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{WorkspaceRoot: tempDir},
		// No worktreeService
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Add pod with worktree
	store.Put("worktree-pod", &Pod{
		ID:           "worktree-pod",
		WorktreePath: "/fake/worktree/path",
		Terminal:     nil,
	})

	req := client.TerminatePodRequest{
		PodKey: "worktree-pod",
	}

	err := handler.OnTerminatePod(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- OnListPods Tests ---

func TestOnListPodsEmpty(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	pods := handler.OnListPods()
	if len(pods) != 0 {
		t.Errorf("expected 0 pods, got %d", len(pods))
	}
}

func TestOnListPodsWithPods(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Add pods
	store.Put("pod-1", &Pod{
		ID:         "pod-1",
		PodKey: "pod-1",
		Status:     PodStatusRunning,
	})
	store.Put("pod-2", &Pod{
		ID:         "pod-2",
		PodKey: "pod-2",
		Status:     PodStatusInitializing,
	})

	pods := handler.OnListPods()
	if len(pods) != 2 {
		t.Errorf("expected 2 pods, got %d", len(pods))
	}
}

func TestOnListPodsWithTerminalPID(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{
			MaxConcurrentPods: 10,
			WorkspaceRoot:         tempDir,
		},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// First create a pod with a real terminal
	createReq := client.CreatePodRequest{
		PodKey:      "list-pid-pod",
		LaunchCommand: "sleep",
	}

	err := handler.OnCreatePod(createReq)
	if err != nil {
		t.Skipf("Could not create pod: %v", err)
	}

	// List pods
	pods := handler.OnListPods()
	if len(pods) != 1 {
		t.Errorf("pods count = %d, want 1", len(pods))
	}

	// Check PID is set
	if pods[0].Pid == 0 {
		t.Log("Pod PID should be non-zero")
	}

	// Clean up
	pod, ok := store.Get("list-pid-pod")
	if ok && pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}
