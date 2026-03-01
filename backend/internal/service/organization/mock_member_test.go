package organization

import (
	"context"
	"errors"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/organization"
)

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
