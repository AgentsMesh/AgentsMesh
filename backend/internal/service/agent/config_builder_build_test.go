package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/anthropics/agentmesh/backend/internal/domain/agent"
)

func TestConfigBuilder_BuildPodConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("basic pod config", func(t *testing.T) {
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

		config, err := builder.BuildPodConfig(ctx, req)
		if err != nil {
			t.Fatalf("BuildPodConfig failed: %v", err)
		}

		if config.LaunchCommand != "claude" {
			t.Errorf("LaunchCommand = %s, want claude", config.LaunchCommand)
		}
		if config.WorkDirConfig.Type != "tempdir" {
			t.Errorf("WorkDirConfig.Type = %s, want tempdir", config.WorkDirConfig.Type)
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
			Branch:         "main",
			MCPPort:        19000,
			PodKey:         "test-pod-456",
		}

		config, err := builder.BuildPodConfig(ctx, req)
		if err != nil {
			t.Fatalf("BuildPodConfig failed: %v", err)
		}

		if config.WorkDirConfig.Type != "worktree" {
			t.Errorf("WorkDirConfig.Type = %s, want worktree", config.WorkDirConfig.Type)
		}
		if config.WorkDirConfig.RepositoryURL != "https://github.com/test/repo.git" {
			t.Errorf("RepositoryURL = %s, want https://github.com/test/repo.git", config.WorkDirConfig.RepositoryURL)
		}
		if config.WorkDirConfig.Branch != "main" {
			t.Errorf("Branch = %s, want main", config.WorkDirConfig.Branch)
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

		config, err := builder.BuildPodConfig(ctx, req)
		if err != nil {
			t.Fatalf("BuildPodConfig failed: %v", err)
		}

		if config.WorkDirConfig.Type != "local" {
			t.Errorf("WorkDirConfig.Type = %s, want local", config.WorkDirConfig.Type)
		}
		if config.WorkDirConfig.LocalPath != "/home/user/project" {
			t.Errorf("LocalPath = %s, want /home/user/project", config.WorkDirConfig.LocalPath)
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

		config, err := builder.BuildPodConfig(ctx, req)
		if err != nil {
			t.Fatalf("BuildPodConfig failed: %v", err)
		}

		found := false
		for i, arg := range config.LaunchArgs {
			if arg == "--model" && i+1 < len(config.LaunchArgs) && config.LaunchArgs[i+1] == "sonnet" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("LaunchArgs should contain --model sonnet, got %v", config.LaunchArgs)
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

		config, err := builder.BuildPodConfig(ctx, req)
		if err != nil {
			t.Fatalf("BuildPodConfig failed: %v", err)
		}

		if config.InitialPrompt != "Fix the bug in main.go" {
			t.Errorf("InitialPrompt = %s, want Fix the bug in main.go", config.InitialPrompt)
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

		_, err := builder.BuildPodConfig(ctx, req)
		if err == nil {
			t.Error("Expected error for invalid agent type")
		}
	})
}

func TestConfigBuilder_BuildPodConfig_ErrorPaths(t *testing.T) {
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

		_, err := builder.BuildPodConfig(ctx, req)
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

		_, err := builder.BuildPodConfig(ctx, req)
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

		_, err := builder.BuildPodConfig(ctx, req)
		if err == nil {
			t.Error("Expected error for invalid file template")
		}
		if !strings.Contains(err.Error(), "failed to build files to create") {
			t.Errorf("Error should contain 'failed to build files to create', got: %v", err)
		}
	})
}

func TestConfigBuilder_BuildPodConfig_FullFlow(t *testing.T) {
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

		config, err := builder.BuildPodConfig(ctx, req)
		if err != nil {
			t.Fatalf("BuildPodConfig failed: %v", err)
		}

		if config.LaunchCommand != "claude" {
			t.Errorf("LaunchCommand = %s, want claude", config.LaunchCommand)
		}

		found := false
		for i, arg := range config.LaunchArgs {
			if arg == "--model" && i+1 < len(config.LaunchArgs) && config.LaunchArgs[i+1] == "sonnet" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("LaunchArgs should contain --model sonnet, got %v", config.LaunchArgs)
		}

		if config.EnvVars["ANTHROPIC_API_KEY"] != "sk-ant-test-key" {
			t.Errorf("EnvVars[ANTHROPIC_API_KEY] = %s, want sk-ant-test-key", config.EnvVars["ANTHROPIC_API_KEY"])
		}

		if len(config.FilesToCreate) != 1 {
			t.Fatalf("FilesToCreate count = %d, want 1", len(config.FilesToCreate))
		}
		if config.FilesToCreate[0].Mode != 0600 {
			t.Errorf("FilesToCreate[0].Mode = %o, want 0600", config.FilesToCreate[0].Mode)
		}

		if config.InitialPrompt != "Hello" {
			t.Errorf("InitialPrompt = %s, want Hello", config.InitialPrompt)
		}
	})
}
