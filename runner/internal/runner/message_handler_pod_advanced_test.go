package runner

import (
	"errors"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
)

// Tests for advanced OnCreatePod scenarios

func TestOnCreatePodWithLaunchArgs(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{
			MaxConcurrentPods: 10,
			WorkspaceRoot:     tempDir,
		},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "launch-args-pod",
		LaunchCommand: "echo",
		LaunchArgs:    []string{"hello", "world"},
		AgentfileSource: "AGENT echo\nPROMPT_POSITION prepend\n",
	}

	err := handler.OnCreatePod(cmd)
	if err != nil {
		t.Logf("OnCreatePod with launch args: %v", err)
	}

	// Clean up
	pod, ok := store.Get("launch-args-pod")
	if ok {
		if comps := testPTYComponents(pod); comps != nil && comps.Terminal != nil {
			comps.Terminal.Stop()
		}
	}
}

func TestOnCreatePodWithPromptInArgs(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{
			MaxConcurrentPods: 10,
			WorkspaceRoot:     tempDir,
		},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Prompt is now passed as first argument (handled by Backend)
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "prompt-pod",
		LaunchCommand: "echo",
		LaunchArgs:    []string{"Hello from test"},
		AgentfileSource: "AGENT echo\nPROMPT_POSITION prepend\n",
	}

	err := handler.OnCreatePod(cmd)
	if err != nil {
		t.Logf("OnCreatePod with prompt in args: %v", err)
	}

	// Clean up
	pod, ok := store.Get("prompt-pod")
	if ok {
		if comps := testPTYComponents(pod); comps != nil && comps.Terminal != nil {
			comps.Terminal.Stop()
		}
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

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "worktree-error-pod",
		LaunchCommand: "echo",
		AgentfileSource: "AGENT echo\nPROMPT_POSITION prepend\n",
		SandboxConfig: &runnerv1.SandboxConfig{
			RepositoryUrl:  "https://github.com/test/repo",
			SourceBranch:   "main",
			CredentialType: "runner_local",
		},
	}

	err := handler.OnCreatePod(cmd)
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
			WorkspaceRoot:     tempDir,
		},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "send-error-pod",
		LaunchCommand: "echo",
		AgentfileSource: "AGENT echo\nPROMPT_POSITION prepend\n",
	}

	err := handler.OnCreatePod(cmd)
	// Pod should still be created even if send fails
	if err != nil {
		t.Logf("OnCreatePod with send error: %v", err)
	}

	// Clean up
	pod, ok := store.Get("send-error-pod")
	if ok {
		if comps := testPTYComponents(pod); comps != nil && comps.Terminal != nil {
			comps.Terminal.Stop()
		}
	}
}
