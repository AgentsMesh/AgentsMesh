package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

func TestConfigBuilder_BuildPodCommand(t *testing.T) {
	ctx := context.Background()

	t.Run("basic pod command", func(t *testing.T) {
		db := setupConfigBuilderTestDB(t)
		provider := createTestProvider(db)
		builder := NewConfigBuilder(provider)

		var at agent.AgentType
		if err := db.First(&at).Error; err != nil {
			t.Fatalf("Failed to get agent type: %v", err)
		}

		req := &ConfigBuildRequest{
			AgentTypeID:    at.ID,
			UserID:         1,
			OrganizationID: 1,
			MCPPort:        19000,
			PodKey:         "test-pod-123",
		}

		cmd, err := builder.BuildPodCommand(ctx, req)
		if err != nil {
			t.Fatalf("BuildPodCommand failed: %v", err)
		}

		if cmd.LaunchCommand != "claude" {
			t.Errorf("LaunchCommand = %s, want claude", cmd.LaunchCommand)
		}
		// No repository URL = no SandboxConfig
		if cmd.SandboxConfig != nil {
			t.Errorf("SandboxConfig should be nil without repository, got %+v", cmd.SandboxConfig)
		}
	})

	t.Run("with repository", func(t *testing.T) {
		db := setupConfigBuilderTestDB(t)
		provider := createTestProvider(db)
		builder := NewConfigBuilder(provider)

		var at agent.AgentType
		db.First(&at)

		req := &ConfigBuildRequest{
			AgentTypeID:    at.ID,
			UserID:         1,
			OrganizationID: 1,
			RepositoryURL:  "https://github.com/test/repo.git",
			SourceBranch:   "main",
			MCPPort:        19000,
			PodKey:         "test-pod-456",
		}

		cmd, err := builder.BuildPodCommand(ctx, req)
		if err != nil {
			t.Fatalf("BuildPodCommand failed: %v", err)
		}

		if cmd.SandboxConfig == nil {
			t.Fatal("SandboxConfig should not be nil with repository")
		}
		if cmd.SandboxConfig.RepositoryUrl != "https://github.com/test/repo.git" {
			t.Errorf("RepositoryUrl = %s, want https://github.com/test/repo.git", cmd.SandboxConfig.RepositoryUrl)
		}
		if cmd.SandboxConfig.SourceBranch != "main" {
			t.Errorf("SourceBranch = %s, want main", cmd.SandboxConfig.SourceBranch)
		}
	})

	t.Run("with local path", func(t *testing.T) {
		db := setupConfigBuilderTestDB(t)
		provider := createTestProvider(db)
		builder := NewConfigBuilder(provider)

		var at agent.AgentType
		db.First(&at)

		req := &ConfigBuildRequest{
			AgentTypeID:    at.ID,
			UserID:         1,
			OrganizationID: 1,
			LocalPath:      "/home/user/project",
			MCPPort:        19000,
			PodKey:         "test-pod-789",
		}

		cmd, err := builder.BuildPodCommand(ctx, req)
		if err != nil {
			t.Fatalf("BuildPodCommand failed: %v", err)
		}

		if cmd.SandboxConfig == nil {
			t.Fatal("SandboxConfig should not be nil with local path")
		}
		if cmd.SandboxConfig.LocalPath != "/home/user/project" {
			t.Errorf("LocalPath = %s, want /home/user/project", cmd.SandboxConfig.LocalPath)
		}
	})

	t.Run("with config overrides", func(t *testing.T) {
		db := setupConfigBuilderTestDB(t)
		provider := createTestProvider(db)
		builder := NewConfigBuilder(provider)

		var at agent.AgentType
		db.First(&at)

		req := &ConfigBuildRequest{
			AgentTypeID:    at.ID,
			UserID:         1,
			OrganizationID: 1,
			ConfigOverrides: map[string]interface{}{
				"model": "sonnet",
			},
			MCPPort: 19000,
			PodKey:  "test-pod-override",
		}

		cmd, err := builder.BuildPodCommand(ctx, req)
		if err != nil {
			t.Fatalf("BuildPodCommand failed: %v", err)
		}

		found := false
		for i, arg := range cmd.LaunchArgs {
			if arg == "--model" && i+1 < len(cmd.LaunchArgs) && cmd.LaunchArgs[i+1] == "sonnet" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("LaunchArgs should contain --model sonnet, got %v", cmd.LaunchArgs)
		}
	})

	t.Run("with initial prompt", func(t *testing.T) {
		db := setupConfigBuilderTestDB(t)
		provider := createTestProvider(db)
		builder := NewConfigBuilder(provider)

		var at agent.AgentType
		db.First(&at)

		req := &ConfigBuildRequest{
			AgentTypeID:    at.ID,
			UserID:         1,
			OrganizationID: 1,
			InitialPrompt:  "Fix the bug in main.go",
			MCPPort:        19000,
			PodKey:         "test-pod-prompt",
		}

		cmd, err := builder.BuildPodCommand(ctx, req)
		if err != nil {
			t.Fatalf("BuildPodCommand failed: %v", err)
		}

		// InitialPrompt is now prepended to LaunchArgs as the first argument
		if len(cmd.LaunchArgs) == 0 || cmd.LaunchArgs[0] != "Fix the bug in main.go" {
			t.Errorf("LaunchArgs[0] = %v, want Fix the bug in main.go (InitialPrompt should be first arg)", cmd.LaunchArgs)
		}
	})

	t.Run("invalid agent type", func(t *testing.T) {
		db := setupConfigBuilderTestDB(t)
		provider := createTestProvider(db)
		builder := NewConfigBuilder(provider)

		req := &ConfigBuildRequest{
			AgentTypeID:    99999,
			UserID:         1,
			OrganizationID: 1,
		}

		_, err := builder.BuildPodCommand(ctx, req)
		if err == nil {
			t.Error("Expected error for invalid agent type")
		}
	})
}

