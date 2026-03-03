package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

// mockCredentialProvider implements AgentConfigProvider for testing buildEnvVars paths
type mockCredentialProvider struct {
	agentType   *agent.AgentType
	credentials agent.EncryptedCredentials
	isRunner    bool
	credErr     error
}

func (m *mockCredentialProvider) GetAgentType(ctx context.Context, id int64) (*agent.AgentType, error) {
	if m.agentType == nil {
		return nil, fmt.Errorf("agent type not found")
	}
	return m.agentType, nil
}

func (m *mockCredentialProvider) GetUserEffectiveConfig(ctx context.Context, userID, agentTypeID int64, overrides agent.ConfigValues) agent.ConfigValues {
	result := make(agent.ConfigValues)
	for k, v := range overrides {
		result[k] = v
	}
	return result
}

func (m *mockCredentialProvider) GetEffectiveCredentialsForPod(ctx context.Context, userID, agentTypeID int64, profileID *int64) (agent.EncryptedCredentials, bool, error) {
	return m.credentials, m.isRunner, m.credErr
}

func TestConfigBuilder_buildEnvVars_WithCredentials(t *testing.T) {
	ctx := context.Background()

	t.Run("injects credentials as env vars", func(t *testing.T) {
		provider := &mockCredentialProvider{
			agentType: &agent.AgentType{
				ID:            1,
				Slug:          "claude-code",
				LaunchCommand: "claude",
				CredentialSchema: agent.CredentialSchema{
					{Name: "api_key", Type: "secret", EnvVar: "ANTHROPIC_API_KEY", Required: true},
					{Name: "base_url", Type: "string", EnvVar: "ANTHROPIC_BASE_URL", Required: false},
				},
			},
			credentials: agent.EncryptedCredentials{
				"api_key":  "sk-ant-test-key",
				"base_url": "https://api.anthropic.com",
			},
			isRunner: false,
		}

		builder := NewConfigBuilder(provider)
		req := &ConfigBuildRequest{
			AgentTypeID: 1,
			UserID:      1,
		}

		envVars, err := builder.buildEnvVars(ctx, req, provider.agentType)
		if err != nil {
			t.Fatalf("buildEnvVars failed: %v", err)
		}

		if envVars["ANTHROPIC_API_KEY"] != "sk-ant-test-key" {
			t.Errorf("ANTHROPIC_API_KEY = %s, want sk-ant-test-key", envVars["ANTHROPIC_API_KEY"])
		}
		if envVars["ANTHROPIC_BASE_URL"] != "https://api.anthropic.com" {
			t.Errorf("ANTHROPIC_BASE_URL = %s, want https://api.anthropic.com", envVars["ANTHROPIC_BASE_URL"])
		}
	})

	t.Run("skips empty credential values", func(t *testing.T) {
		provider := &mockCredentialProvider{
			agentType: &agent.AgentType{
				ID:            1,
				Slug:          "claude-code",
				LaunchCommand: "claude",
				CredentialSchema: agent.CredentialSchema{
					{Name: "api_key", Type: "secret", EnvVar: "ANTHROPIC_API_KEY", Required: true},
					{Name: "base_url", Type: "string", EnvVar: "ANTHROPIC_BASE_URL", Required: false},
				},
			},
			credentials: agent.EncryptedCredentials{
				"api_key":  "sk-ant-test-key",
				"base_url": "",
			},
			isRunner: false,
		}

		builder := NewConfigBuilder(provider)
		req := &ConfigBuildRequest{
			AgentTypeID: 1,
			UserID:      1,
		}

		envVars, err := builder.buildEnvVars(ctx, req, provider.agentType)
		if err != nil {
			t.Fatalf("buildEnvVars failed: %v", err)
		}

		if envVars["ANTHROPIC_API_KEY"] != "sk-ant-test-key" {
			t.Errorf("ANTHROPIC_API_KEY = %s, want sk-ant-test-key", envVars["ANTHROPIC_API_KEY"])
		}
		if _, exists := envVars["ANTHROPIC_BASE_URL"]; exists {
			t.Error("ANTHROPIC_BASE_URL should not be set for empty value")
		}
	})

	t.Run("returns empty envVars for runner host mode", func(t *testing.T) {
		provider := &mockCredentialProvider{
			agentType: &agent.AgentType{
				ID:            1,
				Slug:          "claude-code",
				LaunchCommand: "claude",
				CredentialSchema: agent.CredentialSchema{
					{Name: "api_key", Type: "secret", EnvVar: "ANTHROPIC_API_KEY", Required: true},
				},
			},
			credentials: agent.EncryptedCredentials{
				"api_key": "sk-ant-test-key",
			},
			isRunner: true,
		}

		builder := NewConfigBuilder(provider)
		req := &ConfigBuildRequest{
			AgentTypeID: 1,
			UserID:      1,
		}

		envVars, err := builder.buildEnvVars(ctx, req, provider.agentType)
		if err != nil {
			t.Fatalf("buildEnvVars failed: %v", err)
		}

		if len(envVars) != 0 {
			t.Errorf("envVars should be empty for runner host mode, got %v", envVars)
		}
	})

	t.Run("injects auth_token as env var", func(t *testing.T) {
		provider := &mockCredentialProvider{
			agentType: &agent.AgentType{
				ID:            1,
				Slug:          "claude-code",
				LaunchCommand: "claude",
				CredentialSchema: agent.CredentialSchema{
					{Name: "api_key", Type: "secret", EnvVar: "ANTHROPIC_API_KEY", Required: false},
					{Name: "auth_token", Type: "secret", EnvVar: "ANTHROPIC_AUTH_TOKEN", Required: false},
					{Name: "base_url", Type: "text", EnvVar: "ANTHROPIC_BASE_URL", Required: false},
				},
			},
			credentials: agent.EncryptedCredentials{
				"auth_token": "token-test-123",
				"base_url":   "https://custom.proxy.example.com",
			},
			isRunner: false,
		}

		builder := NewConfigBuilder(provider)
		req := &ConfigBuildRequest{
			AgentTypeID: 1,
			UserID:      1,
		}

		envVars, err := builder.buildEnvVars(ctx, req, provider.agentType)
		if err != nil {
			t.Fatalf("buildEnvVars failed: %v", err)
		}

		if envVars["ANTHROPIC_AUTH_TOKEN"] != "token-test-123" {
			t.Errorf("ANTHROPIC_AUTH_TOKEN = %s, want token-test-123", envVars["ANTHROPIC_AUTH_TOKEN"])
		}
		if envVars["ANTHROPIC_BASE_URL"] != "https://custom.proxy.example.com" {
			t.Errorf("ANTHROPIC_BASE_URL = %s, want https://custom.proxy.example.com", envVars["ANTHROPIC_BASE_URL"])
		}
		if _, exists := envVars["ANTHROPIC_API_KEY"]; exists {
			t.Error("ANTHROPIC_API_KEY should not be set when only auth_token is provided")
		}
	})

	t.Run("returns error on credential fetch failure", func(t *testing.T) {
		provider := &mockCredentialProvider{
			agentType: &agent.AgentType{
				ID:            1,
				Slug:          "claude-code",
				LaunchCommand: "claude",
			},
			credErr: fmt.Errorf("database error"),
		}

		builder := NewConfigBuilder(provider)
		req := &ConfigBuildRequest{
			AgentTypeID: 1,
			UserID:      1,
		}

		_, err := builder.buildEnvVars(ctx, req, provider.agentType)
		if err == nil {
			t.Error("Expected error for credential fetch failure")
		}
	})
}
