package runner

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/client"
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

	// Note: InitialPrompt is now handled by Backend via LaunchArgs
	builder := NewPodBuilder(runner).
		WithPodKey("pod-build-test").
		WithAgentType("claude-code").
		WithLaunchCommand("echo", []string{"hello"}).
		WithTerminalSize(30, 100)

	pod, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if pod == nil {
		t.Fatal("pod should not be nil")
	}

	if pod.PodKey != "pod-build-test" {
		t.Errorf("PodKey = %v, want pod-build-test", pod.PodKey)
	}
	if pod.AgentType != "claude-code" {
		t.Errorf("AgentType = %v, want claude-code", pod.AgentType)
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

	builder := NewPodBuilder(runner).
		WithPodKey("minimal-pod").
		WithLaunchCommand("echo", nil)

	pod, err := builder.Build(context.Background())
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

// --- Test setupWorkDir with WorkDirConfig ---

func TestPodBuilderSetupWorkDirWithWorkspaceManager(t *testing.T) {
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

	builder := NewPodBuilder(runner).
		WithPodKey("test-workdir").
		WithLaunchCommand("echo", []string{"test"}).
		WithWorkDirConfig(&client.WorkDirConfig{
			Type: "tempdir",
		})

	pod, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

func TestPodBuilderSetupWorkDirLocalPath(t *testing.T) {
	tempDir := t.TempDir()
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
		workspace: nil, // No workspace manager
	}

	builder := NewPodBuilder(runner).
		WithPodKey("test-local").
		WithLaunchCommand("echo", []string{"test"}).
		WithWorkDirConfig(&client.WorkDirConfig{
			Type:      "local",
			LocalPath: tempDir,
		})

	pod, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

func TestPodBuilderSetupWorkDirLocalPathNotExist(t *testing.T) {
	tempDir := t.TempDir()
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	builder := NewPodBuilder(runner).
		WithPodKey("test-local-notexist").
		WithLaunchCommand("echo", []string{"test"}).
		WithWorkDirConfig(&client.WorkDirConfig{
			Type:      "local",
			LocalPath: "/nonexistent/path/that/does/not/exist",
		})

	_, err := builder.Build(context.Background())
	if err == nil {
		t.Error("expected error for non-existent local path")
	}
}

func TestPodBuilderSetupWorkDirWorktreeNoManager(t *testing.T) {
	tempDir := t.TempDir()
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
		workspace: nil, // No workspace manager
	}

	builder := NewPodBuilder(runner).
		WithPodKey("test-worktree-nomanager").
		WithLaunchCommand("echo", []string{"test"}).
		WithWorkDirConfig(&client.WorkDirConfig{
			Type:          "worktree",
			RepositoryURL: "https://github.com/test/repo",
			Branch:        "main",
		})

	_, err := builder.Build(context.Background())
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

	builder := NewPodBuilder(runner)
	// Don't add any env vars

	result := builder.mergeEnvVars()

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

	builder := NewPodBuilder(runner)

	result := builder.mergeEnvVars()

	if result["VAR1"] != "value1" {
		t.Errorf("VAR1 = %v, want value1", result["VAR1"])
	}
	if result["VAR2"] != "value2" {
		t.Errorf("VAR2 = %v, want value2", result["VAR2"])
	}
}

func TestPodBuilderMergeEnvVarsOnlyBuilder(t *testing.T) {
	runner := &Runner{
		cfg: &config.Config{
			AgentEnvVars: nil,
		},
	}

	builder := NewPodBuilder(runner).
		WithEnvVar("BUILDER_VAR", "builder_value")

	result := builder.mergeEnvVars()

	if result["BUILDER_VAR"] != "builder_value" {
		t.Errorf("BUILDER_VAR = %v, want builder_value", result["BUILDER_VAR"])
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

// --- Test WithNewProtocol (no-op for compatibility) ---

func TestPodBuilderWithNewProtocolNoOp(t *testing.T) {
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: "/tmp",
		},
	}

	// WithNewProtocol should be a no-op but not cause errors
	builder := NewPodBuilder(runner).
		WithPodKey("test-newprotocol").
		WithNewProtocol(true).
		WithNewProtocol(false)

	if builder.podKey != "test-newprotocol" {
		t.Error("podKey should be set")
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
		builder := NewPodBuilder(runner).
			WithPodKey("benchmark-pod").
			WithAgentType("claude-code").
			WithLaunchCommand("echo", []string{"test"})

		pod, _ := builder.Build(ctx)
		if pod != nil && pod.Terminal != nil {
			pod.Terminal.Stop()
		}
	}
}
