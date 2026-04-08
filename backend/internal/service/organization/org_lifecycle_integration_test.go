package organization

import (
	"context"
	"testing"

	orgDomain "github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestOrgService creates an organization service backed by real SQLite via testkit.
func newTestOrgService(t *testing.T) (*Service, func(email, username string) int64) {
	t.Helper()
	db := testkit.SetupTestDB(t)
	svc := NewService(infra.NewOrganizationRepository(db))
	addUser := func(email, username string) int64 {
		return testkit.CreateUser(t, db, email, username)
	}
	return svc, addUser
}

func TestOrg_CreateWithOwner(t *testing.T) {
	svc, addUser := newTestOrgService(t)
	ctx := context.Background()

	ownerID := addUser("owner@example.com", "owner")

	org, err := svc.Create(ctx, ownerID, &CreateRequest{
		Name: "Acme Corp",
		Slug: "acme",
	})
	require.NoError(t, err)
	require.NotNil(t, org)
	assert.Greater(t, org.ID, int64(0))
	assert.Equal(t, "Acme Corp", org.Name)
	assert.Equal(t, "acme", org.Slug)

	// Verify the owner is automatically a member with role "owner"
	member, err := svc.GetMember(ctx, org.ID, ownerID)
	require.NoError(t, err)
	assert.Equal(t, orgDomain.RoleOwner, member.Role)
}

func TestOrg_AddMemberChangeRole(t *testing.T) {
	svc, addUser := newTestOrgService(t)
	ctx := context.Background()

	ownerID := addUser("owner@example.com", "owner")
	memberID := addUser("member@example.com", "member")

	org, err := svc.Create(ctx, ownerID, &CreateRequest{
		Name: "Team X", Slug: "team-x",
	})
	require.NoError(t, err)

	// Add member
	err = svc.AddMember(ctx, org.ID, memberID, orgDomain.RoleMember)
	require.NoError(t, err)

	member, err := svc.GetMember(ctx, org.ID, memberID)
	require.NoError(t, err)
	assert.Equal(t, orgDomain.RoleMember, member.Role)

	// Promote to admin
	err = svc.UpdateMemberRole(ctx, org.ID, memberID, orgDomain.RoleAdmin)
	require.NoError(t, err)

	member, err = svc.GetMember(ctx, org.ID, memberID)
	require.NoError(t, err)
	assert.Equal(t, orgDomain.RoleAdmin, member.Role)

	// Verify IsAdmin returns true
	isAdmin, err := svc.IsAdmin(ctx, org.ID, memberID)
	require.NoError(t, err)
	assert.True(t, isAdmin)

	// Verify IsOwner returns false for a non-owner
	isOwner, err := svc.IsOwner(ctx, org.ID, memberID)
	require.NoError(t, err)
	assert.False(t, isOwner)
}

func TestOrg_CannotRemoveOwner(t *testing.T) {
	svc, addUser := newTestOrgService(t)
	ctx := context.Background()

	ownerID := addUser("owner@example.com", "owner")

	org, err := svc.Create(ctx, ownerID, &CreateRequest{
		Name: "Protected Org", Slug: "protected",
	})
	require.NoError(t, err)

	err = svc.RemoveMember(ctx, org.ID, ownerID)
	assert.ErrorIs(t, err, ErrCannotRemoveOwner)

	// Owner should still be there
	member, err := svc.GetMember(ctx, org.ID, ownerID)
	require.NoError(t, err)
	assert.Equal(t, orgDomain.RoleOwner, member.Role)
}

func TestOrg_DeleteRemovesOrg(t *testing.T) {
	svc, addUser := newTestOrgService(t)
	ctx := context.Background()

	ownerID := addUser("owner@example.com", "owner")
	memberID := addUser("m@example.com", "m")

	org, err := svc.Create(ctx, ownerID, &CreateRequest{
		Name: "Doomed Org", Slug: "doomed",
	})
	require.NoError(t, err)

	// Add a member
	err = svc.AddMember(ctx, org.ID, memberID, orgDomain.RoleMember)
	require.NoError(t, err)

	// Verify members exist before delete
	members, err := svc.ListMembers(ctx, org.ID)
	require.NoError(t, err)
	assert.Len(t, members, 2)

	// Delete the org
	err = svc.Delete(ctx, org.ID)
	require.NoError(t, err)

	// Org should be gone
	_, err = svc.GetByID(ctx, org.ID)
	assert.ErrorIs(t, err, ErrOrganizationNotFound)

	// Slug should be available for reuse
	org2, err := svc.Create(ctx, ownerID, &CreateRequest{
		Name: "Reborn Org", Slug: "doomed",
	})
	require.NoError(t, err)
	assert.Equal(t, "doomed", org2.Slug)
}

func TestOrg_DuplicateSlug(t *testing.T) {
	svc, addUser := newTestOrgService(t)
	ctx := context.Background()

	ownerID := addUser("o@example.com", "o")

	_, err := svc.Create(ctx, ownerID, &CreateRequest{Name: "A", Slug: "unique"})
	require.NoError(t, err)

	_, err = svc.Create(ctx, ownerID, &CreateRequest{Name: "B", Slug: "unique"})
	assert.ErrorIs(t, err, ErrSlugAlreadyExists)
}

func TestOrg_ListByUser(t *testing.T) {
	svc, addUser := newTestOrgService(t)
	ctx := context.Background()

	user1 := addUser("u1@example.com", "u1")
	user2 := addUser("u2@example.com", "u2")

	org1, err := svc.Create(ctx, user1, &CreateRequest{Name: "Org1", Slug: "org1"})
	require.NoError(t, err)

	_, err = svc.Create(ctx, user2, &CreateRequest{Name: "Org2", Slug: "org2"})
	require.NoError(t, err)

	org3, err := svc.Create(ctx, user1, &CreateRequest{Name: "Org3", Slug: "org3"})
	require.NoError(t, err)

	// user1 should belong to org1 and org3 (as owner of both)
	orgs, err := svc.ListByUser(ctx, user1)
	require.NoError(t, err)
	assert.Len(t, orgs, 2)

	ids := []int64{orgs[0].ID, orgs[1].ID}
	assert.Contains(t, ids, org1.ID)
	assert.Contains(t, ids, org3.ID)
}

func TestOrg_RemoveMemberThenIsMember(t *testing.T) {
	svc, addUser := newTestOrgService(t)
	ctx := context.Background()

	ownerID := addUser("o@example.com", "o")
	memberID := addUser("m@example.com", "m")

	org, err := svc.Create(ctx, ownerID, &CreateRequest{Name: "R", Slug: "r"})
	require.NoError(t, err)

	err = svc.AddMember(ctx, org.ID, memberID, orgDomain.RoleMember)
	require.NoError(t, err)

	isMember, err := svc.IsMember(ctx, org.ID, memberID)
	require.NoError(t, err)
	assert.True(t, isMember)

	err = svc.RemoveMember(ctx, org.ID, memberID)
	require.NoError(t, err)

	isMember, err = svc.IsMember(ctx, org.ID, memberID)
	require.NoError(t, err)
	assert.False(t, isMember)
}

func TestOrg_UpdateOrganization(t *testing.T) {
	svc, addUser := newTestOrgService(t)
	ctx := context.Background()

	ownerID := addUser("o@example.com", "o")
	org, err := svc.Create(ctx, ownerID, &CreateRequest{
		Name: "Old Name", Slug: "upd",
	})
	require.NoError(t, err)

	updated, err := svc.Update(ctx, org.ID, map[string]interface{}{"name": "New Name"})
	require.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
	assert.Equal(t, "upd", updated.Slug) // slug unchanged
}
