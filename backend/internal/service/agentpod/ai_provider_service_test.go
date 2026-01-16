package agentpod

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAIProviderTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Create tables manually for SQLite compatibility
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_ai_providers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			provider_type TEXT NOT NULL,
			name TEXT NOT NULL,
			is_default INTEGER NOT NULL DEFAULT 0,
			is_enabled INTEGER NOT NULL DEFAULT 1,
			encrypted_credentials TEXT NOT NULL,
			last_used_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create user_ai_providers table: %v", err)
	}

	return db
}

func TestNewAIProviderService(t *testing.T) {
	db := setupAIProviderTestDB(t)
	service := NewAIProviderService(db, nil) // nil encryptor for development mode

	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if service.db != db {
		t.Fatal("expected service.db to be the provided db")
	}
}

func TestCreateUserProvider(t *testing.T) {
	db := setupAIProviderTestDB(t)
	service := NewAIProviderService(db, nil)
	ctx := context.Background()

	credentials := map[string]string{
		"api_key": "sk-test-key-123",
	}

	provider, err := service.CreateUserProvider(ctx, 1, agentpod.AIProviderTypeClaude, "My Claude", credentials, true)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
	if provider.UserID != 1 {
		t.Errorf("expected UserID 1, got %d", provider.UserID)
	}
	if provider.ProviderType != agentpod.AIProviderTypeClaude {
		t.Errorf("expected ProviderType '%s', got '%s'", agentpod.AIProviderTypeClaude, provider.ProviderType)
	}
	if provider.Name != "My Claude" {
		t.Errorf("expected Name 'My Claude', got '%s'", provider.Name)
	}
	if !provider.IsDefault {
		t.Error("expected IsDefault to be true")
	}
	if !provider.IsEnabled {
		t.Error("expected IsEnabled to be true")
	}
}

func TestGetUserProviders(t *testing.T) {
	db := setupAIProviderTestDB(t)
	service := NewAIProviderService(db, nil)
	ctx := context.Background()

	// Create multiple providers
	creds := map[string]string{"api_key": "test"}
	_, err := service.CreateUserProvider(ctx, 1, agentpod.AIProviderTypeClaude, "Claude 1", creds, true)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	_, err = service.CreateUserProvider(ctx, 1, agentpod.AIProviderTypeOpenAI, "OpenAI", creds, false)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	_, err = service.CreateUserProvider(ctx, 2, agentpod.AIProviderTypeClaude, "Other User", creds, true)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	providers, err := service.GetUserProviders(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get providers: %v", err)
	}

	if len(providers) != 2 {
		t.Errorf("expected 2 providers for user 1, got %d", len(providers))
	}
}

func TestGetUserProvidersByType(t *testing.T) {
	db := setupAIProviderTestDB(t)
	service := NewAIProviderService(db, nil)
	ctx := context.Background()

	// Create multiple providers
	creds := map[string]string{"api_key": "test"}
	_, err := service.CreateUserProvider(ctx, 1, agentpod.AIProviderTypeClaude, "Claude 1", creds, true)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	_, err = service.CreateUserProvider(ctx, 1, agentpod.AIProviderTypeClaude, "Claude 2", creds, false)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	_, err = service.CreateUserProvider(ctx, 1, agentpod.AIProviderTypeOpenAI, "OpenAI", creds, false)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	providers, err := service.GetUserProvidersByType(ctx, 1, agentpod.AIProviderTypeClaude)
	if err != nil {
		t.Fatalf("failed to get providers: %v", err)
	}

	if len(providers) != 2 {
		t.Errorf("expected 2 Claude providers, got %d", len(providers))
	}
	for _, p := range providers {
		if p.ProviderType != agentpod.AIProviderTypeClaude {
			t.Errorf("expected only Claude providers")
		}
	}
}

