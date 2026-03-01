package user

import (
	"context"
	"errors"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
)

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
