package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	extensionservice "github.com/anthropics/agentsmesh/backend/internal/service/extension"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildFromAgentfile_NormalMode(t *testing.T) {
	db := setupConfigBuilderTestDB(t)

	db.Exec(`INSERT INTO agents (slug, name, launch_command, is_builtin, is_active, agentfile_source)
		VALUES ('claude-code', 'Claude Code', 'claude', 1, 1, 'AGENT claude
EXECUTABLE claude
MODE pty
PROMPT_POSITION prepend')`)

	provider := createTestProvider(db)
	builder := NewConfigBuilder(provider)

	cmd, err := builder.BuildPodCommand(context.Background(), &ConfigBuildRequest{
		AgentSlug:             "claude-code",
		PodKey:                "pod-test-1",
		MergedAgentfileSource: "AGENT claude\nMODE acp\nPROMPT_POSITION prepend",
		Prompt:                "Hello",
		MCPPort:               19000,
		Cols:                  80,
		Rows:                  24,
	})

	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.Equal(t, "pod-test-1", cmd.PodKey)
	// Eval produces launch_command and interaction_mode from AgentFile
	assert.Equal(t, "claude", cmd.LaunchCommand)
	assert.Equal(t, "acp", cmd.InteractionMode)
	// Prompt is passed as separate fields (Runner handles injection into args)
	assert.Equal(t, "prepend", cmd.PromptPosition)
	assert.Equal(t, "Hello", cmd.Prompt)
	// LaunchArgs should NOT contain prompt (Runner injects based on PromptPosition)
	for _, arg := range cmd.LaunchArgs {
		assert.NotEqual(t, "Hello", arg, "Backend should not inject prompt into LaunchArgs")
	}
}

func TestBuildFromAgentfile_SetupWithoutRepository(t *testing.T) {
	db := setupConfigBuilderTestDB(t)

	db.Exec(`INSERT INTO agents (slug, name, launch_command, is_builtin, is_active, agentfile_source)
		VALUES ('claude-code', 'Claude Code', 'claude', 1, 1, 'AGENT claude
EXECUTABLE claude
MODE acp')`)

	provider := createTestProvider(db)
	builder := NewConfigBuilder(provider)

	cmd, err := builder.BuildPodCommand(context.Background(), &ConfigBuildRequest{
		AgentSlug: "claude-code",
		PodKey:    "pod-setup-only",
		MergedAgentfileSource: `AGENT claude
EXECUTABLE claude
MODE acp
SETUP timeout=60 <<SCRIPT
echo "hi"
SCRIPT`,
		MCPPort: 19000,
		Cols:    80,
		Rows:    24,
	})

	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.NotNil(t, cmd.SandboxConfig)
	assert.Equal(t, `echo "hi"`, cmd.SandboxConfig.PreparationScript)
	assert.Equal(t, int32(60), cmd.SandboxConfig.PreparationTimeout)
	assert.Empty(t, cmd.SandboxConfig.HttpCloneUrl)
	assert.Empty(t, cmd.SandboxConfig.SshCloneUrl)
}

func TestBuildFromAgentfile_SetupOverridesRepositoryFallback(t *testing.T) {
	db := setupConfigBuilderTestDB(t)

	db.Exec(`INSERT INTO agents (slug, name, launch_command, is_builtin, is_active, agentfile_source)
		VALUES ('claude-code', 'Claude Code', 'claude', 1, 1, 'AGENT claude
EXECUTABLE claude
MODE acp')`)

	provider := createTestProvider(db)
	builder := NewConfigBuilder(provider)

	cmd, err := builder.BuildPodCommand(context.Background(), &ConfigBuildRequest{
		AgentSlug:          "claude-code",
		PodKey:             "pod-setup-override",
		HttpCloneURL:       "https://github.com/org/repo.git",
		PreparationScript:  "npm install",
		PreparationTimeout: 600,
		MergedAgentfileSource: `AGENT claude
EXECUTABLE claude
MODE acp
SETUP timeout=45 <<SCRIPT
echo "from agentfile"
SCRIPT`,
		MCPPort: 19000,
		Cols:    80,
		Rows:    24,
	})

	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.NotNil(t, cmd.SandboxConfig)
	assert.Equal(t, `echo "from agentfile"`, cmd.SandboxConfig.PreparationScript)
	assert.Equal(t, int32(45), cmd.SandboxConfig.PreparationTimeout)
	assert.Equal(t, "https://github.com/org/repo.git", cmd.SandboxConfig.HttpCloneUrl)
}

