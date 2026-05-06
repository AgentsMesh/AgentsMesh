package user

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
)

func setupProviderService(t *testing.T) (*Service, int64) {
	t.Helper()
	db := setupTestDB(t)
	svc := NewService(infra.NewUserRepository(db))
	ctx := context.Background()
	u, err := svc.Create(ctx, &CreateRequest{Email: "rp@example.com", Username: "rpuser"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return svc, u.ID
}

func TestCreateRepositoryProvider_DefaultsIsActiveTrue(t *testing.T) {
	svc, userID := setupProviderService(t)

	p, err := svc.CreateRepositoryProvider(context.Background(), userID, &CreateRepositoryProviderRequest{
		ProviderType: user.ProviderTypeGitHub,
		Name:         "GitHub",
		BaseURL:      "https://github.com",
	})
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	if !p.IsActive {
		t.Fatal("newly-created provider must default to IsActive=true so it shows as enabled in UI")
	}

	reloaded, err := svc.GetRepositoryProvider(context.Background(), userID, p.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !reloaded.IsActive {
		t.Fatal("IsActive=true must persist to DB")
	}
	if !reloaded.ToResponse().IsActive {
		t.Fatal("ToResponse must surface IsActive=true to the API layer")
	}
}

func TestUpdateRepositoryProvider_TogglesIsActive(t *testing.T) {
	ctx := context.Background()
	svc, userID := setupProviderService(t)

	p, err := svc.CreateRepositoryProvider(ctx, userID, &CreateRepositoryProviderRequest{
		ProviderType: user.ProviderTypeGitHub, Name: "GitHub", BaseURL: "https://github.com",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	off := false
	updated, err := svc.UpdateRepositoryProvider(ctx, userID, p.ID, &UpdateRepositoryProviderRequest{IsActive: &off})
	if err != nil {
		t.Fatalf("update off: %v", err)
	}
	if updated.IsActive {
		t.Fatal("expected IsActive=false after toggling off")
	}

	on := true
	updated, err = svc.UpdateRepositoryProvider(ctx, userID, p.ID, &UpdateRepositoryProviderRequest{IsActive: &on})
	if err != nil {
		t.Fatalf("update on: %v", err)
	}
	if !updated.IsActive {
		t.Fatal("expected IsActive=true after toggling back on — this is the user-reported regression")
	}

	reloaded, err := svc.GetRepositoryProvider(ctx, userID, p.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !reloaded.IsActive {
		t.Fatal("IsActive change must persist across reload")
	}
}

func TestUpdateRepositoryProvider_PartialUpdatePreservesIsActive(t *testing.T) {
	ctx := context.Background()
	svc, userID := setupProviderService(t)

	p, _ := svc.CreateRepositoryProvider(ctx, userID, &CreateRepositoryProviderRequest{
		ProviderType: user.ProviderTypeGitHub, Name: "GitHub", BaseURL: "https://github.com",
	})
	off := false
	if _, err := svc.UpdateRepositoryProvider(ctx, userID, p.ID, &UpdateRepositoryProviderRequest{IsActive: &off}); err != nil {
		t.Fatalf("set inactive: %v", err)
	}

	newName := "GitHub Renamed"
	updated, err := svc.UpdateRepositoryProvider(ctx, userID, p.ID, &UpdateRepositoryProviderRequest{Name: &newName})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if updated.Name != "GitHub Renamed" {
		t.Errorf("expected name updated, got %q", updated.Name)
	}
	if updated.IsActive {
		t.Fatal("partial update without IsActive must preserve previous IsActive=false")
	}
}
