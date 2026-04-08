package agentpod

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

func TestCreateUserProvider(t *testing.T) {
	db := setupTestDB(t)
	service := newTestAIProviderService(db, nil)
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
	db := setupTestDB(t)
	service := newTestAIProviderService(db, nil)
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
	db := setupTestDB(t)
	service := newTestAIProviderService(db, nil)
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
	db := setupTestDB(t)
	service := newTestAIProviderService(db, nil)
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
	db := setupTestDB(t)
	service := newTestAIProviderService(db, nil)
	ctx := context.Background()

	_, err := service.UpdateUserProvider(ctx, 999, "Name", nil, false, true)
	if err != ErrProviderNotFound {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestUpdateUserProvider_WithCredentials(t *testing.T) {
	db := setupTestDB(t)
	service := newTestAIProviderService(db, nil)
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

func TestDeleteUserProvider(t *testing.T) {
	db := setupTestDB(t)
	service := newTestAIProviderService(db, nil)
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
	db := setupTestDB(t)
	service := newTestAIProviderService(db, nil)
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
	db := setupTestDB(t)
	service := newTestAIProviderService(db, nil)
	ctx := context.Background()

	err := service.SetDefaultProvider(ctx, 999)
	if err != ErrProviderNotFound {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestCreateUserProvider_EmptyCredentials(t *testing.T) {
	db := setupTestDB(t)
	service := newTestAIProviderService(db, nil)
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

func TestCreateUserProvider_SetsDefault(t *testing.T) {
	db := setupTestDB(t)
	service := newTestAIProviderService(db, nil)
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
