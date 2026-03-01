package organization

import (
	"context"
	"errors"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/organization"
)

func TestNewMockService(t *testing.T) {
	mock := NewMockService()
	if mock == nil {
		t.Fatal("expected non-nil mock service")
	}
	if mock.orgs == nil {
		t.Error("orgs map should be initialized")
	}
	if mock.orgsBySlug == nil {
		t.Error("orgsBySlug map should be initialized")
	}
	if mock.members == nil {
		t.Error("members map should be initialized")
	}
	if mock.nextID != 1 {
		t.Errorf("nextID = %d, want 1", mock.nextID)
	}
}

func TestMockCreate(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	t.Run("creates organization successfully", func(t *testing.T) {
		req := &CreateRequest{
			Name:    "Test Org",
			Slug:    "test-org",
			LogoURL: "https://example.com/logo.png",
		}

		org, err := mock.Create(ctx, 1, req)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if org.Name != "Test Org" {
			t.Errorf("Name = %s, want Test Org", org.Name)
		}
		if org.Slug != "test-org" {
			t.Errorf("Slug = %s, want test-org", org.Slug)
		}
		if org.LogoURL == nil || *org.LogoURL != "https://example.com/logo.png" {
			t.Error("LogoURL not set correctly")
		}

		// Owner should be added as member
		member, err := mock.GetMember(ctx, org.ID, 1)
		if err != nil {
			t.Fatalf("GetMember failed: %v", err)
		}
		if member.Role != organization.RoleOwner {
			t.Errorf("Role = %s, want owner", member.Role)
		}

		// Request should be captured
		if len(mock.CreatedOrgs) != 1 {
			t.Errorf("CreatedOrgs count = %d, want 1", len(mock.CreatedOrgs))
		}
	})

	t.Run("duplicate slug error", func(t *testing.T) {
		req := &CreateRequest{Name: "Another", Slug: "test-org"}
		_, err := mock.Create(ctx, 2, req)
		if err != ErrSlugAlreadyExists {
			t.Errorf("Expected ErrSlugAlreadyExists, got %v", err)
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("create error")
		mock.CreateErr = customErr

		_, err := mock.Create(ctx, 3, &CreateRequest{Slug: "new-slug"})
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.CreateErr = nil
	})

	t.Run("create without logo", func(t *testing.T) {
		mock2 := NewMockService()
		req := &CreateRequest{Name: "No Logo", Slug: "no-logo"}
		org, err := mock2.Create(ctx, 1, req)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if org.LogoURL != nil {
			t.Error("LogoURL should be nil")
		}
	})
}

func TestMockGetByID(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	org, _ := mock.Create(ctx, 1, &CreateRequest{Slug: "test-org", Name: "Test"})

	t.Run("existing org", func(t *testing.T) {
		result, err := mock.GetByID(ctx, org.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if result.ID != org.ID {
			t.Errorf("ID = %d, want %d", result.ID, org.ID)
		}
	})

	t.Run("non-existent org", func(t *testing.T) {
		_, err := mock.GetByID(ctx, 999)
		if err != ErrOrganizationNotFound {
			t.Errorf("Expected ErrOrganizationNotFound, got %v", err)
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("get error")
		mock.GetByIDErr = customErr
		_, err := mock.GetByID(ctx, org.ID)
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.GetByIDErr = nil
	})
}

func TestMockGetBySlug(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	mock.Create(ctx, 1, &CreateRequest{Slug: "test-org", Name: "Test"})

	t.Run("existing slug", func(t *testing.T) {
		result, err := mock.GetBySlug(ctx, "test-org")
		if err != nil {
			t.Fatalf("GetBySlug failed: %v", err)
		}
		if result.GetSlug() != "test-org" {
			t.Errorf("Slug = %s, want test-org", result.GetSlug())
		}
	})

	t.Run("non-existent slug", func(t *testing.T) {
		_, err := mock.GetBySlug(ctx, "nonexistent")
		if err != ErrOrganizationNotFound {
			t.Errorf("Expected ErrOrganizationNotFound, got %v", err)
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("slug error")
		mock.GetBySlugErr = customErr
		_, err := mock.GetBySlug(ctx, "test-org")
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.GetBySlugErr = nil
	})
}

func TestMockGetOrgBySlug(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	mock.Create(ctx, 1, &CreateRequest{Slug: "test-org", Name: "Test"})

	t.Run("existing slug", func(t *testing.T) {
		result, err := mock.GetOrgBySlug(ctx, "test-org")
		if err != nil {
			t.Fatalf("GetOrgBySlug failed: %v", err)
		}
		if result.Slug != "test-org" {
			t.Errorf("Slug = %s, want test-org", result.Slug)
		}
	})

	t.Run("non-existent slug", func(t *testing.T) {
		_, err := mock.GetOrgBySlug(ctx, "nonexistent")
		if err != ErrOrganizationNotFound {
			t.Errorf("Expected ErrOrganizationNotFound, got %v", err)
		}
	})
}

func TestMockUpdate(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	org, _ := mock.Create(ctx, 1, &CreateRequest{Slug: "test-org", Name: "Test"})

	t.Run("updates organization", func(t *testing.T) {
		updates := map[string]interface{}{"name": "Updated Name"}
		result, err := mock.Update(ctx, org.ID, updates)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if result.Name != "Updated Name" {
			t.Errorf("Name = %s, want Updated Name", result.Name)
		}
		if len(mock.UpdatedOrgs) != 1 {
			t.Errorf("UpdatedOrgs count = %d, want 1", len(mock.UpdatedOrgs))
		}
	})

	t.Run("non-existent org", func(t *testing.T) {
		_, err := mock.Update(ctx, 999, map[string]interface{}{})
		if err != ErrOrganizationNotFound {
			t.Errorf("Expected ErrOrganizationNotFound, got %v", err)
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("update error")
		mock.UpdateErr = customErr
		_, err := mock.Update(ctx, org.ID, map[string]interface{}{})
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.UpdateErr = nil
	})
}

func TestMockDelete(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	org, _ := mock.Create(ctx, 1, &CreateRequest{Slug: "test-org", Name: "Test"})

	t.Run("deletes organization", func(t *testing.T) {
		err := mock.Delete(ctx, org.ID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		_, err = mock.GetByID(ctx, org.ID)
		if err != ErrOrganizationNotFound {
			t.Error("Org should be deleted")
		}

		if len(mock.DeletedOrgIDs) != 1 {
			t.Errorf("DeletedOrgIDs count = %d, want 1", len(mock.DeletedOrgIDs))
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
