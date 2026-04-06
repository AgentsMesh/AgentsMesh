package runner

import (
	"context"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/config"
)

// --- Tests for PodBuilder ---

func TestNewPodBuilder(t *testing.T) {
	runner := &Runner{
		cfg: &config.Config{},
	}

	builder := NewPodBuilderFromRunner(runner)

	if builder == nil {
		t.Fatal("NewPodBuilderFromRunner returned nil")
		return
	}

	if builder.deps.Config != runner.cfg {
		t.Error("deps.Config should be set")
	}

	if builder.rows != 24 {
		t.Errorf("rows default = %d, want 24", builder.rows)
	}

	if builder.cols != 80 {
		t.Errorf("cols default = %d, want 80", builder.cols)
	}
}

func TestPodBuilderWithCommand(t *testing.T) {
	runner := &Runner{cfg: &config.Config{}}
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-key",
		LaunchCommand: "echo",
		LaunchArgs:    []string{"hello"},
		AgentfileSource: "AGENT echo\nPROMPT_POSITION prepend\n",
		EnvVars: map[string]string{
			"VAR1": "value1",
		},
	}

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

	if builder.cmd == nil {
		t.Fatal("cmd should not be nil")
	}
	if builder.cmd.PodKey != "test-key" {
		t.Errorf("podKey = %v, want test-key", builder.cmd.PodKey)
	}
	if builder.cmd.LaunchCommand != "echo" {
		t.Errorf("launchCommand = %v, want echo", builder.cmd.LaunchCommand)
	}
	if len(builder.cmd.LaunchArgs) != 1 || builder.cmd.LaunchArgs[0] != "hello" {
		t.Errorf("launchArgs = %v, want [hello]", builder.cmd.LaunchArgs)
	}
	if builder.cmd.EnvVars["VAR1"] != "value1" {
		t.Errorf("envVars[VAR1] = %v, want value1", builder.cmd.EnvVars["VAR1"])
	}
}

func TestPodBuilderWithPtySize(t *testing.T) {
	runner := &Runner{cfg: &config.Config{}}
	builder := NewPodBuilderFromRunner(runner).WithPtySize(160, 48) // (cols, rows)

	if builder.cols != 160 {
		t.Errorf("cols = %d, want 160", builder.cols)
	}

	if builder.rows != 48 {
		t.Errorf("rows = %d, want 48", builder.rows)
	}
}

func TestPodBuilderWithPtySizeZeroValues(t *testing.T) {
	runner := &Runner{cfg: &config.Config{}}
	builder := NewPodBuilderFromRunner(runner).WithPtySize(0, 0)

	// Should keep defaults
	if builder.rows != 24 {
		t.Errorf("rows = %d, want 24 (default)", builder.rows)
	}

	if builder.cols != 80 {
		t.Errorf("cols = %d, want 80 (default)", builder.cols)
	}
}

func TestPodBuilderWithSandboxConfig(t *testing.T) {
	runner := &Runner{cfg: &config.Config{}}
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-pod",
		LaunchCommand: "echo",
		AgentfileSource: "AGENT echo\nPROMPT_POSITION prepend\n",
		SandboxConfig: &runnerv1.SandboxConfig{
			HttpCloneUrl:   "https://github.com/test/repo.git",
			SourceBranch:   "feature/test",
			CredentialType: "runner_local",
		},
	}

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

	if builder.cmd.SandboxConfig == nil {
		t.Error("sandboxConfig should not be nil")
	}
	if builder.cmd.SandboxConfig.HttpCloneUrl != "https://github.com/test/repo.git" {
		t.Errorf("httpCloneUrl = %v, want https://github.com/test/repo.git", builder.cmd.SandboxConfig.HttpCloneUrl)
	}
	if builder.cmd.SandboxConfig.SourceBranch != "feature/test" {
		t.Errorf("sourceBranch = %v, want feature/test", builder.cmd.SandboxConfig.SourceBranch)
	}
}

