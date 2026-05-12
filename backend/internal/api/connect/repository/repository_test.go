package repositoryconnect

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	repositoryservice "github.com/anthropics/agentsmesh/backend/internal/service/repository"
	repositoryv1 "github.com/anthropics/agentsmesh/proto/gen/go/repository/v1"
)

type fakeOrg struct {
	id   int64
	slug string
}

func (f fakeOrg) GetID() int64    { return f.id }
func (f fakeOrg) GetSlug() string { return f.slug }
func (f fakeOrg) GetName() string { return f.slug }

type fakeOrgService struct {
	role string
}

func (f *fakeOrgService) GetBySlug(_ context.Context, slug string) (middleware.OrganizationGetter, error) {
	if slug == "missing" {
		return nil, errors.New("org not found")
	}
	return fakeOrg{id: 7, slug: slug}, nil
}
func (f *fakeOrgService) IsMember(context.Context, int64, int64) (bool, error) { return true, nil }
func (f *fakeOrgService) GetMemberRole(context.Context, int64, int64) (string, error) {
	return f.role, nil
}

// fakeRepoService is the minimal stub the handler tests exercise — only the
// methods used by ResolveOrgScope-followed paths need real behavior.
type fakeRepoService struct {
	repositoryservice.RepositoryServiceInterface
	repos    []*gitprovider.Repository
	getByID  func(context.Context, int64) (*gitprovider.Repository, error)
	getBySlg func(context.Context, int64, string, string, string) (*gitprovider.Repository, error)
}

func (f *fakeRepoService) ListByOrganizationForUser(
	_ context.Context, _ int64, _ int64,
) ([]*gitprovider.Repository, error) {
	return f.repos, nil
}

func (f *fakeRepoService) GetByID(ctx context.Context, id int64) (*gitprovider.Repository, error) {
	if f.getByID != nil {
		return f.getByID(ctx, id)
	}
	return nil, repositoryservice.ErrRepositoryNotFound
}

func (f *fakeRepoService) GetBySlug(
	ctx context.Context, orgID int64, providerType, providerBaseURL, slug string,
) (*gitprovider.Repository, error) {
	if f.getBySlg != nil {
		return f.getBySlg(ctx, orgID, providerType, providerBaseURL, slug)
	}
	return nil, repositoryservice.ErrRepositoryNotFound
}

// ctxAsUser populates the auth-interceptor stand-in: UserID matters because
// ResolveOrgScope rejects empty TenantContext as Unauthenticated.
func ctxAsUser(userID int64) context.Context {
	return middleware.SetTenant(context.Background(), &middleware.TenantContext{UserID: userID})
}

// connectCodeOf is the canonical accessor for the Connect error code,
// independent of the test framework's error helpers.
func connectCodeOf(t *testing.T, err error) connect.Code {
	t.Helper()
	var ce *connect.Error
	require.True(t, errors.As(err, &ce), "expected *connect.Error, got %v", err)
	return ce.Code()
}

// --- ResolveOrgScope guards ---

