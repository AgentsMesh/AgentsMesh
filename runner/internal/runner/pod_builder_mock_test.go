package runner

import (
	"context"
	"testing"

	"github.com/anthropics/agentmesh/runner/internal/client"
	"github.com/anthropics/agentmesh/runner/internal/config"
)

// --- Tests for PodBuilder ---

func TestNewPodBuilder(t *testing.T) {
	runner := &Runner{
		cfg: &config.Config{},
	}

	builder := NewPodBuilder(runner)

	if builder == nil {
		t.Fatal("NewPodBuilder returned nil")
	}

	if builder.runner != runner {
		t.Error("runner should be set")
	}

	if builder.envVars == nil {
		t.Error("envVars should be initialized")
	}

	if builder.rows != 24 {
		t.Errorf("rows default = %d, want 24", builder.rows)
	}

	if builder.cols != 80 {
		t.Errorf("cols default = %d, want 80", builder.cols)
	}
}

func TestPodBuilderWithPodKey(t *testing.T) {
	runner := &Runner{cfg: &config.Config{}}
	builder := NewPodBuilder(runner).WithPodKey("test-key")

	if builder.podKey != "test-key" {
		t.Errorf("podKey = %v, want test-key", builder.podKey)
	}
}

func TestPodBuilderWithAgentType(t *testing.T) {
	runner := &Runner{cfg: &config.Config{}}
	builder := NewPodBuilder(runner).WithAgentType("claude-code")

	if builder.agentType != "claude-code" {
		t.Errorf("agentType = %v, want claude-code", builder.agentType)
	}
}

func TestPodBuilderWithLaunchCommand(t *testing.T) {
	runner := &Runner{cfg: &config.Config{}}
	builder := NewPodBuilder(runner).WithLaunchCommand("claude", []string{"--headless"})

	if builder.launchCommand != "claude" {
		t.Errorf("launchCommand = %v, want claude", builder.launchCommand)
	}

	if len(builder.launchArgs) != 1 || builder.launchArgs[0] != "--headless" {
		t.Errorf("launchArgs = %v, want [--headless]", builder.launchArgs)
	}
}

func TestPodBuilderWithEnvVars(t *testing.T) {
	runner := &Runner{cfg: &config.Config{}}
	builder := NewPodBuilder(runner).WithEnvVars(map[string]string{
		"VAR1": "value1",
		"VAR2": "value2",
	})

	if builder.envVars["VAR1"] != "value1" {
		t.Errorf("VAR1 = %v, want value1", builder.envVars["VAR1"])
	}

	if builder.envVars["VAR2"] != "value2" {
		t.Errorf("VAR2 = %v, want value2", builder.envVars["VAR2"])
	}
}

func TestPodBuilderWithEnvVar(t *testing.T) {
	runner := &Runner{cfg: &config.Config{}}
	builder := NewPodBuilder(runner).
		WithEnvVar("KEY1", "VALUE1").
		WithEnvVar("KEY2", "VALUE2")

	if builder.envVars["KEY1"] != "VALUE1" {
		t.Errorf("KEY1 = %v, want VALUE1", builder.envVars["KEY1"])
	}

	if builder.envVars["KEY2"] != "VALUE2" {
		t.Errorf("KEY2 = %v, want VALUE2", builder.envVars["KEY2"])
	}
}

func TestPodBuilderWithTerminalSize(t *testing.T) {
	runner := &Runner{cfg: &config.Config{}}
	builder := NewPodBuilder(runner).WithTerminalSize(48, 160)

	if builder.rows != 48 {
		t.Errorf("rows = %d, want 48", builder.rows)
	}

	if builder.cols != 160 {
		t.Errorf("cols = %d, want 160", builder.cols)
	}
}

func TestPodBuilderWithTerminalSizeZeroValues(t *testing.T) {
	runner := &Runner{cfg: &config.Config{}}
	builder := NewPodBuilder(runner).WithTerminalSize(0, 0)

	// Should keep defaults
	if builder.rows != 24 {
		t.Errorf("rows = %d, want 24 (default)", builder.rows)
	}

	if builder.cols != 80 {
		t.Errorf("cols = %d, want 80 (default)", builder.cols)
	}
}

func TestPodBuilderWithInitialPrompt(t *testing.T) {
	runner := &Runner{cfg: &config.Config{}}
	builder := NewPodBuilder(runner).WithInitialPrompt("Hello, Claude!")

	if builder.initialPrompt != "Hello, Claude!" {
		t.Errorf("initialPrompt = %v, want Hello, Claude!", builder.initialPrompt)
	}
}

