package repository

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	db := testkit.SetupTestDB(t)
	return NewService(infra.NewGitProviderRepository(db))
}

func fullCreateRequest(orgID int64) *CreateRequest {
	return &CreateRequest{
		OrganizationID:  orgID,
		ProviderType:    "github",
		ProviderBaseURL: "https://github.com",
		HttpCloneURL:    "https://github.com/acme/widget.git",
		SshCloneURL:     "git@github.com:acme/widget.git",
		ExternalID:      "ext-111",
		Name:            "widget",
		Slug:            "acme/widget",
		DefaultBranch:   "develop",
		Visibility:      "organization",
	}
}

func TestRepo_CreateAndGet(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	req := fullCreateRequest(1)
	created, err := svc.Create(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "widget", created.Name)
	assert.Equal(t, "develop", created.DefaultBranch)

	// GetByID
	got, err := svc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "acme/widget", got.Slug)

	// GetBySlug
	bySlug, err := svc.GetBySlug(ctx, 1, "github", "https://github.com", "acme/widget")
	require.NoError(t, err)
	assert.Equal(t, created.ID, bySlug.ID)

	// GetByExternalID
	byExt, err := svc.GetByExternalID(ctx, "github", "https://github.com", "ext-111")
	require.NoError(t, err)
	assert.Equal(t, created.ID, byExt.ID)

	// FindByOrgSlug
	byOrgSlug, err := svc.FindByOrgSlug(ctx, 1, "acme/widget")
	require.NoError(t, err)
	require.NotNil(t, byOrgSlug)
	assert.Equal(t, created.ID, byOrgSlug.ID)
}

func TestRepo_IdempotentCreate(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	req := fullCreateRequest(1)
	first, err := svc.Create(ctx, req)
	require.NoError(t, err)

	// Second create with same org+provider+slug should update, not error
	req.Name = "widget-v2"
	req.DefaultBranch = "main"
	second, err := svc.Create(ctx, req)
	require.NoError(t, err)

	assert.Equal(t, first.ID, second.ID, "idempotent import must keep same ID")
	assert.Equal(t, "widget-v2", second.Name, "name should be updated")
}

func TestRepo_Update(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, fullCreateRequest(1))
	require.NoError(t, err)

	script := "make deps"
	updated, err := svc.Update(ctx, created.ID, map[string]interface{}{
		"default_branch":     "main",
		"preparation_script": script,
	})
	require.NoError(t, err)
	assert.Equal(t, "main", updated.DefaultBranch)
	assert.Equal(t, &script, updated.PreparationScript)

	// Re-fetch to confirm persistence
	fetched, err := svc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "main", fetched.DefaultBranch)
}

func TestRepo_SoftDeleteAndHardDelete(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, fullCreateRequest(1))
	require.NoError(t, err)

	// Soft delete
	require.NoError(t, svc.Delete(ctx, created.ID))

	// ListByOrganization should exclude soft-deleted
	list, err := svc.ListByOrganization(ctx, 1)
	require.NoError(t, err)
	assert.Empty(t, list)

	// GetByID should fail (soft-deleted)
	_, err = svc.GetByID(ctx, created.ID)
	assert.ErrorIs(t, err, ErrRepositoryNotFound)

	// Create a new repo to hard delete
	req2 := fullCreateRequest(1)
	req2.Slug = "acme/other"
	req2.ExternalID = "ext-222"
	created2, err := svc.Create(ctx, req2)
	require.NoError(t, err)

	require.NoError(t, svc.HardDelete(ctx, created2.ID))

	_, err = svc.GetByID(ctx, created2.ID)
	assert.ErrorIs(t, err, ErrRepositoryNotFound)
}

func TestRepo_DeleteBlockedByLoop(t *testing.T) {
	db := testkit.SetupTestDB(t)
	svc := NewService(infra.NewGitProviderRepository(db))
	ctx := context.Background()

	created, err := svc.Create(ctx, fullCreateRequest(1))
	require.NoError(t, err)

	// Insert a loop referencing this repository
	err = db.Exec(
		`INSERT INTO loops (organization_id, name, slug, created_by_id, prompt_template, repository_id)
		 VALUES (1, 'ci-loop', 'ci-loop', 1, 'run tests', ?)`, created.ID,
	).Error
	require.NoError(t, err)

	// Soft delete should be blocked
	err = svc.Delete(ctx, created.ID)
	assert.ErrorIs(t, err, ErrRepositoryHasLoopRefs)

	// Hard delete should also be blocked
	err = svc.HardDelete(ctx, created.ID)
	assert.ErrorIs(t, err, ErrRepositoryHasLoopRefs)
}

func TestRepo_VisibilityFiltering(t *testing.T) {
	db := testkit.SetupTestDB(t)
	svc := NewService(infra.NewGitProviderRepository(db))
	ctx := context.Background()

	ownerID := int64(10)

	// Public repo
	pubReq := fullCreateRequest(1)
	pubReq.Visibility = "organization"
	_, err := svc.Create(ctx, pubReq)
	require.NoError(t, err)

	// Private repo imported by ownerID
	privReq := &CreateRequest{
		OrganizationID:   1,
		ProviderType:     "github",
		ProviderBaseURL:  "https://github.com",
		HttpCloneURL:     "https://github.com/acme/secret.git",
		ExternalID:       "ext-priv",
		Name:             "secret",
		Slug:             "acme/secret",
		Visibility:       "private",
		ImportedByUserID: &ownerID,
	}
	_, err = svc.Create(ctx, privReq)
	require.NoError(t, err)

	// Owner sees both
	ownerList, err := svc.ListByOrganizationForUser(ctx, 1, ownerID)
	require.NoError(t, err)
	assert.Len(t, ownerList, 2)

	// Other user sees only the public one
	otherList, err := svc.ListByOrganizationForUser(ctx, 1, 99)
	require.NoError(t, err)
	assert.Len(t, otherList, 1)
	assert.Equal(t, "organization", otherList[0].Visibility)
}

func TestRepo_FindByOrgSlug(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// GitHub repo
	gh := fullCreateRequest(1)
	_, err := svc.Create(ctx, gh)
	require.NoError(t, err)

	// GitLab repo with the same slug but different provider
	gl := &CreateRequest{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		HttpCloneURL:    "https://gitlab.com/acme/widget.git",
		ExternalID:      "gl-111",
		Name:            "widget-gl",
		Slug:            "acme/widget",
		Visibility:      "organization",
	}
	_, err = svc.Create(ctx, gl)
	require.NoError(t, err)

	// FindByOrgSlug returns first match regardless of provider
	found, err := svc.FindByOrgSlug(ctx, 1, "acme/widget")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "acme/widget", found.Slug)
}
