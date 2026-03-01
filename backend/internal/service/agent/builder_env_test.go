package agent

import (
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/agent"
)

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