func TestBuildFromAgentfile_CodexHomeConfigAndSkillResources(t *testing.T) {
	db := setupConfigBuilderTestDB(t)

	db.Exec(`INSERT INTO agents (slug, name, launch_command, is_builtin, is_active, agentfile_source)
		VALUES ('codex-cli', 'Codex CLI', 'codex', 1, 1, 'AGENT codex
EXECUTABLE codex
MODE pty')`)

	repoID := int64(42)
	provider := createTestProvider(db)
	builder := NewConfigBuilder(provider)
	builder.SetExtensionProvider(&mockExtensionProvider{
		mcpServers: []*extension.InstalledMcpServer{
			{
				Slug:          "stdio-tool",
				TransportType: extension.TransportTypeStdio,
				Command:       "node",
				Args:          json.RawMessage(`["server.js"]`),
				EnvVars:       json.RawMessage(`{"API_KEY":"secret"}`),
				IsEnabled:     true,
			},
		},
		skills: []*extensionservice.ResolvedSkill{
			{
				Slug:        "repo-skill",
				ContentSha:  "sha123",
				DownloadURL: "https://storage.example.com/repo-skill.tar.gz",
				PackageSize: 1234,
			},
		},
	})

	cmd, err := builder.BuildPodCommand(context.Background(), &ConfigBuildRequest{
		AgentSlug:      "codex-cli",
		OrganizationID: 1,
		UserID:         2,
		RepositoryID:   &repoID,
		PodKey:         "pod-codex",
		MCPPort:        19000,
		MergedAgentfileSource: `# === Identity ===
AGENT codex
EXECUTABLE codex

# === Mode ===
MODE pty
MODE acp "app-server"

# === Configuration ===
CONFIG approval_mode SELECT("untrusted", "on-request", "never") = "untrusted"

# === Environment ===
ENV OPENAI_API_KEY SECRET OPTIONAL
ENV CODEX_HOME = sandbox.root + "/codex-home"

# === Prompt ===
PROMPT_POSITION prepend

# === Capabilities ===
MCP ON

# === Build Logic ===
arg "--ask-for-approval" config.approval_mode when config.approval_mode != "" and mode != "acp"

if mcp.enabled {
  file sandbox.root + "/codex-home/config.toml" codex_mcp_toml(mcp.servers)

  mkdir sandbox.work_dir + "/.codex"
  file sandbox.work_dir + "/.codex/mcp.json" json({ mcpServers: mcp.servers })
}`,
	})

	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.Equal(t, "{{sandbox_root}}/codex-home", cmd.EnvVars["CODEX_HOME"])
	assert.Len(t, cmd.FilesToCreate, 3)

	files := map[string]*runnerv1.FileToCreate{}
	for _, f := range cmd.FilesToCreate {
		files[f.Path] = f
	}

	require.Contains(t, files, "{{work_dir}}/.codex")
	assert.True(t, files["{{work_dir}}/.codex"].IsDirectory)

	require.Contains(t, files, "{{sandbox_root}}/codex-home/config.toml")
	codexToml := files["{{sandbox_root}}/codex-home/config.toml"].Content
	assert.Contains(t, codexToml, "[mcp_servers.agentsmesh]")
	assert.Contains(t, codexToml, `url = "http://127.0.0.1:19000/mcp"`)
	assert.Contains(t, codexToml, "[mcp_servers.agentsmesh.http_headers]")
	assert.NotContains(t, codexToml, "[mcp_servers.agentsmesh.headers]")
	assert.Contains(t, codexToml, `X-Pod-Key = "pod-codex"`)
	assert.Contains(t, codexToml, "[mcp_servers.stdio-tool]")
	assert.Contains(t, codexToml, `command = "node"`)

	require.Contains(t, files, "{{work_dir}}/.codex/mcp.json")
	legacyMcpJSON := files["{{work_dir}}/.codex/mcp.json"].Content
	assert.Contains(t, legacyMcpJSON, `"mcpServers"`)
	assert.Contains(t, legacyMcpJSON, `"agentsmesh"`)
	assert.Contains(t, legacyMcpJSON, `"headers"`)
	assert.Contains(t, legacyMcpJSON, `"X-Pod-Key":"pod-codex"`)
	assert.NotContains(t, legacyMcpJSON, `"http_headers"`)

	require.Len(t, cmd.ResourcesToDownload, 1)
	assert.Equal(t, "sha123", cmd.ResourcesToDownload[0].Sha)
	assert.Equal(t, "https://storage.example.com/repo-skill.tar.gz", cmd.ResourcesToDownload[0].DownloadUrl)
	assert.Equal(t, "{{.sandbox.root_path}}/codex-home/skills/repo-skill", cmd.ResourcesToDownload[0].TargetPath)
	assert.Equal(t, "skill_package", cmd.ResourcesToDownload[0].ResourceType)
	assert.Equal(t, int64(1234), cmd.ResourcesToDownload[0].SizeBytes)
}

type mockExtensionProvider struct {
	mcpServers []*extension.InstalledMcpServer
	skills     []*extensionservice.ResolvedSkill
}

func (m *mockExtensionProvider) GetEffectiveMcpServers(_ context.Context, _, _, _ int64, _ string) ([]*extension.InstalledMcpServer, error) {
	return m.mcpServers, nil
}

func (m *mockExtensionProvider) GetEffectiveSkills(_ context.Context, _, _, _ int64, _ string) ([]*extensionservice.ResolvedSkill, error) {
	return m.skills, nil
}
