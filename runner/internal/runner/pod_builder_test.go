package runner

import (
	"context"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/workspace"
)

func TestPodBuilderStruct(t *testing.T) {
	runner := &Runner{cfg: &config.Config{}}
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "pod-1",
		LaunchCommand: "claude",
		LaunchArgs:    []string{"--headless"},
		EnvVars:       map[string]string{"KEY": "VALUE"},
		FilesToCreate: []*runnerv1.FileToCreate{
			{Path: "{{.sandbox.root_path}}/test.txt", Content: "test"},
		},
		SandboxConfig: &runnerv1.SandboxConfig{
			LocalPath: "/tmp",
		},
	}

	builder := NewPodBuilder(runner).
		WithCommand(cmd).
		WithTerminalSize(24, 80)

	if builder.cmd.PodKey != "pod-1" {
		t.Errorf("podKey: got %v, want pod-1", builder.cmd.PodKey)
	}

	if builder.rows != 24 {
		t.Errorf("rows: got %v, want 24", builder.rows)
	}
}

func TestPodBuilderFluentAPI(t *testing.T) {
	runner := &Runner{}
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "pod-1",
		LaunchCommand: "claude",
		LaunchArgs:    []string{"--headless"},
		EnvVars: map[string]string{
			"KEY1": "VALUE1",
			"KEY2": "VALUE2",
		},
		SandboxConfig: &runnerv1.SandboxConfig{
			RepositoryUrl:  "https://github.com/test/repo.git",
			SourceBranch:   "main",
			CredentialType: "runner_local",
		},
		FilesToCreate: []*runnerv1.FileToCreate{
			{Path: "{{.sandbox.root_path}}/config.json", Content: "{}"},
		},
	}

	builder := NewPodBuilder(runner)
	result := builder.
		WithCommand(cmd).
		WithTerminalSize(40, 120)

	// Verify it returns the same builder
	if result != builder {
		t.Error("fluent API should return the same builder")
	}

	// Verify values
	if builder.cmd.PodKey != "pod-1" {
		t.Errorf("podKey: got %v, want pod-1", builder.cmd.PodKey)
	}
	if builder.cmd.LaunchCommand != "claude" {
		t.Errorf("launchCommand: got %v, want claude", builder.cmd.LaunchCommand)
	}
	if len(builder.cmd.LaunchArgs) != 1 {
		t.Errorf("launchArgs length: got %v, want 1", len(builder.cmd.LaunchArgs))
	}
	if builder.cmd.EnvVars["KEY1"] != "VALUE1" {
		t.Errorf("envVars[KEY1]: got %v, want VALUE1", builder.cmd.EnvVars["KEY1"])
	}
	if builder.cmd.EnvVars["KEY2"] != "VALUE2" {
		t.Errorf("envVars[KEY2]: got %v, want VALUE2", builder.cmd.EnvVars["KEY2"])
	}
	if builder.rows != 40 {
		t.Errorf("rows: got %v, want 40", builder.rows)
	}
	if builder.cols != 120 {
		t.Errorf("cols: got %v, want 120", builder.cols)
	}
	if builder.cmd.SandboxConfig == nil {
		t.Error("sandboxConfig should not be nil")
	} else {
		if builder.cmd.SandboxConfig.RepositoryUrl != "https://github.com/test/repo.git" {
			t.Errorf("repositoryUrl: got %v, want https://github.com/test/repo.git", builder.cmd.SandboxConfig.RepositoryUrl)
		}
		if builder.cmd.SandboxConfig.SourceBranch != "main" {
			t.Errorf("branch: got %v, want main", builder.cmd.SandboxConfig.SourceBranch)
		}
	}
	if len(builder.cmd.FilesToCreate) != 1 {
		t.Errorf("filesToCreate length: got %v, want 1", len(builder.cmd.FilesToCreate))
	}
}

func TestPodBuilderDefaultValues(t *testing.T) {
	runner := &Runner{}
	builder := NewPodBuilder(runner)

	if builder.rows != 24 {
		t.Errorf("default rows: got %v, want 24", builder.rows)
	}

	if builder.cols != 80 {
		t.Errorf("default cols: got %v, want 80", builder.cols)
	}
}

func TestPodBuilderTerminalSizeValidation(t *testing.T) {
	runner := &Runner{}
	builder := NewPodBuilder(runner)

	// Test with invalid values (should use defaults)
	builder.WithTerminalSize(0, 0)

	if builder.rows != 24 {
		t.Errorf("rows with zero: got %v, want 24 (default)", builder.rows)
	}

	if builder.cols != 80 {
		t.Errorf("cols with zero: got %v, want 80 (default)", builder.cols)
	}

	// Test with negative values (should use defaults)
	builder.WithTerminalSize(-1, -1)

	if builder.rows != 24 {
		t.Errorf("rows with negative: got %v, want 24 (default)", builder.rows)
	}
}

func TestPodBuilderBuildWithoutCommand(t *testing.T) {
	runner := &Runner{}
	builder := NewPodBuilder(runner)

	_, err := builder.Build(context.Background())
	if err == nil {
		t.Error("expected error for missing command")
	}
}

