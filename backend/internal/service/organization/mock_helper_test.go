package organization

import (
	"context"
	"errors"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/organization"
)

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
