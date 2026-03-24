package runner

import (
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/config"
)

// Tests for environment variable merging

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

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

	result := builder.mergeEnvVars("", "")

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

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

	result := builder.mergeEnvVars("", "")

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

	builder := NewPodBuilderFromRunner(runner).
		WithCommand(cmd).
		WithTerminalSize(120, 40) // (cols, rows)

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
		NewPodBuilderFromRunner(runner)
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
		NewPodBuilderFromRunner(runner).
			WithCommand(cmd).
			WithTerminalSize(120, 40) // (cols, rows)
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

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder.mergeEnvVars("", "")
	}
}
