package organization

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
)

func TestIsAdmin(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	req := &CreateRequest{Name: "Test Org", Slug: "test-org"}
	org, _ := service.Create(ctx, 1, req)
	service.AddMember(ctx, org.ID, 2, organization.RoleAdmin)
	service.AddMember(ctx, org.ID, 3, organization.RoleMember)

	// Owner is admin
	isAdmin, _ := service.IsAdmin(ctx, org.ID, 1)
	if !isAdmin {
		t.Error("expected owner to be admin")
	}

	// Admin is admin
	isAdmin, _ = service.IsAdmin(ctx, org.ID, 2)
	if !isAdmin {
		t.Error("expected admin to be admin")
	}

	// Member is not admin
	isAdmin, _ = service.IsAdmin(ctx, org.ID, 3)
	if isAdmin {
		t.Error("expected member not to be admin")
	}
}

func TestIsAdminNonMember(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	req := &CreateRequest{Name: "Test Org", Slug: "test-org"}
	org, _ := service.Create(ctx, 1, req)

	// Check non-member
	isAdmin, err := service.IsAdmin(ctx, org.ID, 999)
	if err != nil {
		t.Fatalf("failed to check admin: %v", err)
	}
	if isAdmin {
		t.Error("expected non-member not to be admin")
	}
}

func TestIsOwner(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	req := &CreateRequest{Name: "Test Org", Slug: "test-org"}
	org, _ := service.Create(ctx, 1, req)
	service.AddMember(ctx, org.ID, 2, organization.RoleAdmin)

	// User 1 is owner
	isOwner, _ := service.IsOwner(ctx, org.ID, 1)
	if !isOwner {
		t.Error("expected user 1 to be owner")
	}

	// Admin is not owner
	isOwner, _ = service.IsOwner(ctx, org.ID, 2)
	if isOwner {
		t.Error("expected admin not to be owner")
	}
}

func TestIsOwnerNonMember(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	req := &CreateRequest{Name: "Test Org", Slug: "test-org"}
	org, _ := service.Create(ctx, 1, req)

	isOwner, err := service.IsOwner(ctx, org.ID, 999)
	if err != nil {
		t.Fatalf("failed to check owner: %v", err)
	}
	if isOwner {
		t.Error("expected non-member not to be owner")
	}
}

func TestIsMember(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	req := &CreateRequest{Name: "Test Org", Slug: "test-org"}
	org, _ := service.Create(ctx, 1, req)

	// Owner is member
	isMember, _ := service.IsMember(ctx, org.ID, 1)
	if !isMember {
		t.Error("expected owner to be member")
	}

	// Non-member is not member
	isMember, _ = service.IsMember(ctx, org.ID, 999)
	if isMember {
		t.Error("expected non-member not to be member")
	}
}

func TestGetUserRole(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	req := &CreateRequest{Name: "Test Org", Slug: "test-org"}
	org, _ := service.Create(ctx, 1, req)

	role, err := service.GetUserRole(ctx, org.ID, 1)
	if err != nil {
		t.Fatalf("failed to get user role: %v", err)
	}
	if role != "owner" {
		t.Errorf("expected Role 'owner', got %s", role)
	}
}

func TestGetUserRoleNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	req := &CreateRequest{Name: "Test Org", Slug: "test-org"}
	org, _ := service.Create(ctx, 1, req)

	_, err := service.GetUserRole(ctx, org.ID, 999)
	if err == nil {
		t.Error("expected error for non-member")
	}
}

func TestGetMemberRole(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	req := &CreateRequest{Name: "Test Org", Slug: "test-org"}
	org, _ := service.Create(ctx, 1, req)

	role, err := service.GetMemberRole(ctx, org.ID, 1)
	if err != nil {
		t.Fatalf("failed to get member role: %v", err)
	}
	if role != "owner" {
		t.Errorf("expected Role 'owner', got %s", role)
	}
}
