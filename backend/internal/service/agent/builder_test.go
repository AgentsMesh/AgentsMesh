package agent

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

func TestAgentBuilderRegistry(t *testing.T) {
	t.Run("creates with default builders", func(t *testing.T) {
		registry := NewAgentBuilderRegistry()

		// Check that all built-in builders are registered
		slugs := []string{"claude-code", "codex-cli", "gemini-cli", "aider", "opencode"}
		for _, slug := range slugs {
			if !registry.Has(slug) {
				t.Errorf("Registry should have builder for %s", slug)
			}
		}
	})

	t.Run("returns fallback for unknown slug", func(t *testing.T) {
		registry := NewAgentBuilderRegistry()

		builder := registry.Get("unknown-agent")
		if builder == nil {
			t.Error("Should return fallback builder for unknown slug")
		}
		if builder.Slug() != "default" {
			t.Errorf("Fallback builder slug = %s, want default", builder.Slug())
		}
	})

	t.Run("register custom builder", func(t *testing.T) {
		registry := NewAgentBuilderRegistry()

		customBuilder := NewBaseAgentBuilder("custom-agent")
		registry.Register(customBuilder)

		if !registry.Has("custom-agent") {
			t.Error("Should have custom-agent after registration")
		}
	})

	t.Run("list returns all slugs", func(t *testing.T) {
		registry := NewAgentBuilderRegistry()

		slugs := registry.List()
		if len(slugs) < 5 {
			t.Errorf("List should return at least 5 slugs, got %d", len(slugs))
		}
	})

	t.Run("set fallback", func(t *testing.T) {
		registry := NewAgentBuilderRegistry()

		customFallback := NewBaseAgentBuilder("custom-fallback")
		registry.SetFallback(customFallback)

		builder := registry.Get("unknown-agent")
		if builder.Slug() != "custom-fallback" {
			t.Errorf("Should use custom fallback, got %s", builder.Slug())
		}
	})
}

func TestClaudeCodeBuilder_HandleInitialPrompt(t *testing.T) {
	builder := NewClaudeCodeBuilder()

	t.Run("prepends prompt to args", func(t *testing.T) {
		ctx := &BuildContext{
			Request: &ConfigBuildRequest{
				InitialPrompt: "Fix the bug",
			},
		}
		args := []string{"--model", "opus"}

		result := builder.HandleInitialPrompt(ctx, args)

		if len(result) != 3 {
			t.Fatalf("Result length = %d, want 3", len(result))
		}
		if result[0] != "Fix the bug" {
			t.Errorf("First arg = %s, want 'Fix the bug'", result[0])
		}
		if result[1] != "--model" {
			t.Errorf("Second arg = %s, want '--model'", result[1])
		}
	})

	t.Run("returns args unchanged when no prompt", func(t *testing.T) {
		ctx := &BuildContext{
			Request: &ConfigBuildRequest{
				InitialPrompt: "",
			},
		}
		args := []string{"--model", "opus"}

		result := builder.HandleInitialPrompt(ctx, args)

		if len(result) != 2 {
			t.Errorf("Result length = %d, want 2", len(result))
		}
	})
}

func TestGeminiCLIBuilder_HandleInitialPrompt(t *testing.T) {
	builder := NewGeminiCLIBuilder()

	t.Run("appends prompt to args", func(t *testing.T) {
		ctx := &BuildContext{
			Request: &ConfigBuildRequest{
				InitialPrompt: "Fix the bug",
			},
		}
		args := []string{"--sandbox"}

		result := builder.HandleInitialPrompt(ctx, args)

		if len(result) != 2 {
			t.Fatalf("Result length = %d, want 2", len(result))
		}
		if result[0] != "--sandbox" {
			t.Errorf("First arg = %s, want '--sandbox'", result[0])
		}
		if result[1] != "Fix the bug" {
			t.Errorf("Last arg = %s, want 'Fix the bug'", result[1])
		}
	})

	t.Run("returns args unchanged when no prompt", func(t *testing.T) {
		ctx := &BuildContext{
			Request: &ConfigBuildRequest{
				InitialPrompt: "",
			},
		}
		args := []string{"--sandbox"}

		result := builder.HandleInitialPrompt(ctx, args)

		if len(result) != 1 {
			t.Errorf("Result length = %d, want 1", len(result))
		}
	})
}

