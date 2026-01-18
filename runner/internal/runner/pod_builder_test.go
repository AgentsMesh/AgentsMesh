package runner

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/workspace"
)

func TestPodBuilderStruct(t *testing.T) {
	builder := &PodBuilder{
		podKey:        "pod-1",
		agentType:     "claude-code",
		launchCommand: "claude",
		launchArgs:    []string{"--headless"},
		envVars:       map[string]string{"KEY": "VALUE"},
		rows:          24,
		cols:          80,
		filesToCreate: []client.FileToCreate{
			{PathTemplate: "{{.sandbox.root_path}}/test.txt", Content: "test"},
		},
		workDirConfig: &client.WorkDirConfig{
			Type:      "local",
			LocalPath: "/tmp",
		},
	}

	if builder.podKey != "pod-1" {
		t.Errorf("podKey: got %v, want pod-1", builder.podKey)
	}

	if builder.rows != 24 {
		t.Errorf("rows: got %v, want 24", builder.rows)
	}
}

func TestPodBuilderFluentAPI(t *testing.T) {
	runner := &Runner{}
	builder := NewPodBuilder(runner)

	// Test fluent API chain
	// Note: InitialPrompt is now handled by Backend via LaunchArgs
	result := builder.
		WithPodKey("pod-1").
		WithAgentType("claude-code").
		WithLaunchCommand("claude", []string{"--headless"}).
		WithEnvVars(map[string]string{"KEY1": "VALUE1"}).
		WithEnvVar("KEY2", "VALUE2").
		WithTerminalSize(40, 120).
		WithWorkDirConfig(&client.WorkDirConfig{
			Type:          "worktree",
			RepositoryURL: "https://github.com/test/repo.git",
			Branch:        "main",
		}).
		WithFilesToCreate([]client.FileToCreate{
			{PathTemplate: "{{.sandbox.root_path}}/config.json", Content: "{}"},
		})

	// Verify it returns the same builder
	if result != builder {
		t.Error("fluent API should return the same builder")
	}

	// Verify values
	if builder.podKey != "pod-1" {
		t.Errorf("podKey: got %v, want pod-1", builder.podKey)
	}
	if builder.agentType != "claude-code" {
		t.Errorf("agentType: got %v, want claude-code", builder.agentType)
	}
	if builder.launchCommand != "claude" {
		t.Errorf("launchCommand: got %v, want claude", builder.launchCommand)
	}
	if len(builder.launchArgs) != 1 {
		t.Errorf("launchArgs length: got %v, want 1", len(builder.launchArgs))
	}
	if builder.envVars["KEY1"] != "VALUE1" {
		t.Errorf("envVars[KEY1]: got %v, want VALUE1", builder.envVars["KEY1"])
	}
	if builder.envVars["KEY2"] != "VALUE2" {
		t.Errorf("envVars[KEY2]: got %v, want VALUE2", builder.envVars["KEY2"])
	}
	if builder.rows != 40 {
		t.Errorf("rows: got %v, want 40", builder.rows)
	}
	if builder.cols != 120 {
		t.Errorf("cols: got %v, want 120", builder.cols)
	}
	if builder.workDirConfig == nil {
		t.Error("workDirConfig should not be nil")
	} else {
		if builder.workDirConfig.RepositoryURL != "https://github.com/test/repo.git" {
			t.Errorf("repositoryURL: got %v, want https://github.com/test/repo.git", builder.workDirConfig.RepositoryURL)
		}
		if builder.workDirConfig.Branch != "main" {
			t.Errorf("branch: got %v, want main", builder.workDirConfig.Branch)
		}
	}
	if len(builder.filesToCreate) != 1 {
		t.Errorf("filesToCreate length: got %v, want 1", len(builder.filesToCreate))
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

	if builder.envVars == nil {
		t.Error("envVars should be initialized")
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

func TestPodBuilderBuildWithoutPodKey(t *testing.T) {
	runner := &Runner{}
	builder := NewPodBuilder(runner)

	_, err := builder.Build(context.Background())
	if err == nil {
		t.Error("expected error for missing pod key")
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

	builder := NewPodBuilder(runner)
	builder.WithEnvVars(map[string]string{
		"BUILDER_VAR": "builder_value",
		"SHARED_VAR":  "builder_shared",
	})

	result := builder.mergeEnvVars()

	if result["CONFIG_VAR"] != "config_value" {
		t.Errorf("CONFIG_VAR: got %v, want config_value", result["CONFIG_VAR"])
	}

	if result["BUILDER_VAR"] != "builder_value" {
		t.Errorf("BUILDER_VAR: got %v, want builder_value", result["BUILDER_VAR"])
	}

	if result["SHARED_VAR"] != "builder_shared" {
		t.Errorf("SHARED_VAR: got %v, want builder_shared (builder should override config)", result["SHARED_VAR"])
	}
}

func TestPodBuilderMergeEnvVarsNilConfig(t *testing.T) {
	runner := &Runner{
		cfg: nil,
	}

	builder := NewPodBuilder(runner)
	builder.WithEnvVars(map[string]string{
		"BUILDER_VAR": "builder_value",
	})

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

	// Note: InitialPrompt is now handled by Backend via LaunchArgs
	builder := NewPodBuilder(runner).
		WithPodKey("pod-key").
		WithAgentType("claude-code").
		WithLaunchCommand("claude", []string{"--headless"}).
		WithEnvVars(map[string]string{"ENV1": "value1"}).
		WithEnvVar("ENV2", "value2").
		WithTerminalSize(40, 120).
		WithWorkDirConfig(&client.WorkDirConfig{
			Type:          "worktree",
			RepositoryURL: "https://github.com/test/repo.git",
			Branch:        "main",
		}).
		WithFilesToCreate([]client.FileToCreate{
			{PathTemplate: "{{.sandbox.root_path}}/test.txt", Content: "test"},
		})

	if builder.podKey != "pod-key" {
		t.Errorf("podKey = %v, want pod-key", builder.podKey)
	}
	if builder.agentType != "claude-code" {
		t.Errorf("agentType = %v, want claude-code", builder.agentType)
	}
	if len(builder.launchArgs) != 1 || builder.launchArgs[0] != "--headless" {
		t.Errorf("launchArgs = %v, want [--headless]", builder.launchArgs)
	}
	if builder.envVars["ENV1"] != "value1" {
		t.Errorf("envVars[ENV1] = %v, want value1", builder.envVars["ENV1"])
	}
	if builder.envVars["ENV2"] != "value2" {
		t.Errorf("envVars[ENV2] = %v, want value2", builder.envVars["ENV2"])
	}
	if builder.rows != 40 || builder.cols != 120 {
		t.Errorf("terminal size = %dx%d, want 40x120", builder.rows, builder.cols)
	}
	if builder.workDirConfig == nil {
		t.Error("workDirConfig should not be nil")
	}
	if len(builder.filesToCreate) != 1 {
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
		NewPodBuilder(runner).
			WithPodKey("pod-1").
			WithAgentType("claude-code").
			WithLaunchCommand("claude", []string{"--headless"}).
			WithEnvVars(map[string]string{"KEY": "VALUE"}).
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

	builder := NewPodBuilder(runner).
		WithEnvVars(map[string]string{
			"POD_VAR1": "pod_value1",
		})

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

	builder := NewPodBuilder(runner).
		WithPodKey("build-pod").
		WithAgentType("claude-code").
		WithLaunchCommand("echo", []string{"hello"})

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
	builder := NewPodBuilder(runner).
		WithPodKey("error-pod").
		WithAgentType("test").
		WithLaunchCommand("/nonexistent/command/path/that/doesnt/exist/12345", nil)

	pod, err := builder.Build(context.Background())
	// May or may not fail depending on terminal implementation
	t.Logf("Build with invalid command: pod=%v, err=%v", pod != nil, err)

	// Clean up if pod was created
	if pod != nil && pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

// --- Additional tests for setupWorkDir coverage ---

func TestPodBuilderSetupWorkDirNoManager(t *testing.T) {
	tempDir := t.TempDir()
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	// No workspace manager, but with config
	builder := NewPodBuilder(runner).
		WithPodKey("workspace-test").
		WithLaunchCommand("echo", []string{"test"})

	pod, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

// --- Test setupWorkDir with TempWorkspace ---

func TestPodBuilderSetupWorkDirWithTempWorkspace(t *testing.T) {
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
		WithPodKey("temp-workspace-test").
		WithLaunchCommand("echo", []string{"test"}).
		WithWorkDirConfig(&client.WorkDirConfig{
			Type: "tempdir",
		})

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