func TestConfigBuilder_BuildPodCommand_ErrorPaths(t *testing.T) {
	ctx := context.Background()

	t.Run("returns error on buildEnvVars failure", func(t *testing.T) {
		provider := &mockCredentialProvider{
			agentType: &agent.AgentType{
				ID:            1,
				Slug:          "claude-code",
				LaunchCommand: "claude",
			},
			credErr: fmt.Errorf("credential error"),
		}

		builder := NewConfigBuilder(provider)
		req := &ConfigBuildRequest{
			AgentTypeID: 1,
			UserID:      1,
		}

		_, err := builder.BuildPodCommand(ctx, req)
		if err == nil {
			t.Error("Expected error for credential failure")
		}
		if !strings.Contains(err.Error(), "failed to build env vars") {
			t.Errorf("Error should contain 'failed to build env vars', got: %v", err)
		}
	})

	t.Run("returns error on buildLaunchArgs failure", func(t *testing.T) {
		provider := &mockCredentialProvider{
			agentType: &agent.AgentType{
				ID:            1,
				Slug:          "claude-code",
				LaunchCommand: "claude",
				CommandTemplate: agent.CommandTemplate{
					Args: []agent.ArgRule{
						{Args: []string{"--model", "{{.invalid"}},
					},
				},
			},
			credentials: agent.EncryptedCredentials{},
			isRunner:    false,
		}

		builder := NewConfigBuilder(provider)
		req := &ConfigBuildRequest{
			AgentTypeID: 1,
			UserID:      1,
		}

		_, err := builder.BuildPodCommand(ctx, req)
		if err == nil {
			t.Error("Expected error for invalid template")
		}
		if !strings.Contains(err.Error(), "failed to build launch args") {
			t.Errorf("Error should contain 'failed to build launch args', got: %v", err)
		}
	})

	t.Run("returns error on buildFilesToCreate failure", func(t *testing.T) {
		provider := &mockCredentialProvider{
			agentType: &agent.AgentType{
				ID:            1,
				Slug:          "claude-code",
				LaunchCommand: "claude",
				FilesTemplate: agent.FilesTemplate{
					{
						PathTemplate:    "/tmp/test.txt",
						ContentTemplate: "{{.invalid",
					},
				},
			},
			credentials: agent.EncryptedCredentials{},
			isRunner:    false,
		}

		builder := NewConfigBuilder(provider)
		req := &ConfigBuildRequest{
			AgentTypeID: 1,
			UserID:      1,
		}

		_, err := builder.BuildPodCommand(ctx, req)
		if err == nil {
			t.Error("Expected error for invalid file template")
		}
		if !strings.Contains(err.Error(), "failed to build files to create") {
			t.Errorf("Error should contain 'failed to build files to create', got: %v", err)
		}
	})
}

