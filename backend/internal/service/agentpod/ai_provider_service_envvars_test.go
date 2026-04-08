package agentpod

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

func TestFormatEnvVars(t *testing.T) {
	service := newTestAIProviderService(nil, nil)

	t.Run("claude credentials", func(t *testing.T) {
		creds := map[string]string{
			"api_key":    "sk-test-key",
			"base_url":   "https://api.anthropic.com",
			"auth_token": "auth-123",
		}

		envVars := service.formatEnvVars(agentpod.AIProviderTypeClaude, creds)

		if envVars["ANTHROPIC_API_KEY"] != "sk-test-key" {
			t.Errorf("expected ANTHROPIC_API_KEY 'sk-test-key', got '%s'", envVars["ANTHROPIC_API_KEY"])
		}
		if envVars["ANTHROPIC_BASE_URL"] != "https://api.anthropic.com" {
			t.Errorf("expected ANTHROPIC_BASE_URL, got '%s'", envVars["ANTHROPIC_BASE_URL"])
		}
		if envVars["ANTHROPIC_AUTH_TOKEN"] != "auth-123" {
			t.Errorf("expected ANTHROPIC_AUTH_TOKEN 'auth-123', got '%s'", envVars["ANTHROPIC_AUTH_TOKEN"])
		}
	})

	t.Run("openai credentials", func(t *testing.T) {
		creds := map[string]string{
			"api_key":      "sk-openai-key",
			"organization": "org-123",
		}

		envVars := service.formatEnvVars(agentpod.AIProviderTypeOpenAI, creds)

		if envVars["OPENAI_API_KEY"] != "sk-openai-key" {
			t.Errorf("expected OPENAI_API_KEY 'sk-openai-key', got '%s'", envVars["OPENAI_API_KEY"])
		}
		if envVars["OPENAI_ORG_ID"] != "org-123" {
			t.Errorf("expected OPENAI_ORG_ID 'org-123', got '%s'", envVars["OPENAI_ORG_ID"])
		}
	})

	t.Run("unknown provider type", func(t *testing.T) {
		creds := map[string]string{"api_key": "test"}
		envVars := service.formatEnvVars("unknown", creds)

		if len(envVars) != 0 {
			t.Errorf("expected empty env vars for unknown provider, got %d", len(envVars))
		}
	})
}

func TestFormatEnvVars_Gemini(t *testing.T) {
	service := newTestAIProviderService(nil, nil)

	creds := map[string]string{
		"api_key": "gemini-test-key",
	}

	envVars := service.formatEnvVars(agentpod.AIProviderTypeGemini, creds)

	if envVars["GOOGLE_API_KEY"] != "gemini-test-key" {
		t.Errorf("expected GOOGLE_API_KEY 'gemini-test-key', got '%s'", envVars["GOOGLE_API_KEY"])
	}
}

func TestGetAIProviderEnvVars(t *testing.T) {
	db := setupTestDB(t)
	service := newTestAIProviderService(db, nil)
	ctx := context.Background()

	// Create a default provider
	creds := map[string]string{"api_key": "env-test-key"}
	_, err := service.CreateUserProvider(ctx, 1, agentpod.AIProviderTypeClaude, "Default Claude", creds, true)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	// Get env vars
	envVars, err := service.GetAIProviderEnvVars(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get env vars: %v", err)
	}

	if envVars["ANTHROPIC_API_KEY"] != "env-test-key" {
		t.Errorf("expected ANTHROPIC_API_KEY 'env-test-key', got '%s'", envVars["ANTHROPIC_API_KEY"])
	}
}

func TestGetAIProviderEnvVars_NoDefaultProvider(t *testing.T) {
	db := setupTestDB(t)
	service := newTestAIProviderService(db, nil)
	ctx := context.Background()

	// When no default provider exists, should return nil, nil (not error)
	envVars, err := service.GetAIProviderEnvVars(ctx, 999)
	if err != nil {
		t.Errorf("expected nil error for missing default provider, got %v", err)
	}
	if envVars != nil {
		t.Errorf("expected nil envVars for missing default provider, got %v", envVars)
	}
}

func TestGetAIProviderEnvVarsByID(t *testing.T) {
	db := setupTestDB(t)
	service := newTestAIProviderService(db, nil)
	ctx := context.Background()

	// Create a provider
	creds := map[string]string{"api_key": "by-id-key"}
	provider, err := service.CreateUserProvider(ctx, 1, agentpod.AIProviderTypeClaude, "Claude", creds, true)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	// Get env vars by ID
	envVars, err := service.GetAIProviderEnvVarsByID(ctx, provider.ID)
	if err != nil {
		t.Fatalf("failed to get env vars by ID: %v", err)
	}

	if envVars["ANTHROPIC_API_KEY"] != "by-id-key" {
		t.Errorf("expected ANTHROPIC_API_KEY 'by-id-key', got '%s'", envVars["ANTHROPIC_API_KEY"])
	}
}

func TestGetAIProviderEnvVarsByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	service := newTestAIProviderService(db, nil)
	ctx := context.Background()

	_, err := service.GetAIProviderEnvVarsByID(ctx, 999)
	if err != ErrProviderNotFound {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}
