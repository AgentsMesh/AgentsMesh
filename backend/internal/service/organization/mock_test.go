package organization

import (
	"context"
	"errors"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
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

func TestMockListByUser(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	org1, _ := mock.Create(ctx, 1, &CreateRequest{Slug: "org-1", Name: "Org 1"})
	org2, _ := mock.Create(ctx, 2, &CreateRequest{Slug: "org-2", Name: "Org 2"})
	mock.AddMember(ctx, org2.ID, 1, organization.RoleMember)

	t.Run("lists user organizations", func(t *testing.T) {
		orgs, err := mock.ListByUser(ctx, 1)
		if err != nil {
			t.Fatalf("ListByUser failed: %v", err)
		}
		if len(orgs) != 2 {
			t.Errorf("Orgs count = %d, want 2", len(orgs))
		}
	})

	t.Run("user with no orgs", func(t *testing.T) {
		orgs, err := mock.ListByUser(ctx, 999)
		if err != nil {
			t.Fatalf("ListByUser failed: %v", err)
		}
		if len(orgs) != 0 {
			t.Errorf("Orgs count = %d, want 0", len(orgs))
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("list error")
		mock.ListByUserErr = customErr
		_, err := mock.ListByUser(ctx, 1)
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.ListByUserErr = nil
	})

	_ = org1 // Suppress unused warning
}

func TestMockAddMember(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	org, _ := mock.Create(ctx, 1, &CreateRequest{Slug: "test-org", Name: "Test"})

	t.Run("adds member", func(t *testing.T) {
		err := mock.AddMember(ctx, org.ID, 2, organization.RoleMember)
		if err != nil {
			t.Fatalf("AddMember failed: %v", err)
		}

		member, _ := mock.GetMember(ctx, org.ID, 2)
		if member.Role != organization.RoleMember {
			t.Errorf("Role = %s, want member", member.Role)
		}

		if len(mock.AddedMembers) != 1 {
			t.Errorf("AddedMembers count = %d, want 1", len(mock.AddedMembers))
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("add error")
		mock.AddMemberErr = customErr
		err := mock.AddMember(ctx, org.ID, 3, organization.RoleMember)
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.AddMemberErr = nil
	})
}

func TestMockRemoveMember(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	org, _ := mock.Create(ctx, 1, &CreateRequest{Slug: "test-org", Name: "Test"})
	mock.AddMember(ctx, org.ID, 2, organization.RoleMember)

	t.Run("removes member", func(t *testing.T) {
		err := mock.RemoveMember(ctx, org.ID, 2)
		if err != nil {
			t.Fatalf("RemoveMember failed: %v", err)
		}

		_, err = mock.GetMember(ctx, org.ID, 2)
		if err == nil {
			t.Error("Member should be removed")
		}

		if len(mock.RemovedMembers) != 1 {
			t.Errorf("RemovedMembers count = %d, want 1", len(mock.RemovedMembers))
		}
	})

	t.Run("cannot remove owner", func(t *testing.T) {
		err := mock.RemoveMember(ctx, org.ID, 1)
		if err != ErrCannotRemoveOwner {
			t.Errorf("Expected ErrCannotRemoveOwner, got %v", err)
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("remove error")
		mock.RemoveMemberErr = customErr
		err := mock.RemoveMember(ctx, org.ID, 3)
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.RemoveMemberErr = nil
	})
}

func TestMockUpdateMemberRole(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	org, _ := mock.Create(ctx, 1, &CreateRequest{Slug: "test-org", Name: "Test"})
	mock.AddMember(ctx, org.ID, 2, organization.RoleMember)

	t.Run("updates role", func(t *testing.T) {
		err := mock.UpdateMemberRole(ctx, org.ID, 2, organization.RoleAdmin)
		if err != nil {
			t.Fatalf("UpdateMemberRole failed: %v", err)
		}

		member, _ := mock.GetMember(ctx, org.ID, 2)
		if member.Role != organization.RoleAdmin {
			t.Errorf("Role = %s, want admin", member.Role)
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("role error")
		mock.UpdateMemberRoleErr = customErr
		err := mock.UpdateMemberRole(ctx, org.ID, 2, organization.RoleMember)
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.UpdateMemberRoleErr = nil
	})
}

func TestMockGetMember(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	org, _ := mock.Create(ctx, 1, &CreateRequest{Slug: "test-org", Name: "Test"})

	t.Run("gets existing member", func(t *testing.T) {
		member, err := mock.GetMember(ctx, org.ID, 1)
		if err != nil {
			t.Fatalf("GetMember failed: %v", err)
		}
		if member.UserID != 1 {
			t.Errorf("UserID = %d, want 1", member.UserID)
		}
	})

	t.Run("non-existent member", func(t *testing.T) {
		_, err := mock.GetMember(ctx, org.ID, 999)
		if err != ErrOrganizationNotFound {
			t.Errorf("Expected ErrOrganizationNotFound, got %v", err)
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("member error")
		mock.GetMemberErr = customErr
		_, err := mock.GetMember(ctx, org.ID, 1)
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.GetMemberErr = nil
	})
}

func TestMockListMembers(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	org, _ := mock.Create(ctx, 1, &CreateRequest{Slug: "test-org", Name: "Test"})
	mock.AddMember(ctx, org.ID, 2, organization.RoleMember)

	t.Run("lists members", func(t *testing.T) {
		members, err := mock.ListMembers(ctx, org.ID)
		if err != nil {
			t.Fatalf("ListMembers failed: %v", err)
		}
		if len(members) != 2 {
			t.Errorf("Members count = %d, want 2", len(members))
		}
	})

	t.Run("configurable error", func(t *testing.T) {
		customErr := errors.New("list error")
		mock.ListMembersErr = customErr
		_, err := mock.ListMembers(ctx, org.ID)
		if err != customErr {
			t.Errorf("Expected custom error, got %v", err)
		}
		mock.ListMembersErr = nil
	})
}

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

func TestMockHelperMethods(t *testing.T) {
	ctx := context.Background()
	mock := NewMockService()

	t.Run("AddOrg helper", func(t *testing.T) {
		org := &organization.Organization{
			Name: "Helper Org",
			Slug: "helper-org",
		}
		mock.AddOrg(org)

		result, err := mock.GetBySlug(ctx, "helper-org")
		if err != nil {
			t.Fatalf("GetBySlug failed: %v", err)
		}
		if result.GetID() == 0 {
			t.Error("ID should be auto-assigned")
		}
	})

	t.Run("AddOrg with ID", func(t *testing.T) {
		org := &organization.Organization{
			ID:   100,
			Name: "ID Org",
			Slug: "id-org",
		}
		mock.AddOrg(org)

		result, _ := mock.GetByID(ctx, 100)
		if result == nil {
			t.Error("Org should be found by ID")
		}
	})

	t.Run("SetMember helper", func(t *testing.T) {
		mock.SetMember(1, 10, organization.RoleAdmin)

		member, _ := mock.GetMember(ctx, 1, 10)
		if member.Role != organization.RoleAdmin {
			t.Errorf("Role = %s, want admin", member.Role)
		}
	})

	t.Run("GetOrgs helper", func(t *testing.T) {
		orgs := mock.GetOrgs()
		if len(orgs) < 2 {
			t.Errorf("Expected at least 2 orgs, got %d", len(orgs))
		}
	})

	t.Run("Reset helper", func(t *testing.T) {
		mock.Create(ctx, 1, &CreateRequest{Slug: "reset-org", Name: "Reset"})
		mock.Reset()

		orgs := mock.GetOrgs()
		if len(orgs) != 0 {
			t.Errorf("Orgs should be cleared, got %d", len(orgs))
		}
		if mock.nextID != 1 {
			t.Errorf("nextID should be reset to 1, got %d", mock.nextID)
		}
		if len(mock.CreatedOrgs) != 0 {
			t.Error("CreatedOrgs should be cleared")
		}
	})
}

func TestMockServiceImplementsInterface(t *testing.T) {
	// This test verifies that MockService implements Interface
	var _ Interface = (*MockService)(nil)
}
