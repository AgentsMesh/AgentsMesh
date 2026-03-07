package user

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
)

func TestGetOrCreateByOAuth(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	// Create new user via OAuth
	user, isNew, err := service.GetOrCreateByOAuth(ctx, "github", "12345", "githubuser", "test@example.com", "Test User", "https://example.com/avatar.png")
	if err != nil {
		t.Fatalf("failed to get or create user: %v", err)
	}
	if !isNew {
		t.Error("expected isNew to be true")
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected Email 'test@example.com', got %s", user.Email)
	}

	// Get existing user via OAuth
	user2, isNew2, err := service.GetOrCreateByOAuth(ctx, "github", "12345", "githubuser", "test@example.com", "Test User", "")
	if err != nil {
		t.Fatalf("failed to get existing user: %v", err)
	}
	if isNew2 {
		t.Error("expected isNew to be false")
	}
	if user2.ID != user.ID {
		t.Errorf("expected same user ID %d, got %d", user.ID, user2.ID)
	}
}

func TestGetOrCreateByOAuthExistingVerifiedEmail(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	// Create user via regular signup
	created, _ := service.Create(ctx, &CreateRequest{
		Email:    "existing@example.com",
		Username: "existing",
	})

	// Mark email as verified
	db.Model(created).Update("is_email_verified", true)

	// OAuth with same verified email should link to existing user
	user, isNew, err := service.GetOrCreateByOAuth(ctx, "gitlab", "99999", "gitlabuser", "existing@example.com", "Existing User", "")
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if isNew {
		t.Error("expected isNew to be false for existing verified email")
	}
	if user.Username != "existing" {
		t.Errorf("expected username 'existing', got %s", user.Username)
	}
}

func TestGetOrCreateByOAuthExistingUnverifiedEmail(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	// Create user via regular signup (email NOT verified)
	service.Create(ctx, &CreateRequest{
		Email:    "unverified@example.com",
		Username: "unverified",
	})

	// OAuth with same unverified email should create a NEW user
	// to prevent OAuth email takeover attacks
	user, isNew, err := service.GetOrCreateByOAuth(ctx, "gitlab", "88888", "attacker", "unverified@example.com", "Attacker", "")
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if !isNew {
		t.Error("expected isNew to be true for unverified email (should not link)")
	}
	if user.Username != "attacker" {
		t.Errorf("expected username 'attacker', got %s", user.Username)
	}
}

func TestGetOrCreateByOAuthDuplicateUsername(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	// Create user with username
	service.Create(ctx, &CreateRequest{
		Email:    "first@example.com",
		Username: "duplicate",
	})

	// OAuth with same username should get a modified username
	user, isNew, err := service.GetOrCreateByOAuth(ctx, "github", "11111", "duplicate", "second@example.com", "Second User", "")
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if !isNew {
		t.Error("expected isNew to be true")
	}
	// Username should be modified to avoid collision
	if user.Username == "duplicate" {
		t.Error("expected modified username due to collision")
	}
}

func TestListIdentities(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	// Create user with OAuth
	user, _, _ := service.GetOrCreateByOAuth(ctx, "github", "12345", "githubuser", "test@example.com", "Test", "")

	// List identities
	identities, err := service.ListIdentities(ctx, user.ID)
	if err != nil {
		t.Fatalf("failed to list identities: %v", err)
	}
	if len(identities) != 1 {
		t.Errorf("expected 1 identity, got %d", len(identities))
	}
}

func TestGetIdentity(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	user, _, _ := service.GetOrCreateByOAuth(ctx, "github", "12345", "githubuser", "test@example.com", "Test", "")

	identity, err := service.GetIdentity(ctx, user.ID, "github")
	if err != nil {
		t.Fatalf("failed to get identity: %v", err)
	}
	if identity.Provider != "github" {
		t.Errorf("expected Provider 'github', got %s", identity.Provider)
	}
}

func TestGetIdentityNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	req := &CreateRequest{
		Email:    "noid@example.com",
		Username: "noid",
	}
	created, _ := service.Create(ctx, req)

	_, err := service.GetIdentity(ctx, created.ID, "github")
	if err == nil {
		t.Error("expected error for non-existent identity")
	}
}

func TestDeleteIdentity(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	user, _, _ := service.GetOrCreateByOAuth(ctx, "github", "12345", "githubuser", "test@example.com", "Test", "")

	err := service.DeleteIdentity(ctx, user.ID, "github")
	if err != nil {
		t.Fatalf("failed to delete identity: %v", err)
	}

	identities, _ := service.ListIdentities(ctx, user.ID)
	if len(identities) != 0 {
		t.Errorf("expected 0 identities, got %d", len(identities))
	}
}

func TestUpdateIdentityTokens(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	user, _, _ := service.GetOrCreateByOAuth(ctx, "github", "12345", "githubuser", "test@example.com", "Test", "")

	err := service.UpdateIdentityTokens(ctx, user.ID, "github", "new_access_token", "new_refresh_token", nil)
	if err != nil {
		t.Fatalf("failed to update identity tokens: %v", err)
	}

	identity, _ := service.GetIdentity(ctx, user.ID, "github")
	if identity.AccessTokenEncrypted == nil || *identity.AccessTokenEncrypted != "new_access_token" {
		t.Error("expected access token to be updated")
	}
}

func TestGetOrCreateByOAuthEmptyEmail(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	// First user with empty email via OAuth
	user1, isNew1, err := service.GetOrCreateByOAuth(ctx, "github", "111", "user_a", "", "User A", "")
	if err != nil {
		t.Fatalf("failed to create first user: %v", err)
	}
	if !isNew1 {
		t.Error("expected isNew to be true for first user")
	}

	// Second user with empty email should NOT be linked to the first user
	user2, isNew2, err := service.GetOrCreateByOAuth(ctx, "github", "222", "user_b", "", "User B", "")
	if err != nil {
		t.Fatalf("failed to create second user: %v", err)
	}
	if !isNew2 {
		t.Error("expected isNew to be true for second user")
	}
	if user2.ID == user1.ID {
		t.Errorf("second user should have a different ID from first user, both got %d", user1.ID)
	}

	// Third user with empty email should also be independent
	user3, isNew3, err := service.GetOrCreateByOAuth(ctx, "github", "333", "user_c", "", "User C", "")
	if err != nil {
		t.Fatalf("failed to create third user: %v", err)
	}
	if !isNew3 {
		t.Error("expected isNew to be true for third user")
	}
	if user3.ID == user1.ID || user3.ID == user2.ID {
		t.Error("third user should be independent from first and second")
	}

	// Verify placeholder emails are unique
	if user1.Email == user2.Email || user1.Email == user3.Email || user2.Email == user3.Email {
		t.Errorf("placeholder emails should be unique: %q, %q, %q", user1.Email, user2.Email, user3.Email)
	}
}

// Note: Search tests are skipped because SQLite doesn't support ILIKE
// The Search function is tested through integration tests with PostgreSQL
