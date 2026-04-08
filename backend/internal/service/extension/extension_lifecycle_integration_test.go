package extension

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// testSetup holds common test dependencies for extension integration tests.
type testSetup struct {
	svc    *Service
	ctx    context.Context
	db     *gorm.DB
	orgID  int64
	userID int64
	repoID int64
}

// newIntegrationSetup creates a Service backed by the shared testutil DB.
func newIntegrationSetup(t *testing.T) *testSetup {
	t.Helper()
	db := testkit.SetupTestDB(t)
	repo := infra.NewExtensionRepository(db)
	svc := NewService(repo, nil, nil) // no storage/crypto for these tests

	ctx := context.Background()
	userID := testkit.CreateUser(t, db, "ext-user@test.com", "ext-user")
	orgID := testkit.CreateOrg(t, db, "ext-org", userID)
	repoID := testkit.CreateRepo(t, db, orgID, "ext-org/demo-repo", "https://github.com/ext-org/demo-repo.git")

	return &testSetup{svc: svc, ctx: ctx, db: db, orgID: orgID, userID: userID, repoID: repoID}
}

func TestExtension_McpServerCRUD(t *testing.T) {
	ts := newIntegrationSetup(t)

	// Install custom MCP server
	server := &extension.InstalledMcpServer{
		Slug:          "my-mcp-server",
		Name:          "My MCP Server",
		Scope:         extension.ScopeOrg,
		TransportType: extension.TransportTypeStdio,
		Command:       "npx",
	}
	installed, err := ts.svc.InstallCustomMcpServer(ts.ctx, ts.orgID, ts.repoID, ts.userID, server, nil)
	require.NoError(t, err)
	require.NotNil(t, installed)
	assert.NotZero(t, installed.ID)
	assert.Equal(t, "my-mcp-server", installed.Slug)
	assert.Equal(t, ts.orgID, installed.OrganizationID)
	assert.Equal(t, ts.repoID, installed.RepositoryID)
	assert.True(t, installed.IsEnabled)

	// List MCP servers
	servers, err := ts.svc.ListRepoMcpServers(ts.ctx, ts.orgID, ts.repoID, ts.userID, "all")
	require.NoError(t, err)
	require.Len(t, servers, 1)
	assert.Equal(t, "my-mcp-server", servers[0].Slug)

	// Update (disable)
	disabled := false
	updated, err := ts.svc.UpdateMcpServer(ts.ctx, ts.orgID, ts.repoID, installed.ID, ts.userID, "owner", &disabled, nil)
	require.NoError(t, err)
	assert.False(t, updated.IsEnabled)

	// Uninstall
	err = ts.svc.UninstallMcpServer(ts.ctx, ts.orgID, ts.repoID, installed.ID, ts.userID, "owner")
	require.NoError(t, err)

	servers, err = ts.svc.ListRepoMcpServers(ts.ctx, ts.orgID, ts.repoID, ts.userID, "all")
	require.NoError(t, err)
	assert.Empty(t, servers)
}

func TestExtension_SkillRegistryCRUD(t *testing.T) {
	ts := newIntegrationSetup(t)

	// Create registry
	registry, err := ts.svc.CreateSkillRegistry(ts.ctx, ts.orgID, CreateSkillRegistryInput{
		RepositoryURL: "https://github.com/example/skill-registry",
		Branch:        "main",
	})
	require.NoError(t, err)
	require.NotNil(t, registry)
	assert.NotZero(t, registry.ID)
	assert.Equal(t, "https://github.com/example/skill-registry", registry.RepositoryURL)
	assert.Equal(t, "main", registry.Branch)
	assert.Equal(t, "pending", registry.SyncStatus)
	require.NotNil(t, registry.OrganizationID)
	assert.Equal(t, ts.orgID, *registry.OrganizationID)

	// List registries
	registries, err := ts.svc.ListSkillRegistries(ts.ctx, ts.orgID)
	require.NoError(t, err)
	require.Len(t, registries, 1)
	assert.Equal(t, registry.ID, registries[0].ID)

	// Delete registry
	err = ts.svc.DeleteSkillRegistry(ts.ctx, ts.orgID, registry.ID)
	require.NoError(t, err)

	registries, err = ts.svc.ListSkillRegistries(ts.ctx, ts.orgID)
	require.NoError(t, err)
	assert.Empty(t, registries)
}

func TestExtension_OrgIsolation(t *testing.T) {
	ts := newIntegrationSetup(t)

	// Create second org
	user2ID := testkit.CreateUser(t, ts.db, "bob@test.com", "bob")
	org2ID := testkit.CreateOrg(t, ts.db, "other-org", user2ID)
	repo2ID := testkit.CreateRepo(t, ts.db, org2ID, "other-org/other-repo", "https://github.com/other-org/other.git")

	// Org1 installs MCP server
	server := &extension.InstalledMcpServer{
		Slug:          "secret-tool",
		Name:          "Secret Tool",
		Scope:         extension.ScopeOrg,
		TransportType: extension.TransportTypeStdio,
		Command:       "secret-tool",
	}
	installed, err := ts.svc.InstallCustomMcpServer(ts.ctx, ts.orgID, ts.repoID, ts.userID, server, nil)
	require.NoError(t, err)

	// Org1 can see its server
	org1Servers, err := ts.svc.ListRepoMcpServers(ts.ctx, ts.orgID, ts.repoID, ts.userID, "all")
	require.NoError(t, err)
	assert.Len(t, org1Servers, 1)

	// Org2 cannot see org1's server
	org2Servers, err := ts.svc.ListRepoMcpServers(ts.ctx, org2ID, repo2ID, user2ID, "all")
	require.NoError(t, err)
	assert.Empty(t, org2Servers)

	// Org2 cannot update org1's server
	enabled := true
	_, err = ts.svc.UpdateMcpServer(ts.ctx, org2ID, repo2ID, installed.ID, user2ID, "owner", &enabled, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not belong to this organization")

	// Org2 cannot uninstall org1's server
	err = ts.svc.UninstallMcpServer(ts.ctx, org2ID, repo2ID, installed.ID, user2ID, "owner")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not belong to this organization")

	// Org1 creates skill registry, org2 cannot delete it
	registry, err := ts.svc.CreateSkillRegistry(ts.ctx, ts.orgID, CreateSkillRegistryInput{
		RepositoryURL: "https://github.com/org1/skills",
	})
	require.NoError(t, err)

	err = ts.svc.DeleteSkillRegistry(ts.ctx, org2ID, registry.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not belong to this organization")
}
