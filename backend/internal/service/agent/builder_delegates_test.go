package agent

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// ---------------------------------------------------------------------------
// Tests for delegate methods on CodexCLI, GeminiCLI, OpenCode builders
// These are simple delegates to BaseAgentBuilder but still need coverage.
// ---------------------------------------------------------------------------

func newDelegateBuildContext() *BuildContext {
	return &BuildContext{
		Request: &ConfigBuildRequest{
			PodKey: "delegate-pod",
		},
		AgentType: &agent.AgentType{
			ID:            1,
			Slug:          "test-agent",
			LaunchCommand: "test-cmd",
			CommandTemplate: agent.CommandTemplate{
				Args: []agent.ArgRule{
					{Args: []string{"--flag", "val"}},
				},
			},
			FilesTemplate: agent.FilesTemplate{
				{
					PathTemplate:    "/tmp/test.txt",
					ContentTemplate: "hello",
					Mode:            0644,
				},
			},
			CredentialSchema: agent.CredentialSchema{
				{Name: "api_key", EnvVar: "API_KEY"},
			},
		},
		Config: agent.ConfigValues{},
		Credentials: agent.EncryptedCredentials{
			"api_key": "test-key",
		},
		IsRunnerHost: false,
		TemplateCtx:  map[string]interface{}{},
	}
}

// ---------------------------------------------------------------------------
// CodexCLIBuilder delegates
// ---------------------------------------------------------------------------

func TestCodexCLIBuilder_BuildLaunchArgs(t *testing.T) {
	builder := NewCodexCLIBuilder()
	ctx := newDelegateBuildContext()

	args, err := builder.BuildLaunchArgs(ctx)
	if err != nil {
		t.Fatalf("BuildLaunchArgs failed: %v", err)
	}
	if len(args) != 2 || args[0] != "--flag" || args[1] != "val" {
		t.Errorf("args = %v, want [--flag val]", args)
	}
}

// Note: TestCodexCLIBuilder_BuildFilesToCreate and TestCodexCLIBuilder_BuildEnvVars
// moved to builder_codex_cli_test.go after CodexCLIBuilder was rewritten to override
// these methods (no longer delegates to BaseAgentBuilder).

func TestCodexCLIBuilder_PostProcess(t *testing.T) {
	builder := NewCodexCLIBuilder()
	ctx := newDelegateBuildContext()
	cmd := &runnerv1.CreatePodCommand{}

	err := builder.PostProcess(ctx, cmd)
	if err != nil {
		t.Errorf("PostProcess should return nil, got %v", err)
	}
}

func TestCodexCLIBuilder_HandleInitialPrompt_NoPrompt(t *testing.T) {
	builder := NewCodexCLIBuilder()
	ctx := &BuildContext{
		Request: &ConfigBuildRequest{
			InitialPrompt: "",
		},
	}
	args := []string{"--flag"}

	result := builder.HandleInitialPrompt(ctx, args)
	if len(result) != 1 || result[0] != "--flag" {
		t.Errorf("expected args unchanged, got %v", result)
	}
}

// ---------------------------------------------------------------------------
// GeminiCLIBuilder delegates
// ---------------------------------------------------------------------------

func TestGeminiCLIBuilder_BuildLaunchArgs(t *testing.T) {
	builder := NewGeminiCLIBuilder()
	ctx := newDelegateBuildContext()

	args, err := builder.BuildLaunchArgs(ctx)
	if err != nil {
		t.Fatalf("BuildLaunchArgs failed: %v", err)
	}
	if len(args) != 2 || args[0] != "--flag" || args[1] != "val" {
		t.Errorf("args = %v, want [--flag val]", args)
	}
}