func TestPodBuilderWithFilesToCreateMultiple(t *testing.T) {
	runner := &Runner{cfg: &config.Config{}}
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-pod",
		LaunchCommand: "echo",
		AgentfileSource: "AGENT echo\nPROMPT_POSITION prepend\n",
		FilesToCreate: []*runnerv1.FileToCreate{
			{Path: "{{.sandbox.root_path}}/config.json", Content: "{}", Mode: 0644},
			{Path: "{{.sandbox.work_dir}}/data.txt", Content: "data"},
		},
	}

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

	if len(builder.cmd.FilesToCreate) != 2 {
		t.Errorf("filesToCreate length = %d, want 2", len(builder.cmd.FilesToCreate))
	}
	if builder.cmd.FilesToCreate[0].Path != "{{.sandbox.root_path}}/config.json" {
		t.Errorf("filesToCreate[0].Path = %v, want {{.sandbox.root_path}}/config.json", builder.cmd.FilesToCreate[0].Path)
	}
}

func TestPodBuilderCommandWithAllFields(t *testing.T) {
	runner := &Runner{cfg: &config.Config{}}
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "pod-1",
		LaunchCommand: "claude",
		LaunchArgs:    []string{"--headless"},
		AgentfileSource: "AGENT claude\nPROMPT_POSITION prepend\n",
		EnvVars: map[string]string{
			"API_KEY": "secret",
		},
		SandboxConfig: &runnerv1.SandboxConfig{
			HttpCloneUrl:   "https://github.com/test/repo.git",
			SourceBranch:   "main",
			CredentialType: "runner_local",
		},
		FilesToCreate: []*runnerv1.FileToCreate{
			{Path: "{{.sandbox.root_path}}/test.txt", Content: "test"},
		},
	}

	builder := NewPodBuilderFromRunner(runner).
		WithCommand(cmd).
		WithPtySize(160, 48) // (cols, rows)

	if builder.cmd.PodKey != "pod-1" {
		t.Errorf("podKey = %v, want pod-1", builder.cmd.PodKey)
	}

	if builder.cmd.LaunchCommand != "claude" {
		t.Errorf("launchCommand = %v, want claude", builder.cmd.LaunchCommand)
	}

	if len(builder.cmd.LaunchArgs) != 1 || builder.cmd.LaunchArgs[0] != "--headless" {
		t.Errorf("launchArgs = %v, want [--headless]", builder.cmd.LaunchArgs)
	}

	if builder.cols != 160 {
		t.Errorf("cols = %d, want 160", builder.cols)
	}

	if builder.rows != 48 {
		t.Errorf("rows = %d, want 48", builder.rows)
	}

	if builder.cmd.SandboxConfig == nil {
		t.Error("sandboxConfig should not be nil")
	}

	if len(builder.cmd.FilesToCreate) != 1 {
		t.Error("filesToCreate not set correctly")
	}
}

func TestPodBuilderBuildNilCommand(t *testing.T) {
	runner := &Runner{cfg: &config.Config{}}
	builder := NewPodBuilderFromRunner(runner)
	// Don't set command

	ctx := context.Background()
	_, err := builder.Build(ctx)

	if err == nil {
		t.Error("expected error for nil command")
	}

	if !contains(err.Error(), "command is required") {
		t.Errorf("error = %v, want containing 'command is required'", err)
	}
}

func TestPodBuilderBuildEmptyPodKey(t *testing.T) {
	runner := &Runner{cfg: &config.Config{}}
	cmd := &runnerv1.CreatePodCommand{
		LaunchCommand: "echo",
		AgentfileSource: "AGENT echo\nPROMPT_POSITION prepend\n",
		// PodKey is empty
	}
	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

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
	cmd := &runnerv1.CreatePodCommand{
		PodKey: "test-pod",
		// LaunchCommand is empty
	}
	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

	ctx := context.Background()
	_, err := builder.Build(ctx)

	if err == nil {
		t.Error("expected error for empty launch command")
	}

	if !contains(err.Error(), "launch_command is required") {
		t.Errorf("error = %v, want containing 'launch_command is required'", err)
	}
}

// Note: Additional tests are in pod_builder_test.go, pod_builder_extended_test.go, and pod_builder_integration_test.go
