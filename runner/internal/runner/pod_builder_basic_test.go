package runner

import (
	"context"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/config"
)

// Tests for basic PodBuilder functionality

func TestPodBuilderStruct(t *testing.T) {
	runner := &Runner{cfg: &config.Config{}}
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "pod-1",
		LaunchCommand: "claude",
		LaunchArgs:    []string{"--headless"},
		EnvVars:       map[string]string{"KEY": "VALUE"},
		AgentfileSource: "AGENT claude\nPROMPT_POSITION prepend\n",
		FilesToCreate: []*runnerv1.FileToCreate{
			{Path: "{{.sandbox.root_path}}/test.txt", Content: "test"},
		},
		SandboxConfig: &runnerv1.SandboxConfig{
			LocalPath: "/tmp",
		},
	}

	builder := NewPodBuilderFromRunner(runner).
		WithCommand(cmd).
		WithPtySize(80, 24) // (cols, rows)

	if builder.cmd.PodKey != "pod-1" {
		t.Errorf("podKey: got %v, want pod-1", builder.cmd.PodKey)
	}

	if builder.rows != 24 {
		t.Errorf("rows: got %v, want 24", builder.rows)
	}
	if builder.cols != 80 {
		t.Errorf("cols: got %v, want 80", builder.cols)
	}
}

func TestPodBuilderFluentAPI(t *testing.T) {
	runner := &Runner{}
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "pod-1",
		LaunchCommand: "claude",
		LaunchArgs:    []string{"--headless"},
		AgentfileSource: "AGENT claude\nPROMPT_POSITION prepend\n",
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

	builder := NewPodBuilderFromRunner(runner)
	result := builder.
		WithCommand(cmd).
		WithPtySize(120, 40) // (cols, rows)

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
	builder := NewPodBuilderFromRunner(runner)

	if builder.rows != 24 {
		t.Errorf("default rows: got %v, want 24", builder.rows)
	}

	if builder.cols != 80 {
		t.Errorf("default cols: got %v, want 80", builder.cols)
	}
}

func TestPodBuilderPtySizeValidation(t *testing.T) {
	runner := &Runner{}
	builder := NewPodBuilderFromRunner(runner)

	// Test with invalid values (should use defaults)
	builder.WithPtySize(0, 0)

	if builder.rows != 24 {
		t.Errorf("rows with zero: got %v, want 24 (default)", builder.rows)
	}

	if builder.cols != 80 {
		t.Errorf("cols with zero: got %v, want 80 (default)", builder.cols)
	}

	// Test with negative values (should use defaults)
	builder.WithPtySize(-1, -1)

	if builder.rows != 24 {
		t.Errorf("rows with negative: got %v, want 24 (default)", builder.rows)
	}
}

func TestPodBuilderBuildWithoutCommand(t *testing.T) {
	runner := &Runner{}
	builder := NewPodBuilderFromRunner(runner)

	_, err := builder.Build(context.Background())
	if err == nil {
		t.Error("expected error for missing command")
	}
}

func TestPodStatusConstantsInBuilder(t *testing.T) {
	// Verify pod status constants
	if PodStatusInitializing != "initializing" {
		t.Errorf("PodStatusInitializing = %v, want initializing", PodStatusInitializing)
	}
	if PodStatusRunning != "running" {
		t.Errorf("PodStatusRunning = %v, want running", PodStatusRunning)
	}
}
