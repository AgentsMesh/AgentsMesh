package user

import (
	"context"
	"errors"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
)

func TestGetOrCreateByOAuth_SignupDisabled_BlocksNewUser(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	service.SetDisableOAuthSignup(true)
	ctx := context.Background()

	u, isNew, err := service.GetOrCreateByOAuth(ctx, "google", "new-123", "newuser", "new@example.com", "New User", "")
	if !errors.Is(err, ErrOAuthSignupDisabled) {
		t.Fatalf("expected ErrOAuthSignupDisabled, got user=%v isNew=%v err=%v", u, isNew, err)
	}
	if u != nil {
		t.Errorf("expected no user, got %+v", u)
	}

	if _, err := service.GetByEmail(ctx, "new@example.com"); err == nil {
		t.Error("expected no account to be created when signup is disabled")
	}
}

func TestGetOrCreateByOAuth_SignupDisabled_AllowsExistingIdentity(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	created, isNew, err := service.GetOrCreateByOAuth(ctx, "google", "known-456", "known", "known@example.com", "Known User", "")
	if err != nil || !isNew {
		t.Fatalf("setup: expected new user, got isNew=%v err=%v", isNew, err)
	}

	service.SetDisableOAuthSignup(true)

	got, isNew2, err := service.GetOrCreateByOAuth(ctx, "google", "known-456", "known", "known@example.com", "Known User", "")
	if err != nil {
		t.Fatalf("existing linked identity should authenticate when signup is disabled, got err=%v", err)
	}
	if isNew2 {
		t.Error("expected isNew=false for an existing identity")
	}
	if got.ID != created.ID {
		t.Errorf("expected same user ID %d, got %d", created.ID, got.ID)
	}
}