func TestUpdateUserProvider(t *testing.T) {
	db := setupAIProviderTestDB(t)
	service := NewAIProviderService(db, nil)
	ctx := context.Background()

	// Create a provider
	creds := map[string]string{"api_key": "old-key"}
	provider, err := service.CreateUserProvider(ctx, 1, agentpod.AIProviderTypeClaude, "Original", creds, false)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	// Update provider
	newCreds := map[string]string{"api_key": "new-key"}
	updated, err := service.UpdateUserProvider(ctx, provider.ID, "Updated Name", newCreds, true, true)
	if err != nil {
		t.Fatalf("failed to update provider: %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("expected Name 'Updated Name', got '%s'", updated.Name)
	}
	if !updated.IsDefault {
		t.Error("expected IsDefault to be true")
	}
}

func TestUpdateUserProvider_NotFound(t *testing.T) {
	db := setupAIProviderTestDB(t)
	service := NewAIProviderService(db, nil)
	ctx := context.Background()

	_, err := service.UpdateUserProvider(ctx, 999, "Name", nil, false, true)
	if err != ErrProviderNotFound {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestDeleteUserProvider(t *testing.T) {
	db := setupAIProviderTestDB(t)
	service := NewAIProviderService(db, nil)
	ctx := context.Background()

	// Create a provider
	creds := map[string]string{"api_key": "test"}
	provider, err := service.CreateUserProvider(ctx, 1, agentpod.AIProviderTypeClaude, "To Delete", creds, false)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	// Delete provider
	err = service.DeleteUserProvider(ctx, provider.ID)
	if err != nil {
		t.Fatalf("failed to delete provider: %v", err)
	}

	// Verify deleted
	providers, _ := service.GetUserProviders(ctx, 1)
	if len(providers) != 0 {
		t.Errorf("expected 0 providers after delete, got %d", len(providers))
	}
}

func TestSetDefaultProvider(t *testing.T) {
	db := setupAIProviderTestDB(t)
	service := NewAIProviderService(db, nil)
	ctx := context.Background()

	// Create two providers of same type
	creds := map[string]string{"api_key": "test"}
	p1, err := service.CreateUserProvider(ctx, 1, agentpod.AIProviderTypeClaude, "Claude 1", creds, true)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	p2, err := service.CreateUserProvider(ctx, 1, agentpod.AIProviderTypeClaude, "Claude 2", creds, false)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	// Set second as default
	err = service.SetDefaultProvider(ctx, p2.ID)
	if err != nil {
		t.Fatalf("failed to set default provider: %v", err)
	}

	// Verify first is no longer default
	var provider1 agentpod.UserAIProvider
	db.First(&provider1, p1.ID)
	if provider1.IsDefault {
		t.Error("expected first provider to no longer be default")
	}

	// Verify second is now default
	var provider2 agentpod.UserAIProvider
	db.First(&provider2, p2.ID)
	if !provider2.IsDefault {
		t.Error("expected second provider to be default")
	}
}

func TestSetDefaultProvider_NotFound(t *testing.T) {
	db := setupAIProviderTestDB(t)
	service := NewAIProviderService(db, nil)
	ctx := context.Background()

	err := service.SetDefaultProvider(ctx, 999)
	if err != ErrProviderNotFound {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestGetProviderCredentials(t *testing.T) {
	db := setupAIProviderTestDB(t)
	service := NewAIProviderService(db, nil)
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

func TestGetUserDefaultCredentials(t *testing.T) {
	db := setupAIProviderTestDB(t)
	service := NewAIProviderService(db, nil)
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
	db := setupAIProviderTestDB(t)
	service := NewAIProviderService(db, nil)
	ctx := context.Background()

	_, err := service.GetUserDefaultCredentials(ctx, 1, agentpod.AIProviderTypeClaude)
	if err != ErrProviderNotFound {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestValidateCredentials(t *testing.T) {
	service := NewAIProviderService(nil, nil)

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

func TestFormatEnvVars(t *testing.T) {
	service := NewAIProviderService(nil, nil)

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

func TestDecryptCredentials_EmptyString(t *testing.T) {
	service := NewAIProviderService(nil, nil)

	_, err := service.decryptCredentials("")
	if err != ErrCredentialsNotFound {
		t.Errorf("expected ErrCredentialsNotFound, got %v", err)
	}
}

func TestDecryptCredentials_InvalidJSON(t *testing.T) {
	service := NewAIProviderService(nil, nil)

	_, err := service.decryptCredentials("not-json")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestEncryptDecryptCredentials_DevMode(t *testing.T) {
	service := NewAIProviderService(nil, nil) // nil encryptor = dev mode
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

func TestGetAIProviderEnvVars(t *testing.T) {
	db := setupAIProviderTestDB(t)
	service := NewAIProviderService(db, nil)
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
	db := setupAIProviderTestDB(t)
	service := NewAIProviderService(db, nil)
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
	db := setupAIProviderTestDB(t)
	service := NewAIProviderService(db, nil)
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
	db := setupAIProviderTestDB(t)
	service := NewAIProviderService(db, nil)
	ctx := context.Background()

	_, err := service.GetAIProviderEnvVarsByID(ctx, 999)
	if err != ErrProviderNotFound {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestGetProviderCredentials_NotFound(t *testing.T) {
	db := setupAIProviderTestDB(t)
	service := NewAIProviderService(db, nil)
	ctx := context.Background()

	_, err := service.GetProviderCredentials(ctx, 999)
	if err != ErrProviderNotFound {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestUpdateUserProvider_WithCredentials(t *testing.T) {
	db := setupAIProviderTestDB(t)
	service := NewAIProviderService(db, nil)
	ctx := context.Background()

	// Create a provider
	creds := map[string]string{"api_key": "old-key"}
	provider, _ := service.CreateUserProvider(ctx, 1, agentpod.AIProviderTypeClaude, "Claude", creds, false)

	// Update with new credentials
	newCreds := map[string]string{"api_key": "new-updated-key"}
	updated, err := service.UpdateUserProvider(ctx, provider.ID, "Updated", newCreds, false, true)
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	// Verify credentials updated
	retrievedCreds, _ := service.GetProviderCredentials(ctx, updated.ID)
	if retrievedCreds["api_key"] != "new-updated-key" {
		t.Errorf("expected api_key 'new-updated-key', got '%s'", retrievedCreds["api_key"])
	}
}

func TestCreateUserProvider_EmptyCredentials(t *testing.T) {
	db := setupAIProviderTestDB(t)
	service := NewAIProviderService(db, nil)
	ctx := context.Background()

	// Create with empty credentials - should succeed (validation happens at API layer)
	creds := map[string]string{}
	provider, err := service.CreateUserProvider(ctx, 1, agentpod.AIProviderTypeClaude, "Claude", creds, true)
	if err != nil {
		t.Errorf("unexpected error for empty credentials: %v", err)
	}
	if provider == nil {
		t.Error("expected provider to be created")
	}
}

func TestFormatEnvVars_Gemini(t *testing.T) {
	service := NewAIProviderService(nil, nil)

	creds := map[string]string{
		"api_key": "gemini-test-key",
	}

	envVars := service.formatEnvVars(agentpod.AIProviderTypeGemini, creds)

	if envVars["GOOGLE_API_KEY"] != "gemini-test-key" {
		t.Errorf("expected GOOGLE_API_KEY 'gemini-test-key', got '%s'", envVars["GOOGLE_API_KEY"])
	}
}

func TestCreateUserProvider_SetsDefault(t *testing.T) {
	db := setupAIProviderTestDB(t)
	service := NewAIProviderService(db, nil)
	ctx := context.Background()

	// Create first default provider
	creds := map[string]string{"api_key": "key1"}
	p1, _ := service.CreateUserProvider(ctx, 1, agentpod.AIProviderTypeClaude, "Claude 1", creds, true)

	// Create second default provider of same type
	p2, _ := service.CreateUserProvider(ctx, 1, agentpod.AIProviderTypeClaude, "Claude 2", creds, true)

	// First should no longer be default
	var provider1 agentpod.UserAIProvider
	db.First(&provider1, p1.ID)
	if provider1.IsDefault {
		t.Error("expected first provider to not be default")
	}

	// Second should be default
	var provider2 agentpod.UserAIProvider
	db.First(&provider2, p2.ID)
	if !provider2.IsDefault {
		t.Error("expected second provider to be default")
	}
}
