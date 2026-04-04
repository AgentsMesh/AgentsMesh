package runner

import (
	"context"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/workspace"
)

// Note: PodBuilder now uses WithCommand() with Proto types directly.
// Old builder methods like WithPodKey, WithWorkDirConfig are removed.

func TestPodBuilderBuildWithEmptyCommand(t *testing.T) {
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: "/tmp",
		},
	}

	builder := NewPodBuilderFromRunner(runner)
	// Don't set command

	_, err := builder.Build(context.Background())
	if err == nil {
		t.Error("expected error for nil command")
	}
	if !contains(err.Error(), "command is required") {
		t.Errorf("error = %v, want containing 'command is required'", err)
	}
}

func TestPodBuilderBuildWithEmptyPodKey(t *testing.T) {
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: "/tmp",
		},
	}

	cmd := &runnerv1.CreatePodCommand{
		LaunchCommand: "echo",
		AgentfileSource: "AGENT echo\nPROMPT_POSITION prepend\n",
		// PodKey is empty
	}

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

	_, err := builder.Build(context.Background())
	if err == nil {
		t.Error("expected error for empty pod key")
	}
	if !contains(err.Error(), "pod key is required") {
		t.Errorf("error = %v, want containing 'pod key is required'", err)
	}
}

func TestPodBuilderBuildWithAllOptions(t *testing.T) {
	tempDir := t.TempDir()
	ws, err := workspace.NewManager(tempDir, "")
	if err != nil {
		t.Skipf("Could not create workspace manager: %v", err)
	}

	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
			AgentEnvVars: map[string]string{
				"CONFIG_VAR": "config_value",
			},
		},
		workspace: ws,
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "all-options-pod",
		LaunchCommand: "echo",
		LaunchArgs:    []string{"hello", "world"},
		AgentfileSource: "AGENT echo\nPROMPT_POSITION prepend\n",
		EnvVars: map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
		},
	}

	pod, err := NewPodBuilderFromRunner(runner).
		WithCommand(cmd).
		WithPtySize(100, 30). // (cols, rows)
		Build(context.Background())

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if pod.PodKey != "all-options-pod" {
		t.Errorf("pod key = %s, want all-options-pod", pod.PodKey)
	}
	comps := testPTYComponents(pod)
	if comps == nil || comps.Terminal == nil {
		t.Error("terminal should not be nil")
	} else {
		comps.Terminal.Stop()
	}
}

func TestPodBuilderMergeEnvVarsWithNilConfig(t *testing.T) {
	runner := &Runner{
		cfg: nil,
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-pod",
		LaunchCommand: "echo",
		AgentfileSource: "AGENT echo\nPROMPT_POSITION prepend\n",
		EnvVars: map[string]string{
			"POD_VAR": "pod_value",
		},
	}

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

	result := builder.mergeEnvVars("")

	if len(result) != 1 {
		t.Errorf("result length = %d, want 1", len(result))
	}
	if result["POD_VAR"] != "pod_value" {
		t.Errorf("POD_VAR = %s, want pod_value", result["POD_VAR"])
	}
}

func TestPodBuilderMergeEnvVarsOverride(t *testing.T) {
	runner := &Runner{
		cfg: &config.Config{
			AgentEnvVars: map[string]string{
				"SHARED_VAR": "config_value",
				"CONFIG_VAR": "config_only",
			},
		},
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-pod",
		LaunchCommand: "echo",
		AgentfileSource: "AGENT echo\nPROMPT_POSITION prepend\n",
		EnvVars: map[string]string{
			"SHARED_VAR": "pod_value",
			"POD_VAR":    "pod_only",
		},
	}

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

	result := builder.mergeEnvVars("")

	// Command envVars should override config
	if result["SHARED_VAR"] != "pod_value" {
		t.Errorf("SHARED_VAR = %s, want pod_value", result["SHARED_VAR"])
	}
	if result["CONFIG_VAR"] != "config_only" {
		t.Errorf("CONFIG_VAR = %s, want config_only", result["CONFIG_VAR"])
	}
	if result["POD_VAR"] != "pod_only" {
		t.Errorf("POD_VAR = %s, want pod_only", result["POD_VAR"])
	}
}

func TestPodBuilderPtySizeDefaults(t *testing.T) {
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: "/tmp",
		},
	}

	builder := NewPodBuilderFromRunner(runner).
		WithPtySize(0, 0) // Zero values should use defaults

	if builder.rows != 24 {
		t.Errorf("rows = %d, want 24 (default)", builder.rows)
	}
	if builder.cols != 80 {
		t.Errorf("cols = %d, want 80 (default)", builder.cols)
	}
}

