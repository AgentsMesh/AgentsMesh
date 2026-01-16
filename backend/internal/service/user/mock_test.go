package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
)

func TestNewMockService(t *testing.T) {
	mock := NewMockService()
	if mock == nil {
		t.Fatal("expected non-nil mock service")
	}
	if mock.users == nil {
		t.Error("users map should be initialized")
	}
	if mock.identities == nil {
		t.Error("identities map should be initialized")
	}
	if mock.nextID != 1 {
		t.Errorf("nextID = %d, want 1", mock.nextID)
	}
}

func TestMockServiceCreate(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	t.Run("creates user successfully", func(t *testing.T) {
		req := &CreateRequest{
			Email:    "test@example.com",
			Username: "testuser",
			Name:     "Test User",
		}

		u, err := mock.Create(ctx, req)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if u.Email != "test@example.com" {
			t.Errorf("Email = %s, want test@example.com", u.Email)
		}
		if u.Username != "testuser" {
			t.Errorf("Username = %s, want testuser", u.Username)
		}
		if u.Name == nil || *u.Name != "Test User" {
			t.Error("Name not set correctly")
		}
		if !u.IsActive {
			t.Error("User should be active")
		}
		if len(mock.CreatedUsers) != 1 {
			t.Errorf("CreatedUsers count = %d, want 1", len(mock.CreatedUsers))
		}
	})

	t.Run("creates user without name", func(t *testing.T) {
		mock2 := NewMockService()
		req := &CreateRequest{
			Email:    "noname@example.com",
			Username: "noname",
		}

		u, err := mock2.Create(ctx, req)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if u.Name != nil {
			t.Error("Name should be nil")
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("create error")
		mock.CreateErr = customErr

		_, err := mock.Create(ctx, &CreateRequest{Email: "err@test.com", Username: "err"})
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.CreateErr = nil
	})
}

func TestMockServiceGetByID(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	u, _ := mock.Create(ctx, &CreateRequest{Email: "test@example.com", Username: "test"})

	t.Run("existing user", func(t *testing.T) {
		result, err := mock.GetByID(ctx, u.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if result.ID != u.ID {
			t.Errorf("ID = %d, want %d", result.ID, u.ID)
		}
	})

	t.Run("non-existent user", func(t *testing.T) {
		_, err := mock.GetByID(ctx, 999)
		if err != ErrUserNotFound {
			t.Errorf("Expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("get error")
		mock.GetByIDErr = customErr
		_, err := mock.GetByID(ctx, u.ID)
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.GetByIDErr = nil
	})
}

func TestMockServiceGetByEmail(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	mock.Create(ctx, &CreateRequest{Email: "test@example.com", Username: "test"})

	t.Run("existing email", func(t *testing.T) {
		result, err := mock.GetByEmail(ctx, "test@example.com")
		if err != nil {
			t.Fatalf("GetByEmail failed: %v", err)
		}
		if result.Email != "test@example.com" {
			t.Errorf("Email = %s, want test@example.com", result.Email)
		}
	})

	t.Run("non-existent email", func(t *testing.T) {
		_, err := mock.GetByEmail(ctx, "nonexistent@example.com")
		if err != ErrUserNotFound {
			t.Errorf("Expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("email error")
		mock.GetByEmailErr = customErr
		_, err := mock.GetByEmail(ctx, "test@example.com")
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.GetByEmailErr = nil
	})
}

func TestMockServiceGetByUsername(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	mock.Create(ctx, &CreateRequest{Email: "test@example.com", Username: "testuser"})

	t.Run("existing username", func(t *testing.T) {
		result, err := mock.GetByUsername(ctx, "testuser")
		if err != nil {
			t.Fatalf("GetByUsername failed: %v", err)
		}
		if result.Username != "testuser" {
			t.Errorf("Username = %s, want testuser", result.Username)
		}
	})

	t.Run("non-existent username", func(t *testing.T) {
		_, err := mock.GetByUsername(ctx, "nonexistent")
		if err != ErrUserNotFound {
			t.Errorf("Expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("username error")
		mock.GetByUsernameErr = customErr
		_, err := mock.GetByUsername(ctx, "testuser")
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.GetByUsernameErr = nil
	})
}

func TestMockServiceUpdate(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	u, _ := mock.Create(ctx, &CreateRequest{Email: "test@example.com", Username: "test"})

	t.Run("updates user", func(t *testing.T) {
		updates := map[string]interface{}{
			"name":       "New Name",
			"avatar_url": "https://example.com/avatar.png",
		}
		result, err := mock.Update(ctx, u.ID, updates)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if result.Name == nil || *result.Name != "New Name" {
			t.Error("Name not updated")
		}
		if result.AvatarURL == nil || *result.AvatarURL != "https://example.com/avatar.png" {
			t.Error("AvatarURL not updated")
		}
		if len(mock.UpdatedUsers) != 1 {
			t.Errorf("UpdatedUsers count = %d, want 1", len(mock.UpdatedUsers))
		}
	})

	t.Run("non-existent user", func(t *testing.T) {
		_, err := mock.Update(ctx, 999, map[string]interface{}{})
		if err != ErrUserNotFound {
			t.Errorf("Expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("update error")
		mock.UpdateErr = customErr
		_, err := mock.Update(ctx, u.ID, map[string]interface{}{})
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.UpdateErr = nil
	})
}

func TestMockServiceUpdatePassword(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	u, _ := mock.Create(ctx, &CreateRequest{Email: "test@example.com", Username: "test"})

	t.Run("updates password", func(t *testing.T) {
		err := mock.UpdatePassword(ctx, u.ID, "newpassword")
		if err != nil {
			t.Fatalf("UpdatePassword failed: %v", err)
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("password error")
		mock.UpdatePasswordErr = customErr
		err := mock.UpdatePassword(ctx, u.ID, "newpassword")
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.UpdatePasswordErr = nil
	})
}

func TestMockServiceDelete(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	u, _ := mock.Create(ctx, &CreateRequest{Email: "test@example.com", Username: "test"})

	t.Run("deletes user", func(t *testing.T) {
		err := mock.Delete(ctx, u.ID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		_, err = mock.GetByID(ctx, u.ID)
		if err != ErrUserNotFound {
			t.Error("User should be deleted")
		}

		if len(mock.DeletedUserIDs) != 1 {
			t.Errorf("DeletedUserIDs count = %d, want 1", len(mock.DeletedUserIDs))
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("delete error")
		mock.DeleteErr = customErr
		err := mock.Delete(ctx, 1)
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.DeleteErr = nil
	})
}

func TestMockServiceAuthenticate(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	mock.Create(ctx, &CreateRequest{Email: "test@example.com", Username: "test"})

	t.Run("authenticates user", func(t *testing.T) {
		u, err := mock.Authenticate(ctx, "test@example.com", "password")
		if err != nil {
			t.Fatalf("Authenticate failed: %v", err)
		}
		if u.Email != "test@example.com" {
			t.Errorf("Email = %s, want test@example.com", u.Email)
		}
		if len(mock.AuthAttempts) != 1 {
			t.Errorf("AuthAttempts count = %d, want 1", len(mock.AuthAttempts))
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("auth error")
		mock.AuthenticateErr = customErr
		_, err := mock.Authenticate(ctx, "test@example.com", "password")
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.AuthenticateErr = nil
	})
}

func TestMockServiceGetOrCreateByOAuth(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	t.Run("creates new user", func(t *testing.T) {
		u, isNew, err := mock.GetOrCreateByOAuth(ctx, "github", "12345", "ghuser", "oauth@example.com", "OAuth User", "https://avatar.com")
		if err != nil {
			t.Fatalf("GetOrCreateByOAuth failed: %v", err)
		}
		if !isNew {
			t.Error("Should be new user")
		}
		if u.Email != "oauth@example.com" {
			t.Errorf("Email = %s, want oauth@example.com", u.Email)
		}
	})

	t.Run("returns existing user", func(t *testing.T) {
		u, isNew, err := mock.GetOrCreateByOAuth(ctx, "github", "12345", "ghuser", "oauth@example.com", "OAuth User", "")
		if err != nil {
			t.Fatalf("GetOrCreateByOAuth failed: %v", err)
		}
		if isNew {
			t.Error("Should not be new user")
		}
		if u.Email != "oauth@example.com" {
			t.Errorf("Email = %s, want oauth@example.com", u.Email)
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("oauth error")
		mock.GetOrCreateByOAuthErr = customErr
		_, _, err := mock.GetOrCreateByOAuth(ctx, "github", "999", "user", "new@example.com", "", "")
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.GetOrCreateByOAuthErr = nil
	})
}

func TestMockServiceUpdateIdentityTokens(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	u, _ := mock.Create(ctx, &CreateRequest{Email: "test@example.com", Username: "test"})
	expiresAt := time.Now().Add(time.Hour)

	t.Run("updates tokens", func(t *testing.T) {
		err := mock.UpdateIdentityTokens(ctx, u.ID, "github", "access", "refresh", &expiresAt)
		if err != nil {
			t.Fatalf("UpdateIdentityTokens failed: %v", err)
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("identity error")
		mock.UpdateIdentityErr = customErr
		err := mock.UpdateIdentityTokens(ctx, u.ID, "github", "access", "refresh", nil)
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.UpdateIdentityErr = nil
	})
}

func TestMockServiceGetIdentity(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	u, _ := mock.Create(ctx, &CreateRequest{Email: "test@example.com", Username: "test"})
	mock.AddIdentity(u.ID, &user.Identity{Provider: "github", ProviderUserID: "12345"})

	t.Run("gets existing identity", func(t *testing.T) {
		identity, err := mock.GetIdentity(ctx, u.ID, "github")
		if err != nil {
			t.Fatalf("GetIdentity failed: %v", err)
		}
		if identity.Provider != "github" {
			t.Errorf("Provider = %s, want github", identity.Provider)
		}
	})

	t.Run("non-existent identity", func(t *testing.T) {
		_, err := mock.GetIdentity(ctx, u.ID, "gitlab")
		if err != ErrUserNotFound {
			t.Errorf("Expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("identity error")
		mock.GetIdentityErr = customErr
		_, err := mock.GetIdentity(ctx, u.ID, "github")
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.GetIdentityErr = nil
	})
}

func TestMockServiceListIdentities(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	u, _ := mock.Create(ctx, &CreateRequest{Email: "test@example.com", Username: "test"})
	mock.AddIdentity(u.ID, &user.Identity{Provider: "github", ProviderUserID: "12345"})
	mock.AddIdentity(u.ID, &user.Identity{Provider: "gitlab", ProviderUserID: "67890"})

	t.Run("lists identities", func(t *testing.T) {
		identities, err := mock.ListIdentities(ctx, u.ID)
		if err != nil {
			t.Fatalf("ListIdentities failed: %v", err)
		}
		if len(identities) != 2 {
			t.Errorf("Identities count = %d, want 2", len(identities))
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("list error")
		mock.ListIdentitiesErr = customErr
		_, err := mock.ListIdentities(ctx, u.ID)
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.ListIdentitiesErr = nil
	})
}

func TestMockServiceDeleteIdentity(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	u, _ := mock.Create(ctx, &CreateRequest{Email: "test@example.com", Username: "test"})
	mock.AddIdentity(u.ID, &user.Identity{Provider: "github", ProviderUserID: "12345"})
	mock.AddIdentity(u.ID, &user.Identity{Provider: "gitlab", ProviderUserID: "67890"})

	t.Run("deletes identity", func(t *testing.T) {
		err := mock.DeleteIdentity(ctx, u.ID, "github")
		if err != nil {
			t.Fatalf("DeleteIdentity failed: %v", err)
		}

		identities, _ := mock.ListIdentities(ctx, u.ID)
		if len(identities) != 1 {
			t.Errorf("Identities count = %d, want 1", len(identities))
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("delete error")
		mock.DeleteIdentityErr = customErr
		err := mock.DeleteIdentity(ctx, u.ID, "gitlab")
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.DeleteIdentityErr = nil
	})
}

func TestMockServiceSearch(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	mock.Create(ctx, &CreateRequest{Email: "alice@example.com", Username: "alice"})
	mock.Create(ctx, &CreateRequest{Email: "bob@example.com", Username: "bob"})

	t.Run("searches users", func(t *testing.T) {
		results, err := mock.Search(ctx, "alice", 10)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) == 0 {
			t.Error("Should return some results")
		}
		if len(mock.SearchQueries) != 1 {
			t.Errorf("SearchQueries count = %d, want 1", len(mock.SearchQueries))
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		results, err := mock.Search(ctx, "user", 1)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) > 1 {
			t.Errorf("Results count = %d, want <= 1", len(results))
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("search error")
		mock.SearchErr = customErr
		_, err := mock.Search(ctx, "test", 10)
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.SearchErr = nil
	})
}

func TestMockServiceHelperMethods(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	t.Run("AddUser helper", func(t *testing.T) {
		u := &user.User{
			Email:    "helper@example.com",
			Username: "helper",
		}
		mock.AddUser(u)

		result, err := mock.GetByEmail(ctx, "helper@example.com")
		if err != nil {
			t.Fatalf("GetByEmail failed: %v", err)
		}
		if result.ID == 0 {
			t.Error("ID should be auto-assigned")
		}
	})

	t.Run("AddUser with ID", func(t *testing.T) {
		u := &user.User{
			ID:       100,
			Email:    "iduser@example.com",
			Username: "iduser",
		}
		mock.AddUser(u)

		result, _ := mock.GetByID(ctx, 100)
		if result == nil {
			t.Error("User should be found by ID")
		}
	})

	t.Run("AddIdentity helper", func(t *testing.T) {
		u, _ := mock.Create(ctx, &CreateRequest{Email: "identity@example.com", Username: "identity"})
		mock.AddIdentity(u.ID, &user.Identity{Provider: "github", ProviderUserID: "999"})

		identity, err := mock.GetIdentity(ctx, u.ID, "github")
		if err != nil {
			t.Fatalf("GetIdentity failed: %v", err)
		}
		if identity.ProviderUserID != "999" {
			t.Errorf("ProviderUserID = %s, want 999", identity.ProviderUserID)
		}
	})

	t.Run("GetUsers helper", func(t *testing.T) {
		users := mock.GetUsers()
		if len(users) < 2 {
			t.Errorf("Expected at least 2 users, got %d", len(users))
		}
	})

	t.Run("Reset helper", func(t *testing.T) {
		mock.Create(ctx, &CreateRequest{Email: "reset@example.com", Username: "reset"})
		mock.Reset()

		users := mock.GetUsers()
		if len(users) != 0 {
			t.Errorf("Users should be cleared, got %d", len(users))
		}
		if mock.nextID != 1 {
			t.Errorf("nextID should be reset to 1, got %d", mock.nextID)
		}
		if len(mock.CreatedUsers) != 0 {
			t.Error("CreatedUsers should be cleared")
		}
		if len(mock.AuthAttempts) != 0 {
			t.Error("AuthAttempts should be cleared")
		}
	})
}

func TestMockServiceImplementsInterface(t *testing.T) {
	// This test verifies that MockService implements Interface
	var _ Interface = (*MockService)(nil)
}
