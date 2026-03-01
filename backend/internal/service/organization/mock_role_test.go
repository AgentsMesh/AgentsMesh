package organization

import (
	"context"
	"errors"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/organization"
)

func TestMockIsAdmin(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	org, _ := mock.Create(ctx, 1, &CreateRequest{Slug: "test-org", Name: "Test"})
	mock.AddMember(ctx, org.ID, 2, organization.RoleAdmin)
	mock.AddMember(ctx, org.ID, 3, organization.RoleMember)

	t.Run("owner is admin", func(t *testing.T) {
		isAdmin, _ := mock.IsAdmin(ctx, org.ID, 1)
		if !isAdmin {
			t.Error("Owner should be admin")
		}
	})

	t.Run("admin is admin", func(t *testing.T) {
		isAdmin, _ := mock.IsAdmin(ctx, org.ID, 2)
		if !isAdmin {
			t.Error("Admin should be admin")
		}
	})

	t.Run("member is not admin", func(t *testing.T) {
		isAdmin, _ := mock.IsAdmin(ctx, org.ID, 3)
		if isAdmin {
			t.Error("Member should not be admin")
		}
	})

	t.Run("non-member is not admin", func(t *testing.T) {
		isAdmin, _ := mock.IsAdmin(ctx, org.ID, 999)
		if isAdmin {
			t.Error("Non-member should not be admin")
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("admin error")
		mock.IsAdminErr = customErr
		_, err := mock.IsAdmin(ctx, org.ID, 1)
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.IsAdminErr = nil
	})
}

func TestMockIsOwner(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	org, _ := mock.Create(ctx, 1, &CreateRequest{Slug: "test-org", Name: "Test"})
	mock.AddMember(ctx, org.ID, 2, organization.RoleAdmin)

	t.Run("owner is owner", func(t *testing.T) {
		isOwner, _ := mock.IsOwner(ctx, org.ID, 1)
		if !isOwner {
			t.Error("Owner should be owner")
		}
	})

	t.Run("admin is not owner", func(t *testing.T) {
		isOwner, _ := mock.IsOwner(ctx, org.ID, 2)
		if isOwner {
			t.Error("Admin should not be owner")
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("owner error")
		mock.IsOwnerErr = customErr
		_, err := mock.IsOwner(ctx, org.ID, 1)
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.IsOwnerErr = nil
	})
}

func TestMockIsMember(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	org, _ := mock.Create(ctx, 1, &CreateRequest{Slug: "test-org", Name: "Test"})

	t.Run("member is member", func(t *testing.T) {
		isMember, _ := mock.IsMember(ctx, org.ID, 1)
		if !isMember {
			t.Error("Should be member")
		}
	})

	t.Run("non-member is not member", func(t *testing.T) {
		isMember, _ := mock.IsMember(ctx, org.ID, 999)
		if isMember {
			t.Error("Should not be member")
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("member error")
		mock.IsMemberErr = customErr
		_, err := mock.IsMember(ctx, org.ID, 1)
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.IsMemberErr = nil
	})
}

func TestMockGetUserRole(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	org, _ := mock.Create(ctx, 1, &CreateRequest{Slug: "test-org", Name: "Test"})

	t.Run("gets role", func(t *testing.T) {
		role, err := mock.GetUserRole(ctx, org.ID, 1)
		if err != nil {
			t.Fatalf("GetUserRole failed: %v", err)
		}
		if role != organization.RoleOwner {
			t.Errorf("Role = %s, want owner", role)
		}
	})

	t.Run("non-existent member", func(t *testing.T) {
		_, err := mock.GetUserRole(ctx, org.ID, 999)
		if err != ErrOrganizationNotFound {
			t.Errorf("Expected ErrOrganizationNotFound, got %v", err)
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("role error")
		mock.GetUserRoleErr = customErr
		_, err := mock.GetUserRole(ctx, org.ID, 1)
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.GetUserRoleErr = nil
	})
}

func TestMockGetMemberRole(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	org, _ := mock.Create(ctx, 1, &CreateRequest{Slug: "test-org", Name: "Test"})

	role, err := mock.GetMemberRole(ctx, org.ID, 1)
	if err != nil {
		t.Fatalf("GetMemberRole failed: %v", err)
	}
	if role != organization.RoleOwner {
		t.Errorf("Role = %s, want owner", role)
	}
}