func TestAiderBuilder_HandleInitialPrompt(t *testing.T) {
	builder := NewAiderBuilder()

	t.Run("ignores prompt", func(t *testing.T) {
		ctx := &BuildContext{
			Request: &ConfigBuildRequest{
				InitialPrompt: "Fix the bug",
			},
		}
		args := []string{"--model", "gpt-4"}

		result := builder.HandleInitialPrompt(ctx, args)

		// Aider should ignore the prompt and return args unchanged
		if len(result) != 2 {
			t.Fatalf("Result length = %d, want 2", len(result))
		}
		if result[0] != "--model" || result[1] != "gpt-4" {
			t.Errorf("Args should be unchanged, got %v", result)
		}
	})
}

func TestCodexCLIBuilder_HandleInitialPrompt(t *testing.T) {
	builder := NewCodexCLIBuilder()

	t.Run("prepends prompt to args", func(t *testing.T) {
		ctx := &BuildContext{
			Request: &ConfigBuildRequest{
				InitialPrompt: "Fix the bug",
			},
		}
		args := []string{"--approval-mode", "auto-edit"}

		result := builder.HandleInitialPrompt(ctx, args)

		if len(result) != 3 {
			t.Fatalf("Result length = %d, want 3", len(result))
		}
		if result[0] != "Fix the bug" {
			t.Errorf("First arg = %s, want 'Fix the bug'", result[0])
		}
	})
}

func TestOpenCodeBuilder_HandleInitialPrompt(t *testing.T) {
	builder := NewOpenCodeBuilder()

	t.Run("prepends prompt to args", func(t *testing.T) {
		ctx := &BuildContext{
			Request: &ConfigBuildRequest{
				InitialPrompt: "Fix the bug",
			},
		}
		args := []string{}

		result := builder.HandleInitialPrompt(ctx, args)

		if len(result) != 1 {
			t.Fatalf("Result length = %d, want 1", len(result))
		}
		if result[0] != "Fix the bug" {
			t.Errorf("First arg = %s, want 'Fix the bug'", result[0])
		}
	})
}

func TestBaseAgentBuilder_BuildLaunchArgs(t *testing.T) {
	builder := NewBaseAgentBuilder("test")

	t.Run("builds args from command template", func(t *testing.T) {
		ctx := &BuildContext{
			AgentType: &agent.AgentType{
				CommandTemplate: agent.CommandTemplate{
					Args: []agent.ArgRule{
						{Args: []string{"--model", "opus"}},
						{Args: []string{"--verbose"}},
					},
				},
			},
			Config:      agent.ConfigValues{},
			TemplateCtx: map[string]interface{}{},
		}

		args, err := builder.BuildLaunchArgs(ctx)
		if err != nil {
			t.Fatalf("BuildLaunchArgs failed: %v", err)
		}

		if len(args) != 3 {
			t.Fatalf("Args length = %d, want 3", len(args))
		}
		if args[0] != "--model" || args[1] != "opus" || args[2] != "--verbose" {
			t.Errorf("Args = %v, unexpected values", args)
		}
	})

	t.Run("skips args when condition not met", func(t *testing.T) {
		ctx := &BuildContext{
			AgentType: &agent.AgentType{
				CommandTemplate: agent.CommandTemplate{
					Args: []agent.ArgRule{
						{
							Condition: &agent.Condition{
								Field:    "debug",
								Operator: "eq",
								Value:    true,
							},
							Args: []string{"--debug"},
						},
					},
				},
			},
			Config:      agent.ConfigValues{"debug": false},
			TemplateCtx: map[string]interface{}{},
		}

		args, err := builder.BuildLaunchArgs(ctx)
		if err != nil {
			t.Fatalf("BuildLaunchArgs failed: %v", err)
		}

		if len(args) != 0 {
			t.Errorf("Args should be empty when condition not met, got %v", args)
		}
	})

	t.Run("renders template variables", func(t *testing.T) {
		ctx := &BuildContext{
			AgentType: &agent.AgentType{
				CommandTemplate: agent.CommandTemplate{
					Args: []agent.ArgRule{
						{Args: []string{"--model", "{{.config.model}}"}},
					},
				},
			},
			Config: agent.ConfigValues{"model": "sonnet"},
			TemplateCtx: map[string]interface{}{
				"config": agent.ConfigValues{"model": "sonnet"},
			},
		}

		args, err := builder.BuildLaunchArgs(ctx)
		if err != nil {
			t.Fatalf("BuildLaunchArgs failed: %v", err)
		}

		if len(args) != 2 {
			t.Fatalf("Args length = %d, want 2", len(args))
		}
		if args[1] != "sonnet" {
			t.Errorf("Model arg = %s, want sonnet", args[1])
		}
	})
}

