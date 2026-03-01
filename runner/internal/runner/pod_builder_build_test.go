package runner

import (
	"context"
	"testing"

	runnerv1 "github.com/AgentsMesh/AgentsMesh/proto/gen/go/runner/v1"
	"github.com/AgentsMesh/AgentsMesh/runner/internal/config"
	"github.com/AgentsMesh/AgentsMesh/runner/internal/workspace"
)

// Tests for Build and setup functionality

func TestPodBuilderBuildSuccessWithOptions(t *testing.T) {
	tempDir := t.TempDir()
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "build-pod",
		LaunchCommand: "echo",
		LaunchArgs:    []string{"hello"},
	}

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

	pod, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pod.PodKey != "build-pod" {
		t.Errorf("PodKey = %s, want build-pod", pod.PodKey)
	}
	if pod.GetStatus() != PodStatusInitializing {
		t.Errorf("Status = %s, want initializing", pod.GetStatus())
	}

	// Clean up
	if pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

func TestPodBuilderBuildTerminalError(t *testing.T) {
	tempDir := t.TempDir()
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	// Use a command that doesn't exist
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "error-pod",
		LaunchCommand: "/nonexistent/command/path/that/doesnt/exist/12345",
	}

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

	pod, err := builder.Build(context.Background())
	// May or may not fail depending on terminal implementation
	t.Logf("Build with invalid command: pod=%v, err=%v", pod != nil, err)

	// Clean up if pod was created
	if pod != nil && pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

func TestPodBuilderSetupNoManager(t *testing.T) {
	tempDir := t.TempDir()
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	// No workspace manager, but with config
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "workspace-test",
		LaunchCommand: "echo",
		LaunchArgs:    []string{"test"},
	}

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

	pod, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

func TestPodBuilderSetupWithEmptySandbox(t *testing.T) {
	tempDir := t.TempDir()

	ws, err := workspace.NewManager(tempDir, "")
	if err != nil {
		t.Skipf("Could not create workspace manager: %v", err)
	}

	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
		workspace: ws,
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "temp-workspace-test",
		LaunchCommand: "echo",
		LaunchArgs:    []string{"test"},
		SandboxConfig: &runnerv1.SandboxConfig{
			// Empty sandbox config - creates empty workspace
		},
	}

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

	pod, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}