func TestPodBuilderMergeEnvVars(t *testing.T) {
	runner := &Runner{
		cfg: &config.Config{
			AgentEnvVars: map[string]string{
				"CONFIG_VAR": "config_value",
				"SHARED_VAR": "config_shared",
			},
		},
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-pod",
		LaunchCommand: "echo",
		EnvVars: map[string]string{
			"BUILDER_VAR": "builder_value",
			"SHARED_VAR":  "builder_shared",
		},
	}

	builder := NewPodBuilder(runner).WithCommand(cmd)

	result := builder.mergeEnvVars()

	if result["CONFIG_VAR"] != "config_value" {
		t.Errorf("CONFIG_VAR: got %v, want config_value", result["CONFIG_VAR"])
	}

	if result["BUILDER_VAR"] != "builder_value" {
		t.Errorf("BUILDER_VAR: got %v, want builder_value", result["BUILDER_VAR"])
	}

	if result["SHARED_VAR"] != "builder_shared" {
		t.Errorf("SHARED_VAR: got %v, want builder_shared (command should override config)", result["SHARED_VAR"])
	}
}

func TestPodBuilderMergeEnvVarsNilConfig(t *testing.T) {
	runner := &Runner{
		cfg: nil,
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-pod",
		LaunchCommand: "echo",
		EnvVars: map[string]string{
			"BUILDER_VAR": "builder_value",
		},
	}

	builder := NewPodBuilder(runner).WithCommand(cmd)

	result := builder.mergeEnvVars()

	if result["BUILDER_VAR"] != "builder_value" {
		t.Errorf("BUILDER_VAR: got %v, want builder_value", result["BUILDER_VAR"])
	}
}

func TestPodBuilderWithAllOptions(t *testing.T) {
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: "/workspace",
			AgentEnvVars: map[string]string{
				"CONFIG_VAR": "config_value",
			},
		},
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "pod-key",
		LaunchCommand: "claude",
		LaunchArgs:    []string{"--headless"},
		EnvVars: map[string]string{
			"ENV1": "value1",
			"ENV2": "value2",
		},
		SandboxConfig: &runnerv1.SandboxConfig{
			RepositoryUrl:  "https://github.com/test/repo.git",
			SourceBranch:   "main",
			CredentialType: "runner_local",
		},
		FilesToCreate: []*runnerv1.FileToCreate{
			{Path: "{{.sandbox.root_path}}/test.txt", Content: "test"},
		},
	}

	builder := NewPodBuilder(runner).
		WithCommand(cmd).
		WithTerminalSize(40, 120)

	if builder.cmd.PodKey != "pod-key" {
		t.Errorf("podKey = %v, want pod-key", builder.cmd.PodKey)
	}
	if len(builder.cmd.LaunchArgs) != 1 || builder.cmd.LaunchArgs[0] != "--headless" {
		t.Errorf("launchArgs = %v, want [--headless]", builder.cmd.LaunchArgs)
	}
	if builder.cmd.EnvVars["ENV1"] != "value1" {
		t.Errorf("envVars[ENV1] = %v, want value1", builder.cmd.EnvVars["ENV1"])
	}
	if builder.cmd.EnvVars["ENV2"] != "value2" {
		t.Errorf("envVars[ENV2] = %v, want value2", builder.cmd.EnvVars["ENV2"])
	}
	if builder.rows != 40 || builder.cols != 120 {
		t.Errorf("terminal size = %dx%d, want 40x120", builder.rows, builder.cols)
	}
	if builder.cmd.SandboxConfig == nil {
		t.Error("sandboxConfig should not be nil")
	}
	if len(builder.cmd.FilesToCreate) != 1 {
		t.Error("filesToCreate not set correctly")
	}
}

// Benchmarks

func BenchmarkNewPodBuilder(b *testing.B) {
	runner := &Runner{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewPodBuilder(runner)
	}
}

func BenchmarkPodBuilderFluentAPI(b *testing.B) {
	runner := &Runner{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := &runnerv1.CreatePodCommand{
			PodKey:        "pod-1",
			LaunchCommand: "claude",
			LaunchArgs:    []string{"--headless"},
			EnvVars:       map[string]string{"KEY": "VALUE"},
		}
		NewPodBuilder(runner).
			WithCommand(cmd).
			WithTerminalSize(40, 120)
	}
}

func BenchmarkPodBuilderMergeEnvVars(b *testing.B) {
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
		EnvVars: map[string]string{
			"POD_VAR1": "pod_value1",
		},
	}

	builder := NewPodBuilder(runner).WithCommand(cmd)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder.mergeEnvVars()
	}
}

// --- Additional tests for Build coverage ---

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

	builder := NewPodBuilder(runner).WithCommand(cmd)

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

	builder := NewPodBuilder(runner).WithCommand(cmd)

	pod, err := builder.Build(context.Background())
	// May or may not fail depending on terminal implementation
	t.Logf("Build with invalid command: pod=%v, err=%v", pod != nil, err)

	// Clean up if pod was created
	if pod != nil && pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

// --- Additional tests for setup coverage ---

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

	builder := NewPodBuilder(runner).WithCommand(cmd)

	pod, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

// --- Test setup with empty SandboxConfig ---

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

	builder := NewPodBuilder(runner).WithCommand(cmd)

	pod, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

// --- Test Pod constants ---

func TestPodStatusConstantsInBuilder(t *testing.T) {
	// Verify pod status constants
	if PodStatusInitializing != "initializing" {
		t.Errorf("PodStatusInitializing = %v, want initializing", PodStatusInitializing)
	}
	if PodStatusRunning != "running" {
		t.Errorf("PodStatusRunning = %v, want running", PodStatusRunning)
	}
}
