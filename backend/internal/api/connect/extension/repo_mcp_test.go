package extensionconnect

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	extdom "github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	extensionv1 "github.com/anthropics/agentsmesh/proto/gen/go/extension/v1"
)

func TestListRepoMcpServers_MissingOrgSlug_InvalidArgument(t *testing.T) {
	srv := NewRepoMcpServer(NewServer(nil, &fakeOrgService{role: "member"}))
	_, err := srv.ListRepoMcpServers(ctxAsUser(42), connect.NewRequest(&extensionv1.ListRepoMcpServersRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestInstallCustomMcpServer_InvalidTransportType(t *testing.T) {
	srv := NewRepoMcpServer(NewServer(nil, &fakeOrgService{role: "admin"}))
	_, err := srv.InstallCustomMcpServer(ctxAsUser(42), connect.NewRequest(&extensionv1.InstallCustomMcpServerRequest{
		OrgSlug:       "acme",
		RepositoryId:  7,
		Name:          "custom",
		Slug:          "custom-mcp",
		TransportType: "websocket", // not in the allowed list
		Scope:         "user",
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestInstallCustomMcpServer_InvalidArgsJSON(t *testing.T) {
	srv := NewRepoMcpServer(NewServer(nil, &fakeOrgService{role: "admin"}))
	bad := "{not valid json"
	_, err := srv.InstallCustomMcpServer(ctxAsUser(42), connect.NewRequest(&extensionv1.InstallCustomMcpServerRequest{
		OrgSlug:       "acme",
		RepositoryId:  7,
		Name:          "custom",
		Slug:          "custom-mcp",
		TransportType: "stdio",
		Args:          &bad,
		Scope:         "user",
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestInstallCustomMcpServer_ArgsCountExceeded(t *testing.T) {
	srv := NewRepoMcpServer(NewServer(nil, &fakeOrgService{role: "admin"}))
	// Build a JSON array with 51 entries (limit is 50)
	largeArgs := "["
	for i := 0; i < 51; i++ {
		if i > 0 {
			largeArgs += ","
		}
		largeArgs += "\"arg\""
	}
	largeArgs += "]"
	_, err := srv.InstallCustomMcpServer(ctxAsUser(42), connect.NewRequest(&extensionv1.InstallCustomMcpServerRequest{
		OrgSlug:       "acme",
		RepositoryId:  7,
		Name:          "custom",
		Slug:          "custom-mcp",
		TransportType: "stdio",
		Args:          &largeArgs,
		Scope:         "user",
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestDecodeEnvVars_Empty(t *testing.T) {
	m, err := decodeEnvVars("")
	require.NoError(t, err)
	assert.Nil(t, m)
}

func TestDecodeEnvVars_ValidJSONObject(t *testing.T) {
	m, err := decodeEnvVars(`{"FOO":"bar","BAZ":"qux"}`)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"FOO": "bar", "BAZ": "qux"}, m)
}

func TestDecodeEnvVars_InvalidJSON(t *testing.T) {
	_, err := decodeEnvVars("not json")
	require.Error(t, err)
}

func TestToProtoInstalledMcpServer_DecryptedEnvVarsPassedThrough(t *testing.T) {
	orgID := int64(42)
	installedBy := int64(99)
	marketItemID := int64(21)
	in := &extdom.InstalledMcpServer{
		ID:             1,
		OrganizationID: orgID,
		RepositoryID:   7,
		MarketItemID:   &marketItemID,
		Scope:          "org",
		InstalledBy:    &installedBy,
		Name:           "GitHub MCP",
		Slug:           "github",
		TransportType:  "stdio",
		Command:        "npx",
		Args:           []byte(`["@modelcontextprotocol/server-github"]`),
		HttpURL:        "",
		HttpHeaders:    []byte(`{}`),
		EnvVars:        []byte(`{"GITHUB_TOKEN":"ghp_xxx"}`),
		IsEnabled:      true,
		CreatedAt:      mustParseTime(t, "2026-05-01T00:00:00Z"),
		UpdatedAt:      mustParseTime(t, "2026-05-12T00:00:00Z"),
	}
	got := toProtoInstalledMcpServer(in)
	require.NotNil(t, got)
	assert.Equal(t, int64(1), got.GetId())
	assert.Equal(t, marketItemID, got.GetMarketItemId())
	assert.Equal(t, installedBy, got.GetInstalledBy())
	assert.Equal(t, "github", got.GetSlug())
	assert.Equal(t, "stdio", got.GetTransportType())
	assert.Equal(t, "npx", got.GetCommand())
	assert.Equal(t, `["@modelcontextprotocol/server-github"]`, got.GetArgs())
	assert.Equal(t, `{"GITHUB_TOKEN":"ghp_xxx"}`, got.GetEnvVars())
	assert.True(t, got.GetIsEnabled())
}

func TestToProtoInstalledMcpServer_CustomInstall_DefaultsForEmptyJSON(t *testing.T) {
	in := &extdom.InstalledMcpServer{
		ID:             2,
		OrganizationID: 42,
		RepositoryID:   7,
		// MarketItemID nil
		Scope:         "user",
		Name:          "custom",
		Slug:          "custom-mcp",
		TransportType: "http",
		HttpURL:       "https://example.com/mcp",
		IsEnabled:     true,
		CreatedAt:     mustParseTime(t, "2026-05-01T00:00:00Z"),
		UpdatedAt:     mustParseTime(t, "2026-05-12T00:00:00Z"),
	}
	got := toProtoInstalledMcpServer(in)
	require.NotNil(t, got)
	assert.Nil(t, got.MarketItemId)
	assert.Equal(t, "[]", got.GetArgs())
	assert.Equal(t, "{}", got.GetHttpHeaders())
	assert.Equal(t, "{}", got.GetEnvVars())
	assert.Equal(t, "https://example.com/mcp", got.GetHttpUrl())
}

func TestToProtoInstalledMcpServer_NilInputReturnsNil(t *testing.T) {
	assert.Nil(t, toProtoInstalledMcpServer(nil))
}