func TestBaseAgentBuilder_BuildEnvVars(t *testing.T) {
	builder := NewBaseAgentBuilder("test")

	t.Run("maps credentials to env vars", func(t *testing.T) {
		ctx := &BuildContext{
			AgentType: &agent.AgentType{
				CredentialSchema: agent.CredentialSchema{
					{Name: "api_key", EnvVar: "API_KEY"},
					{Name: "secret", EnvVar: "SECRET"},
				},
			},
			Credentials: agent.EncryptedCredentials{
				"api_key": "test-key",
				"secret":  "test-secret",
			},
			IsRunnerHost: false,
		}

		envVars, err := builder.BuildEnvVars(ctx)
		if err != nil {
			t.Fatalf("BuildEnvVars failed: %v", err)
		}

		if envVars["API_KEY"] != "test-key" {
			t.Errorf("API_KEY = %s, want test-key", envVars["API_KEY"])
		}
		if envVars["SECRET"] != "test-secret" {
			t.Errorf("SECRET = %s, want test-secret", envVars["SECRET"])
		}
	})

	t.Run("returns empty for runner host mode", func(t *testing.T) {
		ctx := &BuildContext{
			AgentType: &agent.AgentType{
				CredentialSchema: agent.CredentialSchema{
					{Name: "api_key", EnvVar: "API_KEY"},
				},
			},
			Credentials: agent.EncryptedCredentials{
				"api_key": "test-key",
			},
			IsRunnerHost: true,
		}

		envVars, err := builder.BuildEnvVars(ctx)
		if err != nil {
			t.Fatalf("BuildEnvVars failed: %v", err)
		}

		if len(envVars) != 0 {
			t.Errorf("EnvVars should be empty for runner host mode, got %v", envVars)
		}
	})

	t.Run("skips empty credential values", func(t *testing.T) {
		ctx := &BuildContext{
			AgentType: &agent.AgentType{
				CredentialSchema: agent.CredentialSchema{
					{Name: "api_key", EnvVar: "API_KEY"},
					{Name: "empty", EnvVar: "EMPTY"},
				},
			},
			Credentials: agent.EncryptedCredentials{
				"api_key": "test-key",
				"empty":   "",
			},
			IsRunnerHost: false,
		}

		envVars, err := builder.BuildEnvVars(ctx)
		if err != nil {
			t.Fatalf("BuildEnvVars failed: %v", err)
		}

		if _, exists := envVars["EMPTY"]; exists {
			t.Error("EMPTY should not be in envVars")
		}
	})
}