func TestGeminiCLIBuilder_BuildFilesToCreate(t *testing.T) {
	builder := NewGeminiCLIBuilder()
	ctx := newDelegateBuildContext()

	files, err := builder.BuildFilesToCreate(ctx)
	if err != nil {
		t.Fatalf("BuildFilesToCreate failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Content != "hello" {
		t.Errorf("Content = %q, want %q", files[0].Content, "hello")
	}
}

func TestGeminiCLIBuilder_BuildEnvVars(t *testing.T) {
	builder := NewGeminiCLIBuilder()
	ctx := newDelegateBuildContext()

	envVars, err := builder.BuildEnvVars(ctx)
	if err != nil {
		t.Fatalf("BuildEnvVars failed: %v", err)
	}
	if envVars["API_KEY"] != "test-key" {
		t.Errorf("API_KEY = %q, want %q", envVars["API_KEY"], "test-key")
	}
}

func TestGeminiCLIBuilder_PostProcess(t *testing.T) {
	builder := NewGeminiCLIBuilder()
	ctx := newDelegateBuildContext()
	cmd := &runnerv1.CreatePodCommand{}

	err := builder.PostProcess(ctx, cmd)
	if err != nil {
		t.Errorf("PostProcess should return nil, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// OpenCodeBuilder delegates
// ---------------------------------------------------------------------------

func TestOpenCodeBuilder_BuildLaunchArgs(t *testing.T) {
	builder := NewOpenCodeBuilder()
	ctx := newDelegateBuildContext()

	args, err := builder.BuildLaunchArgs(ctx)
	if err != nil {
		t.Fatalf("BuildLaunchArgs failed: %v", err)
	}
	if len(args) != 2 || args[0] != "--flag" || args[1] != "val" {
		t.Errorf("args = %v, want [--flag val]", args)
	}
}

func TestOpenCodeBuilder_BuildFilesToCreate(t *testing.T) {
	builder := NewOpenCodeBuilder()
	ctx := newDelegateBuildContext()

	files, err := builder.BuildFilesToCreate(ctx)
	if err != nil {
		t.Fatalf("BuildFilesToCreate failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
}

func TestOpenCodeBuilder_BuildEnvVars(t *testing.T) {
	builder := NewOpenCodeBuilder()
	ctx := newDelegateBuildContext()

	envVars, err := builder.BuildEnvVars(ctx)
	if err != nil {
		t.Fatalf("BuildEnvVars failed: %v", err)
	}
	if envVars["API_KEY"] != "test-key" {
		t.Errorf("API_KEY = %q, want %q", envVars["API_KEY"], "test-key")
	}
}

func TestOpenCodeBuilder_PostProcess(t *testing.T) {
	builder := NewOpenCodeBuilder()
	ctx := newDelegateBuildContext()
	cmd := &runnerv1.CreatePodCommand{}

	err := builder.PostProcess(ctx, cmd)
	if err != nil {
		t.Errorf("PostProcess should return nil, got %v", err)
	}
}

func TestOpenCodeBuilder_HandleInitialPrompt_NoPrompt(t *testing.T) {
	builder := NewOpenCodeBuilder()
	ctx := &BuildContext{
		Request: &ConfigBuildRequest{
			InitialPrompt: "",
		},
	}
	args := []string{"--flag"}

	result := builder.HandleInitialPrompt(ctx, args)
	if len(result) != 1 || result[0] != "--flag" {
		t.Errorf("expected args unchanged, got %v", result)
	}
}

// ---------------------------------------------------------------------------
// BaseAgentBuilder uncovered methods
// ---------------------------------------------------------------------------

func TestBaseAgentBuilder_HandleInitialPrompt(t *testing.T) {
	builder := NewBaseAgentBuilder("generic")

	t.Run("prepends prompt to args", func(t *testing.T) {
		ctx := &BuildContext{
			Request: &ConfigBuildRequest{
				InitialPrompt: "Do something",
			},
		}
		args := []string{"--flag"}

		result := builder.HandleInitialPrompt(ctx, args)
		if len(result) != 2 {
			t.Fatalf("expected 2 args, got %d", len(result))
		}
		if result[0] != "Do something" {
			t.Errorf("result[0] = %q, want %q", result[0], "Do something")
		}
		if result[1] != "--flag" {
			t.Errorf("result[1] = %q, want %q", result[1], "--flag")
		}
	})

	t.Run("returns args unchanged when no prompt", func(t *testing.T) {
		ctx := &BuildContext{
			Request: &ConfigBuildRequest{
				InitialPrompt: "",
			},
		}
		args := []string{"--flag"}

		result := builder.HandleInitialPrompt(ctx, args)
		if len(result) != 1 || result[0] != "--flag" {
			t.Errorf("expected args unchanged, got %v", result)
		}
	})
}

func TestBaseAgentBuilder_SupportsMcp(t *testing.T) {
	builder := NewBaseAgentBuilder("generic")
	if !builder.SupportsMcp() {
		t.Error("BaseAgentBuilder.SupportsMcp() should return true")
	}
}

func TestBaseAgentBuilder_SupportsSkills(t *testing.T) {
	builder := NewBaseAgentBuilder("generic")
	if !builder.SupportsSkills() {
		t.Error("BaseAgentBuilder.SupportsSkills() should return true")
	}
}

func TestBaseAgentBuilder_BuildResourcesToDownload(t *testing.T) {
	builder := NewBaseAgentBuilder("generic")
	ctx := &BuildContext{}

	resources, err := builder.BuildResourcesToDownload(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resources != nil {
		t.Errorf("expected nil, got %v", resources)
	}
}

func TestBaseAgentBuilder_GetConfigString(t *testing.T) {
	builder := NewBaseAgentBuilder("test")

	t.Run("returns string value", func(t *testing.T) {
		config := agent.ConfigValues{"model": "opus"}
		result := builder.getConfigString(config, "model")
		if result != "opus" {
			t.Errorf("getConfigString = %q, want %q", result, "opus")
		}
	})

	t.Run("returns empty for missing key", func(t *testing.T) {
		config := agent.ConfigValues{}
		result := builder.getConfigString(config, "model")
		if result != "" {
			t.Errorf("getConfigString = %q, want empty string", result)
		}
	})

	t.Run("returns empty for non-string value", func(t *testing.T) {
		config := agent.ConfigValues{"model": 42}
		result := builder.getConfigString(config, "model")
		if result != "" {
			t.Errorf("getConfigString = %q, want empty string for non-string value", result)
		}
	})
}

func TestBaseAgentBuilder_GetConfigBool(t *testing.T) {
	builder := NewBaseAgentBuilder("test")

	t.Run("returns bool value true", func(t *testing.T) {
		config := agent.ConfigValues{"debug": true}
		result := builder.getConfigBool(config, "debug")
		if !result {
			t.Error("getConfigBool should return true")
		}
	})

	t.Run("returns bool value false", func(t *testing.T) {
		config := agent.ConfigValues{"debug": false}
		result := builder.getConfigBool(config, "debug")
		if result {
			t.Error("getConfigBool should return false")
		}
	})

	t.Run("returns false for missing key", func(t *testing.T) {
		config := agent.ConfigValues{}
		result := builder.getConfigBool(config, "debug")
		if result {
			t.Error("getConfigBool should return false for missing key")
		}
	})

	t.Run("returns false for non-bool value", func(t *testing.T) {
		config := agent.ConfigValues{"debug": "yes"}
		result := builder.getConfigBool(config, "debug")
		if result {
			t.Error("getConfigBool should return false for non-bool value")
		}
	})
}

// ---------------------------------------------------------------------------
// NewConfigBuilderWithRegistry
// ---------------------------------------------------------------------------

func TestNewConfigBuilderWithRegistry(t *testing.T) {
	provider := &mockCredentialProvider{
		agentType: &agent.AgentType{
			ID:            1,
			Slug:          "claude-code",
			LaunchCommand: "claude",
		},
		credentials: agent.EncryptedCredentials{},
	}

	registry := NewAgentBuilderRegistry()

	builder := NewConfigBuilderWithRegistry(provider, registry)
	if builder == nil {
		t.Fatal("NewConfigBuilderWithRegistry returned nil")
	}

	// Verify that the builder uses the given registry by building a pod command
	// that exercises the registry lookup
	ctx := newDelegateBuildContext()
	ctx.Request.InitialPrompt = ""
	ctx.AgentType.Slug = "claude-code"

	// The builder should resolve claude-code from the registry
	cmd, err := builder.BuildPodCommand(t.Context(), &ConfigBuildRequest{
		AgentTypeID: 1,
		UserID:      1,
	})
	if err != nil {
		t.Fatalf("BuildPodCommand failed: %v", err)
	}
	if cmd == nil {
		t.Fatal("cmd should not be nil")
	}
}
