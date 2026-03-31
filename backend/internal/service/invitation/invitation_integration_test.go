package invitation

import (
	"context"
	"testing"
	"time"

	invitationDomain "github.com/anthropics/agentsmesh/backend/internal/domain/invitation"
	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestService creates a Service backed by an in-memory SQLite DB with
// seed data: one user (inviter), one org, and the inviter as org owner.
func setupTestService(t *testing.T) (*Service, context.Context, int64, int64) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	repo := infra.NewInvitationRepository(db)
	svc := NewService(repo, nil) // nil email service — no emails in tests

	inviterID := testutil.CreateUser(t, db, "inviter@test.com", "inviter")
	orgID := testutil.CreateOrg(t, db, "test-org", inviterID)

	return svc, context.Background(), orgID, inviterID
}

func TestInvitation_CreateAndQuery(t *testing.T) {
	svc, ctx, orgID, inviterID := setupTestService(t)

	inv, err := svc.Create(ctx, &CreateRequest{
		OrganizationID: orgID,
		Email:          "new-member@test.com",
		Role:           organization.RoleMember,
		InviterID:      inviterID,
		InviterName:    "inviter",
		OrgName:        "test-org",
	})
	require.NoError(t, err)
	require.NotNil(t, inv)

	// Token should be a 64-char hex string (32 random bytes).
	assert.Len(t, inv.Token, 64)
	assert.Equal(t, orgID, inv.OrganizationID)
	assert.Equal(t, "new-member@test.com", inv.Email)
	assert.Equal(t, organization.RoleMember, inv.Role)
	assert.Nil(t, inv.AcceptedAt)
	assert.True(t, inv.ExpiresAt.After(time.Now()))

	// Query back by token
	fetched, err := svc.GetByToken(ctx, inv.Token)
	require.NoError(t, err)
	assert.Equal(t, inv.ID, fetched.ID)

	// Query back by ID
	fetched2, err := svc.GetByID(ctx, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, inv.Token, fetched2.Token)

	// List by organization
	list, err := svc.ListByOrganization(ctx, orgID)
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestInvitation_AcceptInvitation(t *testing.T) {
	svc, ctx, orgID, inviterID := setupTestService(t)

	inv, err := svc.Create(ctx, &CreateRequest{
		OrganizationID: orgID,
		Email:          "joiner@test.com",
		Role:           organization.RoleMember,
		InviterID:      inviterID,
		InviterName:    "inviter",
		OrgName:        "test-org",
	})
	require.NoError(t, err)

	// Create a second user to accept the invitation
	db := testutil.SetupTestDB(t) // need a fresh user; reuse setupTestService's DB via repo
	_ = db                        // not used — we test via service only

	// We need the accepting user in the same DB. Rebuild with full setup.
	db2 := testutil.SetupTestDB(t)
	repo2 := infra.NewInvitationRepository(db2)
	svc2 := NewService(repo2, nil)
	inviterID2 := testutil.CreateUser(t, db2, "inviter2@test.com", "inviter2")
	orgID2 := testutil.CreateOrg(t, db2, "org2", inviterID2)
	joinerID := testutil.CreateUser(t, db2, "joiner2@test.com", "joiner2")

	inv2, err := svc2.Create(ctx, &CreateRequest{
		OrganizationID: orgID2,
		Email:          "joiner2@test.com",
		Role:           organization.RoleMember,
		InviterID:      inviterID2,
		InviterName:    "inviter2",
		OrgName:        "org2",
	})
	require.NoError(t, err)

	result, err := svc2.Accept(ctx, inv2.Token, joinerID)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, orgID2, result.Organization.ID)

	// Verify the invitation is now accepted — trying again should fail
	_, err = svc2.Accept(ctx, inv2.Token, joinerID)
	assert.ErrorIs(t, err, ErrInvitationAccepted)

	// The original invitation from svc should still be independent
	_ = inv
}

func TestInvitation_ExpiredInvitation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := infra.NewInvitationRepository(db)
	svc := NewService(repo, nil)
	ctx := context.Background()

	inviterID := testutil.CreateUser(t, db, "inviter@test.com", "inviter")
	orgID := testutil.CreateOrg(t, db, "test-org", inviterID)
	joinerID := testutil.CreateUser(t, db, "joiner@test.com", "joiner")

	// Directly insert an already-expired invitation
	expired := &invitationDomain.Invitation{
		OrganizationID: orgID,
		Email:          "joiner@test.com",
		Role:           organization.RoleMember,
		Token:          "expired-token-abc123",
		InvitedBy:      inviterID,
		ExpiresAt:      time.Now().Add(-24 * time.Hour), // expired yesterday
	}
	err := db.Create(expired).Error
	require.NoError(t, err)

	// Accepting should fail with ErrInvitationExpired
	_, err = svc.Accept(ctx, expired.Token, joinerID)
	assert.ErrorIs(t, err, ErrInvitationExpired)
}

func TestInvitation_InvalidRole(t *testing.T) {
	svc, ctx, orgID, inviterID := setupTestService(t)

	_, err := svc.Create(ctx, &CreateRequest{
		OrganizationID: orgID,
		Email:          "bad-role@test.com",
		Role:           "superuser",
		InviterID:      inviterID,
	})
	assert.ErrorIs(t, err, ErrInvalidRole)
}

func TestInvitation_DuplicatePending(t *testing.T) {
	svc, ctx, orgID, inviterID := setupTestService(t)

	_, err := svc.Create(ctx, &CreateRequest{
		OrganizationID: orgID,
		Email:          "dup@test.com",
		Role:           organization.RoleMember,
		InviterID:      inviterID,
		InviterName:    "inviter",
		OrgName:        "test-org",
	})
	require.NoError(t, err)

	// Creating a second invitation for the same email should fail
	_, err = svc.Create(ctx, &CreateRequest{
		OrganizationID: orgID,
		Email:          "dup@test.com",
		Role:           organization.RoleMember,
		InviterID:      inviterID,
		InviterName:    "inviter",
		OrgName:        "test-org",
	})
	assert.ErrorIs(t, err, ErrPendingInvitation)
}

func TestInvitation_Revoke(t *testing.T) {
	svc, ctx, orgID, inviterID := setupTestService(t)

	inv, err := svc.Create(ctx, &CreateRequest{
		OrganizationID: orgID,
		Email:          "revoke@test.com",
		Role:           organization.RoleMember,
		InviterID:      inviterID,
		InviterName:    "inviter",
		OrgName:        "test-org",
	})
	require.NoError(t, err)

	err = svc.Revoke(ctx, inv.ID)
	require.NoError(t, err)

	// After revocation, token lookup should fail
	_, err = svc.GetByID(ctx, inv.ID)
	assert.ErrorIs(t, err, ErrInvitationNotFound)
}
