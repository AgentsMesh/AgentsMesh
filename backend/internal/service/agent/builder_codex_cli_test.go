package agent

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// codexAgentType mirrors the DB command_template from migration 000059.
func codexAgentType() *agent.AgentType {
	return &agent.AgentType{
		Slug:          CodexCLISlug,
		LaunchCommand: "codex",
		CommandTemplate: agent.CommandTemplate{
			Args: []agent.ArgRule{
				{
					Condition: &agent.Condition{
						Field:    "approval_mode",
						Operator: "not_empty",
					},
					Args: []string{"--ask-for-approval", "{{.config.approval_mode}}"},
				},
			},
		},
	}
}

func codexBuildContext(approvalMode, agentVersion string) *BuildContext {
	config := agent.ConfigValues{}
	if approvalMode != "" {
		config["approval_mode"] = approvalMode
	}
	return &BuildContext{
		Request:   &ConfigBuildRequest{},
		AgentType: codexAgentType(),
		Config:    config,
		TemplateCtx: map[string]interface{}{
			"config":   config,
			"mcp_port": 19000,
			"pod_key":  "test-pod-abc",
		},
		AgentVersion: agentVersion,
	}
}

// --- BuildLaunchArgs tests ---

func TestCodexCLIBuilder_BuildLaunchArgs_MinVersion(t *testing.T) {
	builder := NewCodexCLIBuilder()

	t.Run("rejects old Node.js version", func(t *testing.T) {
		ctx := codexBuildContext("suggest", "0.1.2025050100")
		_, err := builder.BuildLaunchArgs(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")
		assert.Contains(t, err.Error(), "0.100.0")
	})

	t.Run("accepts Rust version", func(t *testing.T) {
		ctx := codexBuildContext("suggest", "0.101.0")
		args, err := builder.BuildLaunchArgs(ctx)
		require.NoError(t, err)
		assert.Contains(t, args, "--ask-for-approval")
	})

	t.Run("accepts exact minimum version", func(t *testing.T) {
		ctx := codexBuildContext("suggest", "0.100.0")
		args, err := builder.BuildLaunchArgs(ctx)
		require.NoError(t, err)
		assert.Contains(t, args, "--ask-for-approval")
	})

	t.Run("skips check when version unknown", func(t *testing.T) {
		ctx := codexBuildContext("suggest", "")
		args, err := builder.BuildLaunchArgs(ctx)
		require.NoError(t, err)
		assert.NotNil(t, args)
	})
}

func TestCodexCLIBuilder_BuildLaunchArgs_ApprovalMapping(t *testing.T) {
	builder := NewCodexCLIBuilder()

	tests := []struct {
		name     string
		approval string
		wantArgs []string
	}{
		{"suggest maps to --ask-for-approval on-request", "suggest", []string{"--ask-for-approval", "on-request"}},
		{"auto-edit maps to --full-auto", "auto-edit", []string{"--full-auto"}},
		{"full-auto maps to --yolo", "full-auto", []string{"--yolo"}},
		{"unknown value passes through", "on-request", []string{"--ask-for-approval", "on-request"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := codexBuildContext(tt.approval, "0.101.0")
			args, err := builder.BuildLaunchArgs(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantArgs, args)
		})
	}

	t.Run("always maps regardless of version", func(t *testing.T) {
		// Even without version, approval values are always mapped (only Rust supported)
		ctx := codexBuildContext("full-auto", "")
		args, err := builder.BuildLaunchArgs(ctx)
		require.NoError(t, err)
		assert.Equal(t, []string{"--yolo"}, args)
	})

	t.Run("empty approval mode filters out args", func(t *testing.T) {
		ctx := codexBuildContext("", "0.101.0")
		args, err := builder.BuildLaunchArgs(ctx)
		require.NoError(t, err)
		assert.Nil(t, args)
	})
}

// --- BuildFilesToCreate tests ---

func TestCodexCLIBuilder_BuildFilesToCreate(t *testing.T) {
	builder := NewCodexCLIBuilder()

	t.Run("generates codex-home directory and config.toml", func(t *testing.T) {
		ctx := codexBuildContext("suggest", "0.101.0")
		files, err := builder.BuildFilesToCreate(ctx)
		require.NoError(t, err)

		foundDir := false
		foundConfig := false
		for _, f := range files {
			if f.IsDirectory && strings.HasSuffix(f.Path, "codex-home") {
				foundDir = true
			}
			if strings.HasSuffix(f.Path, "codex-home/config.toml") {
				foundConfig = true
				// Verify TOML content
				assert.Contains(t, f.Content, "[mcp_servers.agentsmesh]")
				assert.Contains(t, f.Content, "http://127.0.0.1:19000/mcp")
				assert.Contains(t, f.Content, "AGENTSMESH_POD_KEY")
			}
		}
		assert.True(t, foundDir, "should create codex-home directory")
		assert.True(t, foundConfig, "should create config.toml")
	})

	t.Run("no config.toml when no mcp_port", func(t *testing.T) {
		ctx := codexBuildContext("suggest", "0.101.0")
		ctx.TemplateCtx["mcp_port"] = 0
		ctx.McpServers = nil

		files, err := builder.BuildFilesToCreate(ctx)
		require.NoError(t, err)

		for _, f := range files {
			if strings.HasSuffix(f.Path, "config.toml") {
				t.Error("should not create config.toml when no MCP servers")
			}
		}
	})
}

// --- BuildEnvVars tests ---

func TestCodexCLIBuilder_BuildEnvVars(t *testing.T) {
	builder := NewCodexCLIBuilder()

	t.Run("injects CODEX_HOME and AGENTSMESH_POD_KEY", func(t *testing.T) {
		ctx := codexBuildContext("suggest", "0.101.0")
		envVars, err := builder.BuildEnvVars(ctx)
		require.NoError(t, err)

		assert.Equal(t, "{{.sandbox.root_path}}/codex-home", envVars["CODEX_HOME"])
		assert.Equal(t, "test-pod-abc", envVars["AGENTSMESH_POD_KEY"])
	})
}

// --- TOML generation tests ---

func TestCodexCLIBuilder_TomlMcpConfig(t *testing.T) {
	builder := NewCodexCLIBuilder()

	t.Run("platform agentsmesh server", func(t *testing.T) {
		ctx := codexBuildContext("suggest", "0.101.0")
		toml := builder.buildCodexTomlMcpConfig(ctx)

		assert.Contains(t, toml, "[mcp_servers.agentsmesh]")
		assert.Contains(t, toml, `url = "http://127.0.0.1:19000/mcp"`)
		assert.Contains(t, toml, `env_http_headers = { "X-Pod-Key" = "AGENTSMESH_POD_KEY" }`)
	})

	t.Run("includes repo-bound stdio MCP server", func(t *testing.T) {
		ctx := codexBuildContext("suggest", "0.101.0")
		ctx.McpServers = []*extension.InstalledMcpServer{
			{
				Slug:          "my-server",
				IsEnabled:     true,
				TransportType: "stdio",
				Command:       "npx",
				Args:          json.RawMessage(`["-y", "@my/mcp-server"]`),
			},
		}

		toml := builder.buildCodexTomlMcpConfig(ctx)
		assert.Contains(t, toml, "[mcp_servers.my_server]") // hyphens → underscores
		assert.Contains(t, toml, `command = "npx"`)
		assert.Contains(t, toml, `args = ["-y", "@my/mcp-server"]`)
	})

	t.Run("includes repo-bound http MCP server", func(t *testing.T) {
		ctx := codexBuildContext("suggest", "0.101.0")
		ctx.McpServers = []*extension.InstalledMcpServer{
			{
				Slug:          "ext-api",
				IsEnabled:     true,
				TransportType: "http",
				HttpURL:       "https://api.example.com/mcp",
				HttpHeaders:   json.RawMessage(`{"Authorization": "Bearer token123"}`),
			},
		}

		toml := builder.buildCodexTomlMcpConfig(ctx)
		assert.Contains(t, toml, "[mcp_servers.ext_api]")
		assert.Contains(t, toml, `url = "https://api.example.com/mcp"`)
		assert.Contains(t, toml, `"Authorization"`)
		assert.Contains(t, toml, `"Bearer token123"`)
	})

	t.Run("skips disabled servers", func(t *testing.T) {
		ctx := codexBuildContext("suggest", "0.101.0")
		ctx.McpServers = []*extension.InstalledMcpServer{
			{
				Slug:          "disabled-server",
				IsEnabled:     false,
				TransportType: "stdio",
				Command:       "disabled-cmd",
			},
		}

		toml := builder.buildCodexTomlMcpConfig(ctx)
		assert.NotContains(t, toml, "disabled_server")
	})
}

// --- SupportsSkills tests ---

func TestCodexCLIBuilder_SupportsSkills(t *testing.T) {
	builder := NewCodexCLIBuilder()
	assert.False(t, builder.SupportsSkills(), "Codex CLI should not support skills")
}

func TestCodexCLIBuilder_SupportsMcp(t *testing.T) {
	builder := NewCodexCLIBuilder()
	assert.True(t, builder.SupportsMcp(), "Codex CLI should support MCP")
}