func TestPodBuilderWithWorkDirConfig(t *testing.T) {
	runner := &Runner{cfg: &config.Config{}}
	builder := NewPodBuilder(runner).WithWorkDirConfig(&client.WorkDirConfig{
		Type:          "worktree",
		RepositoryURL: "https://github.com/test/repo.git",
		Branch:        "feature/test",
	})

	if builder.workDirConfig == nil {
		t.Error("workDirConfig should not be nil")
	}
	if builder.workDirConfig.RepositoryURL != "https://github.com/test/repo.git" {
		t.Errorf("repositoryURL = %v, want https://github.com/test/repo.git", builder.workDirConfig.RepositoryURL)
	}
	if builder.workDirConfig.Branch != "feature/test" {
		t.Errorf("branch = %v, want feature/test", builder.workDirConfig.Branch)
	}
}

func TestPodBuilderWithFilesToCreateMultiple(t *testing.T) {
	runner := &Runner{cfg: &config.Config{}}
	builder := NewPodBuilder(runner).WithFilesToCreate([]client.FileToCreate{
		{PathTemplate: "{{.sandbox.root_path}}/config.json", Content: "{}", Mode: 0644},
		{PathTemplate: "{{.sandbox.work_dir}}/data.txt", Content: "data"},
	})

	if len(builder.filesToCreate) != 2 {
		t.Errorf("filesToCreate length = %d, want 2", len(builder.filesToCreate))
	}
	if builder.filesToCreate[0].PathTemplate != "{{.sandbox.root_path}}/config.json" {
		t.Errorf("filesToCreate[0].PathTemplate = %v, want {{.sandbox.root_path}}/config.json", builder.filesToCreate[0].PathTemplate)
	}
}

func TestPodBuilderChaining(t *testing.T) {
	runner := &Runner{cfg: &config.Config{}}
	builder := NewPodBuilder(runner).
		WithPodKey("pod-1").
		WithAgentType("claude-code").
		WithLaunchCommand("claude", []string{"--headless"}).
		WithEnvVar("API_KEY", "secret").
		WithTerminalSize(48, 160).
		WithInitialPrompt("Hello!").
		WithWorkDirConfig(&client.WorkDirConfig{
			Type:          "worktree",
			RepositoryURL: "https://github.com/test/repo.git",
			Branch:        "main",
		}).
		WithFilesToCreate([]client.FileToCreate{
			{PathTemplate: "{{.sandbox.root_path}}/test.txt", Content: "test"},
		})

	if builder.podKey != "pod-1" {
		t.Errorf("podKey = %v, want pod-1", builder.podKey)
	}

	if builder.agentType != "claude-code" {
		t.Errorf("agentType = %v, want claude-code", builder.agentType)
	}

	if builder.launchCommand != "claude" {
		t.Errorf("launchCommand = %v, want claude", builder.launchCommand)
	}

	if builder.rows != 48 {
		t.Errorf("rows = %d, want 48", builder.rows)
	}

	if builder.initialPrompt != "Hello!" {
		t.Errorf("initialPrompt = %v, want Hello!", builder.initialPrompt)
	}

	if builder.workDirConfig == nil {
		t.Error("workDirConfig should not be nil")
	}

	if len(builder.filesToCreate) != 1 {
		t.Error("filesToCreate not set correctly")
	}
}

func TestPodBuilderBuildEmptyPodKey(t *testing.T) {
	runner := &Runner{cfg: &config.Config{}}
	builder := NewPodBuilder(runner)

	ctx := context.Background()
	_, err := builder.Build(ctx)

	if err == nil {
		t.Error("expected error for empty pod key")
	}

	if !contains(err.Error(), "pod key is required") {
		t.Errorf("error = %v, want containing 'pod key is required'", err)
	}
}

func TestPodBuilderBuildEmptyLaunchCommand(t *testing.T) {
	runner := &Runner{cfg: &config.Config{}}
	builder := NewPodBuilder(runner).WithPodKey("test-pod")

	ctx := context.Background()
	_, err := builder.Build(ctx)

	if err == nil {
		t.Error("expected error for empty launch command")
	}

	if !contains(err.Error(), "launch command is required") {
		t.Errorf("error = %v, want containing 'launch command is required'", err)
	}
}

// Note: Additional tests are in pod_builder_test.go and pod_builder_extended_test.go