func TestConfigBuilder_BuildPodCommand_FullFlow(t *testing.T) {
	ctx := context.Background()

	t.Run("full flow with credentials and files", func(t *testing.T) {
		provider := &mockCredentialProvider{
			agentType: &agent.AgentType{
				ID:            1,
				Slug:          "claude-code",
				Name:          "Claude Code",
				LaunchCommand: "claude",
				ConfigSchema: agent.ConfigSchema{
					Fields: []agent.ConfigField{
						{Name: "model", Type: "select", Default: "opus"},
					},
				},
				CommandTemplate: agent.CommandTemplate{
					Args: []agent.ArgRule{
						{Args: []string{"--model", "{{.config.model}}"}},
					},
				},
				FilesTemplate: agent.FilesTemplate{
					{
						PathTemplate:    "{{.sandbox.root_path}}/config.json",
						ContentTemplate: `{"model":"{{.config.model}}"}`,
						Mode:            0600,
					},
				},
				CredentialSchema: agent.CredentialSchema{
					{Name: "api_key", Type: "secret", EnvVar: "ANTHROPIC_API_KEY", Required: true},
				},
			},
			credentials: agent.EncryptedCredentials{
				"api_key": "sk-ant-test-key",
			},
			isRunner: false,
		}

		builder := NewConfigBuilder(provider)
		req := &ConfigBuildRequest{
			AgentTypeID:     1,
			UserID:          1,
			OrganizationID:  1,
			MCPPort:         19000,
			PodKey:          "test-pod-full",
			InitialPrompt:   "Hello",
			ConfigOverrides: map[string]interface{}{"model": "sonnet"},
		}

		cmd, err := builder.BuildPodCommand(ctx, req)
		if err != nil {
			t.Fatalf("BuildPodCommand failed: %v", err)
		}

		if cmd.LaunchCommand != "claude" {
			t.Errorf("LaunchCommand = %s, want claude", cmd.LaunchCommand)
		}

		found := false
		for i, arg := range cmd.LaunchArgs {
			if arg == "--model" && i+1 < len(cmd.LaunchArgs) && cmd.LaunchArgs[i+1] == "sonnet" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("LaunchArgs should contain --model sonnet, got %v", cmd.LaunchArgs)
		}

		if cmd.EnvVars["ANTHROPIC_API_KEY"] != "sk-ant-test-key" {
			t.Errorf("EnvVars[ANTHROPIC_API_KEY] = %s, want sk-ant-test-key", cmd.EnvVars["ANTHROPIC_API_KEY"])
		}

		// Claude Code builder generates: 1 base template file + 4 plugin directory entries
		// (agentsmesh-plugin/, .claude-plugin/plugin.json, .mcp.json, skills/)
		if len(cmd.FilesToCreate) < 1 {
			t.Fatalf("FilesToCreate count = %d, want >= 1", len(cmd.FilesToCreate))
		}
		if cmd.FilesToCreate[0].Mode != 0600 {
			t.Errorf("FilesToCreate[0].Mode = %o, want 0600", cmd.FilesToCreate[0].Mode)
		}

		// InitialPrompt is now prepended to LaunchArgs as the first argument
		if len(cmd.LaunchArgs) == 0 || cmd.LaunchArgs[0] != "Hello" {
			t.Errorf("LaunchArgs[0] = %v, want Hello (InitialPrompt should be first arg)", cmd.LaunchArgs)
		}
	})
}