func TestListRepositories_MissingOrgSlug_InvalidArgument(t *testing.T) {
	srv := NewServer(&fakeRepoService{}, &fakeOrgService{role: "admin"})
	_, err := srv.ListRepositories(ctxAsUser(42), connect.NewRequest(&repositoryv1.ListRepositoriesRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestListRepositories_NoAuth_Unauthenticated(t *testing.T) {
	srv := NewServer(&fakeRepoService{}, &fakeOrgService{role: "admin"})
	_, err := srv.ListRepositories(context.Background(), connect.NewRequest(&repositoryv1.ListRepositoriesRequest{OrgSlug: "acme"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connectCodeOf(t, err))
}

// --- requireAdmin gates (Create/Update/Delete) ---

func TestCreateRepository_NonAdmin_PermissionDenied(t *testing.T) {
	srv := NewServer(&fakeRepoService{}, &fakeOrgService{role: "member"})
	_, err := srv.CreateRepository(ctxAsUser(42), connect.NewRequest(&repositoryv1.CreateRepositoryRequest{
		OrgSlug: "acme", ProviderType: "github", ProviderBaseUrl: "https://github.com",
		ExternalId: "1", Name: "test", Slug: "test/test",
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connectCodeOf(t, err))
}

func TestUpdateRepository_NonAdmin_PermissionDenied(t *testing.T) {
	srv := NewServer(&fakeRepoService{}, &fakeOrgService{role: "member"})
	_, err := srv.UpdateRepository(ctxAsUser(42), connect.NewRequest(&repositoryv1.UpdateRepositoryRequest{
		OrgSlug: "acme", Id: 1,
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connectCodeOf(t, err))
}

func TestDeleteRepository_NonAdmin_PermissionDenied(t *testing.T) {
	srv := NewServer(&fakeRepoService{}, &fakeOrgService{role: "member"})
	_, err := srv.DeleteRepository(ctxAsUser(42), connect.NewRequest(&repositoryv1.DeleteRepositoryRequest{
		OrgSlug: "acme", Id: 1,
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connectCodeOf(t, err))
}

// --- mapServiceError table ---

func TestMapServiceError(t *testing.T) {
	cases := []struct {
		name string
		in   error
		want connect.Code
	}{
		{"not_found", repositoryservice.ErrRepositoryNotFound, connect.CodeNotFound},
		{"webhook_not_found", repositoryservice.ErrWebhookNotFound, connect.CodeNotFound},
		{"no_permission", repositoryservice.ErrNoPermission, connect.CodePermissionDenied},
		{"exists", repositoryservice.ErrRepositoryExists, connect.CodeAlreadyExists},
		{"has_loop_refs", repositoryservice.ErrRepositoryHasLoopRefs, connect.CodeFailedPrecondition},
		{"generic_error", errors.New("oops"), connect.CodeInternal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := connectCodeOf(t, mapServiceError(tc.in))
			assert.Equal(t, tc.want, got)
		})
	}
}

// --- toProtoRepository: every field round-trips into the proto message ---

func TestToProtoRepository_AllFieldsPopulated(t *testing.T) {
	importedBy := int64(11)
	timeout := 600
	tprefix := "PROJ"
	prepScript := "make install"
	createdAt := mustParseTime(t, "2026-05-01T00:00:00Z")
	updatedAt := mustParseTime(t, "2026-05-10T00:00:00Z")

	r := &gitprovider.Repository{
		ID:                 7,
		OrganizationID:     42,
		ProviderType:       "github",
		ProviderBaseURL:    "https://github.com",
		HttpCloneURL:       "https://github.com/acme/api.git",
		SshCloneURL:        "git@github.com:acme/api.git",
		ExternalID:         "100",
		Name:               "api",
		Slug:               "acme/api",
		DefaultBranch:      "main",
		TicketPrefix:       &tprefix,
		Visibility:         "organization",
		ImportedByUserID:   &importedBy,
		PreparationScript:  &prepScript,
		PreparationTimeout: &timeout,
		IsActive:           true,
		WebhookConfig: &gitprovider.WebhookConfig{
			ID:               "wh-1",
			URL:              "https://api.agentsmesh.local/webhook",
			Events:           []string{"push"},
			IsActive:         true,
			NeedsManualSetup: false,
			LastError:        "transient timeout",
			CreatedAt:        "2026-05-01T00:00:00Z",
		},
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	got := toProtoRepository(r)
	require.NotNil(t, got)
	assert.Equal(t, int64(7), got.GetId())
	assert.Equal(t, int64(42), got.GetOrganizationId())
	assert.Equal(t, "github", got.GetProviderType())
	assert.Equal(t, "https://github.com", got.GetProviderBaseUrl())
	assert.Equal(t, "https://github.com/acme/api.git", got.GetHttpCloneUrl())
	assert.Equal(t, "git@github.com:acme/api.git", got.GetSshCloneUrl())
	assert.Equal(t, "100", got.GetExternalId())
	assert.Equal(t, "api", got.GetName())
	assert.Equal(t, "acme/api", got.GetSlug())
	assert.Equal(t, "main", got.GetDefaultBranch())
	assert.Equal(t, "PROJ", got.GetTicketPrefix())
	assert.Equal(t, "organization", got.GetVisibility())
	assert.Equal(t, int64(11), got.GetImportedByUserId())
	assert.Equal(t, "make install", got.GetPreparationScript())
	assert.Equal(t, int32(600), got.GetPreparationTimeout())
	assert.True(t, got.GetIsActive())
	assert.Equal(t, "2026-05-01T00:00:00Z", got.GetCreatedAt())
	assert.Equal(t, "2026-05-10T00:00:00Z", got.GetUpdatedAt())
	require.NotNil(t, got.GetWebhookConfig())
	assert.Equal(t, "wh-1", got.GetWebhookConfig().GetId())
	assert.Equal(t, "https://api.agentsmesh.local/webhook", got.GetWebhookConfig().GetUrl())
	assert.Equal(t, []string{"push"}, got.GetWebhookConfig().GetEvents())
	assert.True(t, got.GetWebhookConfig().GetIsActive())
	assert.False(t, got.GetWebhookConfig().GetNeedsManualSetup())
	assert.Equal(t, "transient timeout", got.GetWebhookConfig().GetLastError())
	assert.Equal(t, "2026-05-01T00:00:00Z", got.GetWebhookConfig().GetCreatedAt())
}

func TestToProtoRepository_MinimalFields_OptionalsAbsent(t *testing.T) {
	r := &gitprovider.Repository{
		ID:              1,
		OrganizationID:  1,
		ProviderType:    "github",
		ProviderBaseURL: "https://github.com",
		ExternalID:      "x",
		Name:            "n",
		Slug:            "n",
		DefaultBranch:   "main",
		Visibility:      "organization",
		IsActive:        true,
		CreatedAt:       mustParseTime(t, "2026-05-01T00:00:00Z"),
		UpdatedAt:       mustParseTime(t, "2026-05-01T00:00:00Z"),
	}
	got := toProtoRepository(r)
	require.NotNil(t, got)
	assert.Nil(t, got.TicketPrefix)
	assert.Nil(t, got.ImportedByUserId)
	assert.Nil(t, got.PreparationScript)
	assert.Nil(t, got.PreparationTimeout)
	assert.Nil(t, got.WebhookConfig)
}

func TestToProtoRepository_NilInput_ReturnsNil(t *testing.T) {
	assert.Nil(t, toProtoRepository(nil))
}

// --- toProtoWebhookStatus / toProtoWebhookResult / toProtoMergeRequest ---

func TestToProtoWebhookStatus_AllFields(t *testing.T) {
	s := &gitprovider.WebhookStatus{
		Registered:       true,
		WebhookID:        "wh-1",
		WebhookURL:       "https://example/webhook",
		Events:           []string{"push", "pull_request"},
		IsActive:         true,
		NeedsManualSetup: false,
		LastError:        "rate limit",
		RegisteredAt:     "2026-05-01T00:00:00Z",
	}
	got := toProtoWebhookStatus(s)
	require.NotNil(t, got)
	assert.True(t, got.GetRegistered())
	assert.Equal(t, "wh-1", got.GetWebhookId())
	assert.Equal(t, "https://example/webhook", got.GetWebhookUrl())
	assert.Equal(t, []string{"push", "pull_request"}, got.GetEvents())
	assert.True(t, got.GetIsActive())
	assert.False(t, got.GetNeedsManualSetup())
	assert.Equal(t, "rate limit", got.GetLastError())
	assert.Equal(t, "2026-05-01T00:00:00Z", got.GetRegisteredAt())
}

func TestToProtoWebhookStatus_NilInput_ReturnsEmpty(t *testing.T) {
	got := toProtoWebhookStatus(nil)
	require.NotNil(t, got)
	assert.False(t, got.GetRegistered())
}

func TestToProtoWebhookResult_ManualSetup(t *testing.T) {
	r := &repositoryservice.WebhookResult{
		RepoID:              7,
		Registered:          false,
		WebhookID:           "",
		NeedsManualSetup:    true,
		ManualWebhookURL:    "https://example/wh",
		ManualWebhookSecret: "secret-token",
		Error:               "",
	}
	got := toProtoWebhookResult(r)
	require.NotNil(t, got)
	assert.Equal(t, int64(7), got.GetRepoId())
	assert.False(t, got.GetRegistered())
	assert.True(t, got.GetNeedsManualSetup())
	assert.Equal(t, "https://example/wh", got.GetManualWebhookUrl())
	assert.Equal(t, "secret-token", got.GetManualWebhookSecret())
}

func TestToProtoMergeRequest_AllFields(t *testing.T) {
	pipelineID := int64(10)
	pipelineStatus := "running"
	pipelineURL := "https://gitlab/pipelines/10"
	ticketID := int64(99)
	podID := int64(123)
	mr := &repositoryservice.MergeRequestInfo{
		ID:             1, MRIID: 5, Title: "fix", State: "opened",
		MRURL: "https://gitlab/mr/5",
		SourceBranch: "feat", TargetBranch: "main",
		PipelineStatus: &pipelineStatus,
		PipelineID:     &pipelineID,
		PipelineURL:    &pipelineURL,
		TicketID:       &ticketID,
		PodID:          &podID,
	}
	got := toProtoMergeRequest(mr)
	require.NotNil(t, got)
	assert.Equal(t, int64(1), got.GetId())
	assert.Equal(t, int32(5), got.GetMrIid())
	assert.Equal(t, "fix", got.GetTitle())
	assert.Equal(t, "opened", got.GetState())
	assert.Equal(t, "feat", got.GetSourceBranch())
	assert.Equal(t, "main", got.GetTargetBranch())
	assert.Equal(t, "running", got.GetPipelineStatus())
	assert.Equal(t, int64(10), got.GetPipelineId())
	assert.Equal(t, "https://gitlab/pipelines/10", got.GetPipelineUrl())
	assert.Equal(t, int64(99), got.GetTicketId())
	assert.Equal(t, int64(123), got.GetPodId())
}

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	tt, err := time.Parse(time.RFC3339, s)
	require.NoError(t, err)
	return tt
}
