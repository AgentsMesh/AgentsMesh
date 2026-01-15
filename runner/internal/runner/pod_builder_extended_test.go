package runner

import (
	"context"
	"testing"

	"github.com/anthropics/agentmesh/runner/internal/client"
	"github.com/anthropics/agentmesh/runner/internal/config"
	"github.com/anthropics/agentmesh/runner/internal/workspace"
)

// Note: Worktree functionality is now handled by PodBuilder.setupWorkDir
// based on WorkDirConfig from Backend.

func TestPodBuilderBuildWithEmptyPodKey(t *testing.T) {
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: "/tmp",
		},
	}

	builder := NewPodBuilder(runner)
	// Don't set pod key

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

	builder := NewPodBuilder(runner).
		WithPodKey("all-options-pod").
		WithAgentType("claude-code").
		WithLaunchCommand("echo", []string{"hello", "world"}).
		WithEnvVars(map[string]string{"VAR1": "value1"}).
		WithEnvVar("VAR2", "value2").
		WithTerminalSize(30, 100).
		WithInitialPrompt("Hello!")

	pod, err := builder.Build(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if pod.PodKey != "all-options-pod" {
		t.Errorf("pod key = %s, want all-options-pod", pod.PodKey)
	}
	if pod.AgentType != "claude-code" {
		t.Errorf("agent type = %s, want claude-code", pod.AgentType)
	}
	if pod.InitialPrompt != "Hello!" {
		t.Errorf("initial prompt = %s, want Hello!", pod.InitialPrompt)
	}
	if pod.Terminal == nil {
		t.Error("terminal should not be nil")
	} else {
		pod.Terminal.Stop()
	}
}

func TestPodBuilderMergeEnvVarsWithNilConfig(t *testing.T) {
	runner := &Runner{
		cfg: nil,
	}

	builder := NewPodBuilder(runner).
		WithEnvVar("POD_VAR", "pod_value")

	result := builder.mergeEnvVars()

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

	builder := NewPodBuilder(runner).
		WithEnvVar("SHARED_VAR", "pod_value").
		WithEnvVar("POD_VAR", "pod_only")

	result := builder.mergeEnvVars()

	// Pod builder envVars should override config
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

func TestPodBuilderTerminalSizeDefaults(t *testing.T) {
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: "/tmp",
		},
	}

	builder := NewPodBuilder(runner).
		WithTerminalSize(0, 0) // Zero values should use defaults

	if builder.rows != 24 {
		t.Errorf("rows = %d, want 24 (default)", builder.rows)
	}
	if builder.cols != 80 {
		t.Errorf("cols = %d, want 80 (default)", builder.cols)
	}
}

func TestPodBuilderWithLocalWorkDirConfig(t *testing.T) {
	tempDir := t.TempDir()
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
		workspace: nil,
	}

	builder := NewPodBuilder(runner).
		WithPodKey("local-workdir-pod").
		WithLaunchCommand("echo", []string{"test"}).
		WithWorkDirConfig(&client.WorkDirConfig{
			Type:      "local",
			LocalPath: tempDir,
		})

	pod, err := builder.Build(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if pod.PodKey != "local-workdir-pod" {
		t.Errorf("pod key = %s, want local-workdir-pod", pod.PodKey)
	}
	if pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

func TestPodBuilderWithFilesToCreate(t *testing.T) {
	tempDir := t.TempDir()
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	builder := NewPodBuilder(runner).
		WithPodKey("files-pod").
		WithLaunchCommand("echo", []string{"test"}).
		WithFilesToCreate([]client.FileToCreate{
			{
				PathTemplate: "{{.sandbox.root_path}}/config.json",
				Content:      `{"key": "value"}`,
				Mode:         0644,
			},
		})

	pod, err := builder.Build(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

func TestPodBuilderWithTempDirConfig(t *testing.T) {
	tempDir := t.TempDir()

	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	builder := NewPodBuilder(runner).
		WithPodKey("tempdir-pod").
		WithLaunchCommand("echo", []string{"test"}).
		WithWorkDirConfig(&client.WorkDirConfig{
			Type: "tempdir",
		})

	pod, err := builder.Build(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

func TestPodBuilderFluentChaining(t *testing.T) {
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: "/tmp",
		},
	}

	builder := NewPodBuilder(runner).
		WithPodKey("chain-pod").
		WithAgentType("test-agent").
		WithLaunchCommand("test", []string{"arg1"}).
		WithEnvVars(map[string]string{"VAR1": "val1"}).
		WithEnvVar("VAR2", "val2").
		WithTerminalSize(40, 120).
		WithInitialPrompt("prompt").
		WithWorkDirConfig(&client.WorkDirConfig{
			Type:          "worktree",
			RepositoryURL: "https://example.com/repo",
			Branch:        "develop",
		}).
		WithFilesToCreate([]client.FileToCreate{
			{PathTemplate: "{{.sandbox.root_path}}/test.txt", Content: "test"},
		})

	if builder.podKey != "chain-pod" {
		t.Error("podKey not set")
	}
	if builder.agentType != "test-agent" {
		t.Error("agentType not set")
	}
	if builder.launchCommand != "test" {
		t.Error("launchCommand not set")
	}
	if len(builder.launchArgs) != 1 || builder.launchArgs[0] != "arg1" {
		t.Error("launchArgs not set correctly")
	}
	if builder.envVars["VAR1"] != "val1" || builder.envVars["VAR2"] != "val2" {
		t.Error("envVars not set correctly")
	}
	if builder.rows != 40 || builder.cols != 120 {
		t.Error("terminal size not set correctly")
	}
	if builder.initialPrompt != "prompt" {
		t.Error("initialPrompt not set")
	}
	if builder.workDirConfig == nil {
		t.Error("workDirConfig not set")
	} else {
		if builder.workDirConfig.Type != "worktree" {
			t.Error("workDirConfig type not set correctly")
		}
		if builder.workDirConfig.RepositoryURL != "https://example.com/repo" {
			t.Error("workDirConfig repositoryURL not set correctly")
		}
		if builder.workDirConfig.Branch != "develop" {
			t.Error("workDirConfig branch not set correctly")
		}
	}
	if len(builder.filesToCreate) != 1 {
		t.Error("filesToCreate not set correctly")
	}
}
