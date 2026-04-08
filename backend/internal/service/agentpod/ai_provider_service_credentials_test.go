package agentpod

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

func TestGetProviderCredentials(t *testing.T) {
	db := setupTestDB(t)
	service := newTestAIProviderService(db, nil)
	ctx := context.Background()

	// Create a provider
	creds := map[string]string{
		"api_key":  "sk-test-key",
		"base_url": "https://api.example.com",
	}
	provider, err := service.CreateUserProvider(ctx, 1, agentpod.AIProviderTypeClaude, "Claude", creds, true)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	// Get credentials
	retrieved, err := service.GetProviderCredentials(ctx, provider.ID)
	if err != nil {
		t.Fatalf("failed to get credentials: %v", err)
	}

	if retrieved["api_key"] != "sk-test-key" {
		t.Errorf("expected api_key 'sk-test-key', got '%s'", retrieved["api_key"])
	}
	if retrieved["base_url"] != "https://api.example.com" {
		t.Errorf("expected base_url 'https://api.example.com', got '%s'", retrieved["base_url"])
	}
}

func TestGetProviderCredentials_NotFound(t *testing.T) {
	db := setupTestDB(t)
	service := newTestAIProviderService(db, nil)
	ctx := context.Background()

	_, err := service.GetProviderCredentials(ctx, 999)
	if err != ErrProviderNotFound {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestGetUserDefaultCredentials(t *testing.T) {
	db := setupTestDB(t)
	service := newTestAIProviderService(db, nil)
	ctx := context.Background()

	// Create providers
	creds1 := map[string]string{"api_key": "key-1"}
	creds2 := map[string]string{"api_key": "key-2"}
	_, err := service.CreateUserProvider(ctx, 1, agentpod.AIProviderTypeClaude, "Claude 1", creds1, false)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	_, err = service.CreateUserProvider(ctx, 1, agentpod.AIProviderTypeClaude, "Claude 2", creds2, true)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	// Get default credentials
	creds, err := service.GetUserDefaultCredentials(ctx, 1, agentpod.AIProviderTypeClaude)
	if err != nil {
		t.Fatalf("failed to get default credentials: %v", err)
	}

	if creds["api_key"] != "key-2" {
		t.Errorf("expected default credentials api_key 'key-2', got '%s'", creds["api_key"])
	}
}

func TestGetUserDefaultCredentials_NotFound(t *testing.T) {
	db := setupTestDB(t)
	service := newTestAIProviderService(db, nil)
	ctx := context.Background()

	_, err := service.GetUserDefaultCredentials(ctx, 1, agentpod.AIProviderTypeClaude)
	if err != ErrProviderNotFound {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestValidateCredentials(t *testing.T) {
	service := newTestAIProviderService(nil, nil)

	tests := []struct {
		name         string
		providerType string
		credentials  map[string]string
		expectError  bool
	}{
		{
			name:         "claude with api_key",
			providerType: agentpod.AIProviderTypeClaude,
			credentials:  map[string]string{"api_key": "sk-test"},
			expectError:  false,
		},
		{
			name:         "claude with auth_token",
			providerType: agentpod.AIProviderTypeClaude,
			credentials:  map[string]string{"auth_token": "token-123"},
			expectError:  false,
		},
		{
			name:         "claude without credentials",
			providerType: agentpod.AIProviderTypeClaude,
			credentials:  map[string]string{},
			expectError:  true,
		},
		{
			name:         "openai with api_key",
			providerType: agentpod.AIProviderTypeOpenAI,
			credentials:  map[string]string{"api_key": "sk-test"},
			expectError:  false,
		},
		{
			name:         "openai without api_key",
			providerType: agentpod.AIProviderTypeOpenAI,
			credentials:  map[string]string{},
			expectError:  true,
		},
		{
			name:         "gemini with api_key",
			providerType: agentpod.AIProviderTypeGemini,
			credentials:  map[string]string{"api_key": "test-key"},
			expectError:  false,
		},
		{
			name:         "gemini without api_key",
			providerType: agentpod.AIProviderTypeGemini,
			credentials:  map[string]string{},
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateCredentials(tt.providerType, tt.credentials)
			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestDecryptCredentials_EmptyString(t *testing.T) {
	service := newTestAIProviderService(nil, nil)

	_, err := service.decryptCredentials("")
	if err != ErrCredentialsNotFound {
		t.Errorf("expected ErrCredentialsNotFound, got %v", err)
	}
}

func TestDecryptCredentials_InvalidJSON(t *testing.T) {
	service := newTestAIProviderService(nil, nil)

	_, err := service.decryptCredentials("not-json")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestEncryptDecryptCredentials_DevMode(t *testing.T) {
	service := newTestAIProviderService(nil, nil) // nil encryptor = dev mode
	ctx := context.Background()

	creds := map[string]string{
		"api_key":  "secret-key",
		"base_url": "https://example.com",
	}

	// Encrypt
	encrypted, err := service.encryptCredentials(creds)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	// In dev mode, encrypted should be plain JSON
	var decoded map[string]string
	if err := json.Unmarshal([]byte(encrypted), &decoded); err != nil {
		t.Fatalf("expected valid JSON in dev mode: %v", err)
	}

	// Decrypt
	decrypted, err := service.decryptCredentials(encrypted)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}

	if decrypted["api_key"] != "secret-key" {
		t.Errorf("expected api_key 'secret-key', got '%s'", decrypted["api_key"])
	}

	// Suppress unused ctx warning
	_ = ctx
}
