package extensionconnect

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	extdom "github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	extensionv1 "github.com/anthropics/agentsmesh/proto/gen/go/extension/v1"
)

func TestListMarketSkills_MissingOrgSlug_InvalidArgument(t *testing.T) {
	srv := NewMarketServer(NewServer(nil, &fakeOrgService{role: "member"}))
	_, err := srv.ListMarketSkills(ctxAsUser(42), connect.NewRequest(&extensionv1.ListMarketSkillsRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestListMarketMcpServers_MissingOrgSlug_InvalidArgument(t *testing.T) {
	srv := NewMarketServer(NewServer(nil, &fakeOrgService{role: "member"}))
	_, err := srv.ListMarketMcpServers(ctxAsUser(42), connect.NewRequest(&extensionv1.ListMarketMcpServersRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestToProtoSkillMarketItem_PopulatedFields(t *testing.T) {
	in := &extdom.SkillMarketItem{
		ID:            11,
		RegistryID:    7,
		Slug:          "format-go",
		DisplayName:   "Format Go",
		Description:   "Skill description",
		License:       "MIT",
		Compatibility: "go",
		AllowedTools:  "Bash",
		Category:      "code",
		ContentSha:    "abc123",
		StorageKey:    "skills/format-go.zip",
		PackageSize:   4096,
		Version:       2,
		AgentFilter:   []byte(`["claude-code","codex"]`),
		IsActive:      true,
		CreatedAt:     mustParseTime(t, "2026-05-01T00:00:00Z"),
		UpdatedAt:     mustParseTime(t, "2026-05-12T00:00:00Z"),
	}
	got := toProtoSkillMarketItem(in)
	require.NotNil(t, got)
	assert.Equal(t, int64(11), got.GetId())
	assert.Equal(t, int64(7), got.GetRegistryId())
	assert.Equal(t, "format-go", got.GetSlug())
	assert.Equal(t, "Format Go", got.GetDisplayName())
	assert.Equal(t, int64(4096), got.GetPackageSize())
	assert.Equal(t, int32(2), got.GetVersion())
	assert.Equal(t, []string{"claude-code", "codex"}, got.GetAgentFilter())
	assert.True(t, got.GetIsActive())
	assert.Equal(t, "2026-05-01T00:00:00Z", got.GetCreatedAt())
}

func TestToProtoSkillMarketItem_EmptyAgentFilterAllAgents(t *testing.T) {
	in := &extdom.SkillMarketItem{
		ID:        1,
		Slug:      "demo",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		// AgentFilter raw bytes is empty → "all agents"
	}
	got := toProtoSkillMarketItem(in)
	require.NotNil(t, got)
	assert.Nil(t, got.AgentFilter, "empty agent_filter must remain absent on the wire")
}

func TestToProtoMcpMarketItem_JSONFieldsPassedThrough(t *testing.T) {
	syncTime := mustParseTime(t, "2026-05-12T13:16:10Z")
	in := &extdom.McpMarketItem{
		ID:                 21,
		Slug:               "github",
		Name:               "GitHub MCP",
		Description:        "MCP for GitHub",
		Icon:               "github",
		TransportType:      "stdio",
		Command:            "npx",
		DefaultArgs:        []byte(`["@modelcontextprotocol/server-github"]`),
		DefaultHttpURL:     "",
		DefaultHttpHeaders: []byte(`[]`),
		EnvVarSchema:       []byte(`[{"name":"GITHUB_TOKEN","label":"GitHub Token","required":true,"sensitive":true,"placeholder":"ghp_xxx"}]`),
		AgentFilter:        []byte(`["claude-code"]`),
		Category:           "vcs",
		IsActive:           true,
		Source:             "seed",
		RegistryName:       "official",
		Version:            "1.0.0",
		RepositoryURL:      "https://github.com/modelcontextprotocol/servers",
		RegistryMeta:       []byte(`{}`),
		LastSyncedAt:       &syncTime,
		CreatedAt:          mustParseTime(t, "2026-05-01T00:00:00Z"),
		UpdatedAt:          mustParseTime(t, "2026-05-12T00:00:00Z"),
	}
	got := toProtoMcpMarketItem(in)
	require.NotNil(t, got)
	assert.Equal(t, int64(21), got.GetId())
	assert.Equal(t, `["@modelcontextprotocol/server-github"]`, got.GetDefaultArgs())
	assert.Equal(t, `[]`, got.GetDefaultHttpHeaders())
	assert.Equal(t, `{}`, got.GetRegistryMeta())
	require.Len(t, got.GetEnvVarSchema(), 1)
	entry := got.GetEnvVarSchema()[0]
	assert.Equal(t, "GITHUB_TOKEN", entry.GetName())
	assert.Equal(t, "GitHub Token", entry.GetLabel())
	assert.True(t, entry.GetRequired())
	assert.True(t, entry.GetSensitive())
	assert.Equal(t, "ghp_xxx", entry.GetPlaceholder())
	assert.Equal(t, []string{"claude-code"}, got.GetAgentFilter())
	assert.Equal(t, "2026-05-12T13:16:10Z", got.GetLastSyncedAt())
}

func TestToProtoMcpMarketItem_EmptyJSONFieldsGetDefaults(t *testing.T) {
	in := &extdom.McpMarketItem{
		ID:        99,
		Slug:      "demo",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		// All json.RawMessage fields nil → use sentinel defaults
	}
	got := toProtoMcpMarketItem(in)
	require.NotNil(t, got)
	assert.Equal(t, "[]", got.GetDefaultArgs(), "empty default_args must encode as JSON empty array")
	assert.Equal(t, "[]", got.GetDefaultHttpHeaders())
	assert.Equal(t, "{}", got.GetRegistryMeta())
	assert.Nil(t, got.LastSyncedAt)
}

func TestToProtoSkillMarketItem_NilInputReturnsNil(t *testing.T) {
	assert.Nil(t, toProtoSkillMarketItem(nil))
}

func TestToProtoMcpMarketItem_NilInputReturnsNil(t *testing.T) {
	assert.Nil(t, toProtoMcpMarketItem(nil))
}