func TestBaseAgentBuilder_BuildFilesToCreate(t *testing.T) {
	builder := NewBaseAgentBuilder("test")

	t.Run("builds files from template", func(t *testing.T) {
		ctx := &BuildContext{
			AgentType: &agent.AgentType{
				FilesTemplate: agent.FilesTemplate{
					{
						PathTemplate:    "/tmp/config.json",
						ContentTemplate: `{"key":"value"}`,
						Mode:            0600,
					},
				},
			},
			Config:      agent.ConfigValues{},
			TemplateCtx: map[string]interface{}{},
		}

		files, err := builder.BuildFilesToCreate(ctx)
		if err != nil {
			t.Fatalf("BuildFilesToCreate failed: %v", err)
		}

		if len(files) != 1 {
			t.Fatalf("Files length = %d, want 1", len(files))
		}
		if files[0].Path != "/tmp/config.json" {
			t.Errorf("Path = %s, want /tmp/config.json", files[0].Path)
		}
		if files[0].Content != `{"key":"value"}` {
			t.Errorf("Content = %s, unexpected", files[0].Content)
		}
		if files[0].Mode != 0600 {
			t.Errorf("Mode = %o, want 0600", files[0].Mode)
		}
	})

	t.Run("creates directories", func(t *testing.T) {
		ctx := &BuildContext{
			AgentType: &agent.AgentType{
				FilesTemplate: agent.FilesTemplate{
					{
						PathTemplate: "/tmp/mydir",
						IsDirectory:  true,
					},
				},
			},
			Config:      agent.ConfigValues{},
			TemplateCtx: map[string]interface{}{},
		}

		files, err := builder.BuildFilesToCreate(ctx)
		if err != nil {
			t.Fatalf("BuildFilesToCreate failed: %v", err)
		}

		if len(files) != 1 {
			t.Fatalf("Files length = %d, want 1", len(files))
		}
		if !files[0].IsDirectory {
			t.Error("IsDirectory should be true")
		}
	})

	t.Run("skips files when condition not met", func(t *testing.T) {
		ctx := &BuildContext{
			AgentType: &agent.AgentType{
				FilesTemplate: agent.FilesTemplate{
					{
						Condition: &agent.Condition{
							Field:    "mcp_enabled",
							Operator: "eq",
							Value:    true,
						},
						PathTemplate:    "/tmp/mcp.json",
						ContentTemplate: "{}",
					},
				},
			},
			Config:      agent.ConfigValues{"mcp_enabled": false},
			TemplateCtx: map[string]interface{}{},
		}

		files, err := builder.BuildFilesToCreate(ctx)
		if err != nil {
			t.Fatalf("BuildFilesToCreate failed: %v", err)
		}

		if len(files) != 0 {
			t.Errorf("Files should be empty when condition not met, got %v", files)
		}
	})

	t.Run("uses default mode when not specified", func(t *testing.T) {
		ctx := &BuildContext{
			AgentType: &agent.AgentType{
				FilesTemplate: agent.FilesTemplate{
					{
						PathTemplate:    "/tmp/config.json",
						ContentTemplate: "{}",
						Mode:            0, // Not specified
					},
				},
			},
			Config:      agent.ConfigValues{},
			TemplateCtx: map[string]interface{}{},
		}

		files, err := builder.BuildFilesToCreate(ctx)
		if err != nil {
			t.Fatalf("BuildFilesToCreate failed: %v", err)
		}

		if files[0].Mode != 0644 {
			t.Errorf("Mode = %o, want 0644 (default)", files[0].Mode)
		}
	})
}

func TestBaseAgentBuilder_PostProcess(t *testing.T) {
	builder := NewBaseAgentBuilder("test")

	t.Run("returns nil by default", func(t *testing.T) {
		ctx := &BuildContext{}
		cmd := &runnerv1.CreatePodCommand{}

		err := builder.PostProcess(ctx, cmd)
		if err != nil {
			t.Errorf("PostProcess should return nil, got %v", err)
		}
	})
}

func TestBuildContext(t *testing.T) {
	t.Run("NewBuildContext creates context correctly", func(t *testing.T) {
		req := &ConfigBuildRequest{
			AgentTypeID: 1,
			UserID:      2,
		}
		agentType := &agent.AgentType{
			Slug: "test-agent",
		}
		config := agent.ConfigValues{"key": "value"}
		creds := agent.EncryptedCredentials{"secret": "hidden"}
		templateCtx := map[string]interface{}{"template": "data"}

		ctx := NewBuildContext(req, agentType, config, creds, true, templateCtx)

		if ctx.Request != req {
			t.Error("Request not set correctly")
		}
		if ctx.AgentType != agentType {
			t.Error("AgentType not set correctly")
		}
		if ctx.Config["key"] != "value" {
			t.Error("Config not set correctly")
		}
		if ctx.Credentials["secret"] != "hidden" {
			t.Error("Credentials not set correctly")
		}
		if !ctx.IsRunnerHost {
			t.Error("IsRunnerHost not set correctly")
		}
		if ctx.TemplateCtx["template"] != "data" {
			t.Error("TemplateCtx not set correctly")
		}
	})
}

func TestBuilderSlugs(t *testing.T) {
	tests := []struct {
		name     string
		builder  AgentBuilder
		expected string
	}{
		{"ClaudeCode", NewClaudeCodeBuilder(), ClaudeCodeSlug},
		{"CodexCLI", NewCodexCLIBuilder(), CodexCLISlug},
		{"GeminiCLI", NewGeminiCLIBuilder(), GeminiCLISlug},
		{"Aider", NewAiderBuilder(), AiderSlug},
		{"OpenCode", NewOpenCodeBuilder(), OpenCodeSlug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.builder.Slug() != tt.expected {
				t.Errorf("Slug() = %s, want %s", tt.builder.Slug(), tt.expected)
			}
		})
	}
}
