package runner

import (
	"context"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/workspace"
)

// Tests for Build and setup functionality

const testMinimalAgentFile = "AGENT echo\nPROMPT_POSITION prepend\n"

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
		AgentfileSource: testMinimalAgentFile,
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

	if pod.IO != nil {
		pod.IO.Stop()
	}
}

func TestPodBuilderBuildTerminalError(t *testing.T) {
	tempDir := t.TempDir()
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "error-pod",
		LaunchCommand: "/nonexistent/command/path/that/doesnt/exist/12345",
		AgentfileSource: "AGENT /nonexistent/command/path/that/doesnt/exist/12345\n",
	}

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

	pod, err := builder.Build(context.Background())
	t.Logf("Build with invalid command: pod=%v, err=%v", pod != nil, err)

	if pod != nil && pod.IO != nil {
		pod.IO.Stop()
	}
}

func TestPodBuilderSetupNoManager(t *testing.T) {
	tempDir := t.TempDir()
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "workspace-test",
		LaunchCommand: "echo",
		LaunchArgs:    []string{"test"},
		AgentfileSource: testMinimalAgentFile,
	}

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

	pod, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pod.IO != nil {
		pod.IO.Stop()
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
		AgentfileSource: testMinimalAgentFile,
		SandboxConfig: &runnerv1.SandboxConfig{},
	}

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

	pod, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pod.IO != nil {
		pod.IO.Stop()
	}
}
