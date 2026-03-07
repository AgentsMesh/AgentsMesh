package organization

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
)

func TestAddMember(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	req := &CreateRequest{Name: "Test Org", Slug: "test-org"}
	org, _ := service.Create(ctx, 1, req)

	// Add a new member
	err := service.AddMember(ctx, org.ID, 2, organization.RoleMember)
	if err != nil {
		t.Fatalf("failed to add member: %v", err)
	}

	member, err := service.GetMember(ctx, org.ID, 2)
	if err != nil {
		t.Fatalf("failed to get member: %v", err)
	}
	if member.Role != "member" {
		t.Errorf("expected Role 'member', got %s", member.Role)
	}
}

func TestRemoveMember(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	req := &CreateRequest{Name: "Test Org", Slug: "test-org"}
	org, _ := service.Create(ctx, 1, req)
	service.AddMember(ctx, org.ID, 2, organization.RoleMember)

	err := service.RemoveMember(ctx, org.ID, 2)
	if err != nil {
		t.Fatalf("failed to remove member: %v", err)
	}

	_, err = service.GetMember(ctx, org.ID, 2)
	if err == nil {
		t.Error("expected error when getting removed member")
	}
}

func TestRemoveMemberOwner(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	req := &CreateRequest{Name: "Test Org", Slug: "test-org"}
	org, _ := service.Create(ctx, 1, req)

	// Try to remove owner
	err := service.RemoveMember(ctx, org.ID, 1)
	if err != ErrCannotRemoveOwner {
		t.Errorf("expected ErrCannotRemoveOwner, got %v", err)
	}
}

func TestUpdateMemberRole(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	req := &CreateRequest{Name: "Test Org", Slug: "test-org"}
	org, _ := service.Create(ctx, 1, req)
	service.AddMember(ctx, org.ID, 2, organization.RoleMember)

	err := service.UpdateMemberRole(ctx, org.ID, 2, organization.RoleAdmin)
	if err != nil {
		t.Fatalf("failed to update member role: %v", err)
	}

	member, _ := service.GetMember(ctx, org.ID, 2)
	if member.Role != "admin" {
		t.Errorf("expected Role 'admin', got %s", member.Role)
	}
}

func TestListMembers(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	// Create test users first
	db.Exec("INSERT INTO users (id, email, username) VALUES (1, 'user1@test.com', 'user1')")
	db.Exec("INSERT INTO users (id, email, username) VALUES (2, 'user2@test.com', 'user2')")
	db.Exec("INSERT INTO users (id, email, username) VALUES (3, 'user3@test.com', 'user3')")

	req := &CreateRequest{Name: "Test Org", Slug: "test-org"}
	org, _ := service.Create(ctx, 1, req)
	service.AddMember(ctx, org.ID, 2, organization.RoleAdmin)
	service.AddMember(ctx, org.ID, 3, organization.RoleMember)

	members, err := service.ListMembers(ctx, org.ID)
	if err != nil {
		t.Fatalf("failed to list members: %v", err)
	}

	if len(members) != 3 {
		t.Errorf("expected 3 members, got %d", len(members))
	}
}

func TestListMembersEmpty(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	members, err := service.ListMembers(ctx, 999)
	if err != nil {
		t.Fatalf("failed to list members: %v", err)
	}

	if len(members) != 0 {
		t.Errorf("expected 0 members, got %d", len(members))
	}
}

func TestGetMemberNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	req := &CreateRequest{Name: "Test Org", Slug: "test-org"}
	org, _ := service.Create(ctx, 1, req)

	_, err := service.GetMember(ctx, org.ID, 999)
	if err == nil {
		t.Error("expected error for non-member")
	}
}
