package extensionconnect

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	extdom "github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	extensionv1 "github.com/anthropics/agentsmesh/proto/gen/go/extension/v1"
)

func TestListRepoSkills_MissingOrgSlug_InvalidArgument(t *testing.T) {
	srv := NewRepoSkillServer(NewServer(nil, &fakeOrgService{role: "member"}))
	_, err := srv.ListRepoSkills(ctxAsUser(42), connect.NewRequest(&extensionv1.ListRepoSkillsRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestListRepoSkills_NoAuth_Unauthenticated(t *testing.T) {
	srv := NewRepoSkillServer(NewServer(nil, &fakeOrgService{role: "member"}))
	_, err := srv.ListRepoSkills(context.Background(), connect.NewRequest(&extensionv1.ListRepoSkillsRequest{OrgSlug: "acme", RepositoryId: 7}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connectCodeOf(t, err))
}

func TestToProtoInstalledSkill_AllFieldsRoundTrip(t *testing.T) {
	orgID := int64(42)
	repoID := int64(7)
	marketItemID := int64(11)
	installedBy := int64(99)
	pinned := 2
	in := &extdom.InstalledSkill{
		ID:             1,
		OrganizationID: orgID,
		RepositoryID:   repoID,
		MarketItemID:   &marketItemID,
		Scope:          "user",
		InstalledBy:    &installedBy,
		Slug:           "format-go",
		InstallSource:  "market",
		SourceURL:      "",
		ContentSha:     "abc123",
		StorageKey:     "skills/format-go.zip",
		PackageSize:    4096,
		PinnedVersion:  &pinned,
		IsEnabled:      true,
		CreatedAt:      mustParseTime(t, "2026-05-01T00:00:00Z"),
		UpdatedAt:      mustParseTime(t, "2026-05-12T00:00:00Z"),
	}
	got := toProtoInstalledSkill(in)
	require.NotNil(t, got)
	assert.Equal(t, int64(1), got.GetId())
	assert.Equal(t, orgID, got.GetOrganizationId())
	assert.Equal(t, repoID, got.GetRepositoryId())
	assert.Equal(t, marketItemID, got.GetMarketItemId())
	assert.Equal(t, "user", got.GetScope())
	assert.Equal(t, installedBy, got.GetInstalledBy())
	assert.Equal(t, "format-go", got.GetSlug())
	assert.Equal(t, "market", got.GetInstallSource())
	assert.Equal(t, "abc123", got.GetContentSha())
	assert.Equal(t, int64(4096), got.GetPackageSize())
	assert.Equal(t, int32(2), got.GetPinnedVersion())
	assert.True(t, got.GetIsEnabled())
}

func TestToProtoInstalledSkill_GitHubInstall_NoMarketItem(t *testing.T) {
	in := &extdom.InstalledSkill{
		ID:             2,
		OrganizationID: 42,
		RepositoryID:   7,
		// MarketItemID nil → github install
		Scope:         "org",
		Slug:          "github-skill",
		InstallSource: "github",
		SourceURL:     "https://github.com/owner/repo",
		IsEnabled:     true,
		CreatedAt:     mustParseTime(t, "2026-05-01T00:00:00Z"),
		UpdatedAt:     mustParseTime(t, "2026-05-12T00:00:00Z"),
	}
	got := toProtoInstalledSkill(in)
	require.NotNil(t, got)
	assert.Nil(t, got.MarketItemId, "github installs must not carry market_item_id")
	assert.Nil(t, got.PinnedVersion, "no pinned version absent on the wire")
	assert.Equal(t, "https://github.com/owner/repo", got.GetSourceUrl())
}

func TestToProtoInstalledSkill_NilInputReturnsNil(t *testing.T) {
	assert.Nil(t, toProtoInstalledSkill(nil))
}