func TestPodBuilderWithLocalPath(t *testing.T) {
	tempDir := t.TempDir()
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
		workspace: nil,
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "local-path-pod",
		LaunchCommand: "echo",
		LaunchArgs:    []string{"test"},
		AgentfileSource: "AGENT echo\nPROMPT_POSITION prepend\n",
		SandboxConfig: &runnerv1.SandboxConfig{
			LocalPath: tempDir,
		},
	}

	pod, err := NewPodBuilderFromRunner(runner).
		WithCommand(cmd).
		Build(context.Background())

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if pod.PodKey != "local-path-pod" {
		t.Errorf("pod key = %s, want local-path-pod", pod.PodKey)
	}
	if pod.IO != nil {
		pod.IO.Stop()
	}
}

func TestPodBuilderWithFilesToCreate(t *testing.T) {
	tempDir := t.TempDir()
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "files-pod",
		LaunchCommand: "echo",
		LaunchArgs:    []string{"test"},
		AgentfileSource: "AGENT echo\nPROMPT_POSITION prepend\n",
		FilesToCreate: []*runnerv1.FileToCreate{
			{
				Path:    "{{.sandbox.root_path}}/config.json",
				Content: `{"key": "value"}`,
				Mode:    0644,
			},
		},
	}

	pod, err := NewPodBuilderFromRunner(runner).
		WithCommand(cmd).
		Build(context.Background())

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if pod.IO != nil {
		pod.IO.Stop()
	}
}

func TestPodBuilderWithEmptySandboxConfig(t *testing.T) {
	tempDir := t.TempDir()

	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "empty-sandbox-pod",
		LaunchCommand: "echo",
		LaunchArgs:    []string{"test"},
		AgentfileSource: "AGENT echo\nPROMPT_POSITION prepend\n",
		SandboxConfig: &runnerv1.SandboxConfig{
			// Empty sandbox config - creates empty workspace
		},
	}

	pod, err := NewPodBuilderFromRunner(runner).
		WithCommand(cmd).
		Build(context.Background())

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if pod.IO != nil {
		pod.IO.Stop()
	}
}

func TestPodBuilderCommandFields(t *testing.T) {
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: "/tmp",
		},
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "command-fields-pod",
		LaunchCommand: "test",
		LaunchArgs:    []string{"arg1"},
		AgentfileSource: "AGENT test\nPROMPT_POSITION prepend\n",
		EnvVars: map[string]string{
			"VAR1": "val1",
			"VAR2": "val2",
		},
		SandboxConfig: &runnerv1.SandboxConfig{
			RepositoryUrl:  "https://example.com/repo",
			SourceBranch:   "develop",
			CredentialType: "runner_local",
		},
		FilesToCreate: []*runnerv1.FileToCreate{
			{Path: "{{.sandbox.root_path}}/test.txt", Content: "test"},
		},
	}

	builder := NewPodBuilderFromRunner(runner).
		WithCommand(cmd).
		WithPtySize(120, 40) // (cols, rows)

	if builder.cmd.PodKey != "command-fields-pod" {
		t.Error("podKey not set")
	}
	if builder.cmd.LaunchCommand != "test" {
		t.Error("launchCommand not set")
	}
	if len(builder.cmd.LaunchArgs) != 1 || builder.cmd.LaunchArgs[0] != "arg1" {
		t.Error("launchArgs not set correctly")
	}
	if builder.cmd.EnvVars["VAR1"] != "val1" || builder.cmd.EnvVars["VAR2"] != "val2" {
		t.Error("envVars not set correctly")
	}
	if builder.rows != 40 || builder.cols != 120 {
		t.Error("PTY size not set correctly")
	}
	if builder.cmd.SandboxConfig == nil {
		t.Error("sandboxConfig not set")
	} else {
		if builder.cmd.SandboxConfig.RepositoryUrl != "https://example.com/repo" {
			t.Error("sandboxConfig repositoryUrl not set correctly")
		}
		if builder.cmd.SandboxConfig.SourceBranch != "develop" {
			t.Error("sandboxConfig branch not set correctly")
		}
		if builder.cmd.SandboxConfig.CredentialType != "runner_local" {
			t.Error("sandboxConfig credentialType not set correctly")
		}
	}
	if len(builder.cmd.FilesToCreate) != 1 {
		t.Error("filesToCreate not set correctly")
	}
}
