//go:build integration

package runner

import (
	"context"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/workspace"
)

// --- Test Build method ---

func TestPodBuilderBuildSuccess(t *testing.T) {
	tempDir := t.TempDir()
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
			AgentEnvVars:  map[string]string{"CONFIG_VAR": "value"},
		},
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "pod-build-test",
		LaunchCommand: "echo",
		LaunchArgs:    []string{"hello"},
	}

	pod, err := NewPodBuilderFromRunner(runner).
		WithCommand(cmd).
		WithTerminalSize(100, 30). // (cols, rows)
		Build(context.Background())

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if pod == nil {
		t.Fatal("pod should not be nil")
	}

	if pod.PodKey != "pod-build-test" {
		t.Errorf("PodKey = %v, want pod-build-test", pod.PodKey)
	}
	if pod.GetStatus() != PodStatusInitializing {
		t.Errorf("Status = %v, want initializing", pod.GetStatus())
	}

	// Clean up terminal if created
	if pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

func TestPodBuilderBuildWithMinimalConfig(t *testing.T) {
	tempDir := t.TempDir()
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "minimal-pod",
		LaunchCommand: "echo",
	}

	pod, err := NewPodBuilderFromRunner(runner).
		WithCommand(cmd).
		Build(context.Background())

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if pod.PodKey != "minimal-pod" {
		t.Errorf("PodKey = %v, want minimal-pod", pod.PodKey)
	}

	// Clean up
	if pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

// --- Test setup with SandboxConfig ---

func TestPodBuilderSetupWithWorkspaceManager(t *testing.T) {
	// Create a temporary workspace manager
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
		PodKey:        "test-sandbox",
		LaunchCommand: "echo",
		LaunchArgs:    []string{"test"},
		SandboxConfig: &runnerv1.SandboxConfig{
			// Empty sandbox config - creates empty workspace
		},
	}

	pod, err := NewPodBuilderFromRunner(runner).
		WithCommand(cmd).
		Build(context.Background())

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

func TestPodBuilderSetupLocalPath(t *testing.T) {
	tempDir := t.TempDir()
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
		workspace: nil, // No workspace manager
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-local",
		LaunchCommand: "echo",
		LaunchArgs:    []string{"test"},
		SandboxConfig: &runnerv1.SandboxConfig{
			LocalPath: tempDir,
		},
	}

	pod, err := NewPodBuilderFromRunner(runner).
		WithCommand(cmd).
		Build(context.Background())

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

func TestPodBuilderSetupLocalPathNotExist(t *testing.T) {
	tempDir := t.TempDir()
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-local-notexist",
		LaunchCommand: "echo",
		LaunchArgs:    []string{"test"},
		SandboxConfig: &runnerv1.SandboxConfig{
			LocalPath: "/nonexistent/path/that/does/not/exist",
		},
	}

	_, err := NewPodBuilderFromRunner(runner).
		WithCommand(cmd).
		Build(context.Background())

	if err == nil {
		t.Error("expected error for non-existent local path")
	}
}

func TestPodBuilderSetupWorktreeNoManager(t *testing.T) {
	tempDir := t.TempDir()
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
		workspace: nil, // No workspace manager
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-worktree-nomanager",
		LaunchCommand: "echo",
		LaunchArgs:    []string{"test"},
		SandboxConfig: &runnerv1.SandboxConfig{
			RepositoryUrl:  "https://github.com/test/repo",
			SourceBranch:   "main",
			CredentialType: "runner_local",
		},
	}

	_, err := NewPodBuilderFromRunner(runner).
		WithCommand(cmd).
		Build(context.Background())

	if err == nil {
		t.Error("expected error for worktree without workspace manager")
	}
}

// --- Test mergeEnvVars edge cases ---

func TestPodBuilderMergeEnvVarsEmptyBoth(t *testing.T) {
	runner := &Runner{
		cfg: &config.Config{
			AgentEnvVars: nil,
		},
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-pod",
		LaunchCommand: "echo",
		// No EnvVars
	}

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

	result := builder.mergeEnvVars("")

	if len(result) != 0 {
		t.Errorf("result length = %d, want 0", len(result))
	}
}

func TestPodBuilderMergeEnvVarsOnlyConfig(t *testing.T) {
	runner := &Runner{
		cfg: &config.Config{
			AgentEnvVars: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
		},
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-pod",
		LaunchCommand: "echo",
		// No EnvVars
	}

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

	result := builder.mergeEnvVars("")

	if result["VAR1"] != "value1" {
		t.Errorf("VAR1 = %v, want value1", result["VAR1"])
	}
	if result["VAR2"] != "value2" {
		t.Errorf("VAR2 = %v, want value2", result["VAR2"])
	}
}

func TestPodBuilderMergeEnvVarsOnlyCommand(t *testing.T) {
	runner := &Runner{
		cfg: &config.Config{
			AgentEnvVars: nil,
		},
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-pod",
		LaunchCommand: "echo",
		EnvVars: map[string]string{
			"CMD_VAR": "cmd_value",
		},
	}

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

	result := builder.mergeEnvVars("")

	if result["CMD_VAR"] != "cmd_value" {
		t.Errorf("CMD_VAR = %v, want cmd_value", result["CMD_VAR"])
	}
}

// --- Test Pod status ---

func TestPodStatusConstants(t *testing.T) {
	// Verify pod status constants exist
	if PodStatusInitializing != "initializing" {
		t.Errorf("PodStatusInitializing = %v, want initializing", PodStatusInitializing)
	}
	if PodStatusRunning != "running" {
		t.Errorf("PodStatusRunning = %v, want running", PodStatusRunning)
	}
}

// --- Benchmark ---

func BenchmarkPodBuilderBuild(b *testing.B) {
	tempDir := b.TempDir()
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := &runnerv1.CreatePodCommand{
			PodKey:        "benchmark-pod",
			LaunchCommand: "echo",
			LaunchArgs:    []string{"test"},
		}

		pod, _ := NewPodBuilderFromRunner(runner).
			WithCommand(cmd).
			Build(ctx)

		if pod != nil && pod.Terminal != nil {
			pod.Terminal.Stop()
		}
	}
}
