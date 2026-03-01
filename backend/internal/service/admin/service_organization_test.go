package admin

import (
	"context"
	"errors"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/organization"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListOrganizations(t *testing.T) {
	t.Run("should list organizations with pagination", func(t *testing.T) {
		db := newMockDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Org 1", Slug: "org-1"}
		db.organizations[2] = &organization.Organization{ID: 2, Name: "Org 2", Slug: "org-2"}
		db.totalOrgs = 2

		svc := NewService(db)
		result, err := svc.ListOrganizations(context.Background(), &OrganizationListQuery{
			Page:     1,
			PageSize: 20,
		})

		require.NoError(t, err)
		assert.Equal(t, int64(2), result.Total)
	})
}

func TestGetOrganization(t *testing.T) {
	t.Run("should return organization when found", func(t *testing.T) {
		db := newMockDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org", Slug: "test-org"}

		svc := NewService(db)
		org, err := svc.GetOrganization(context.Background(), 1)

		require.NoError(t, err)
		assert.Equal(t, int64(1), org.ID)
		assert.Equal(t, "Test Org", org.Name)
	})

	t.Run("should return error when organization not found", func(t *testing.T) {
		db := newMockDB()

		svc := NewService(db)
		org, err := svc.GetOrganization(context.Background(), 999)

		assert.Error(t, err)
		assert.Equal(t, ErrOrganizationNotFound, err)
		assert.Nil(t, org)
	})
}

func TestGetOrganizationWithMembers(t *testing.T) {
	t.Run("should return organization with members", func(t *testing.T) {
		db := newMockDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.members = []organization.Member{
			{ID: 1, UserID: 100, OrganizationID: 1, Role: "owner"},
			{ID: 2, UserID: 101, OrganizationID: 1, Role: "member"},
		}

		svc := NewService(db)
		org, members, err := svc.GetOrganizationWithMembers(context.Background(), 1)

		require.NoError(t, err)
		assert.NotNil(t, org)
		assert.Len(t, members, 2)
	})
}

func TestDeleteOrganization(t *testing.T) {
	t.Run("should delete organization successfully when no runners", func(t *testing.T) {
		db := newMockDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.runnerCount = 0 // Use runnerCount for Model(&runner.Runner{}).Where("organization_id = ?", ...).Count()

		svc := NewService(db)
		err := svc.DeleteOrganization(context.Background(), 1)

		require.NoError(t, err)
	})

	t.Run("should return error when organization has runners", func(t *testing.T) {
		db := newMockDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.runnerCount = 5 // Use runnerCount for Model(&runner.Runner{}).Where("organization_id = ?", ...).Count()

		svc := NewService(db)
		err := svc.DeleteOrganization(context.Background(), 1)

		assert.Error(t, err)
		assert.Equal(t, ErrOrganizationHasActiveRunner, err)
	})

	t.Run("should return error when organization not found", func(t *testing.T) {
		db := newMockDB()

		svc := NewService(db)
		err := svc.DeleteOrganization(context.Background(), 999)

		assert.Error(t, err)
		assert.Equal(t, ErrOrganizationNotFound, err)
	})
}

func TestListOrganizations_WithFilters(t *testing.T) {
	t.Run("should filter by search term", func(t *testing.T) {
		db := newMockDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.totalOrgs = 1

		svc := NewService(db)
		result, err := svc.ListOrganizations(context.Background(), &OrganizationListQuery{
			Page:     1,
			PageSize: 20,
			Search:   "Test",
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("should return error when count fails", func(t *testing.T) {
		db := newMockDB()
		db.countErr = errors.New("count failed")

		svc := NewService(db)
		result, err := svc.ListOrganizations(context.Background(), &OrganizationListQuery{
			Page:     1,
			PageSize: 20,
		})

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("should return error when find fails", func(t *testing.T) {
		db := newMockDB()
		db.findErr = errors.New("find failed")

		svc := NewService(db)
		result, err := svc.ListOrganizations(context.Background(), &OrganizationListQuery{
			Page:     1,
			PageSize: 20,
		})

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestGetOrganizationWithMembers_ErrorPaths(t *testing.T) {
	t.Run("should return error when organization not found", func(t *testing.T) {
		db := newMockDB()

		svc := NewService(db)
		org, members, err := svc.GetOrganizationWithMembers(context.Background(), 999)

		assert.Error(t, err)
		assert.Equal(t, ErrOrganizationNotFound, err)
		assert.Nil(t, org)
		assert.Nil(t, members)
	})

	t.Run("should return error when find members fails", func(t *testing.T) {
		db := newMockDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.findErr = errors.New("find failed")

		svc := NewService(db)
		org, members, err := svc.GetOrganizationWithMembers(context.Background(), 1)

		assert.Error(t, err)
		assert.Nil(t, org)
		assert.Nil(t, members)
	})
}

func TestDeleteOrganization_ErrorPaths(t *testing.T) {
	t.Run("should return error when count runners fails", func(t *testing.T) {
		db := newMockDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.countErr = errors.New("count failed")

		svc := NewService(db)
		err := svc.DeleteOrganization(context.Background(), 1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check runners")
	})

	t.Run("should return error when delete fails", func(t *testing.T) {
		db := newMockDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.runnerCount = 0 // Use runnerCount for Model(&runner.Runner{}).Where("organization_id = ?", ...).Count()
		db.deleteErr = errors.New("delete failed")

		svc := NewService(db)
		err := svc.DeleteOrganization(context.Background(), 1)

		assert.Error(t, err)
	})
}
