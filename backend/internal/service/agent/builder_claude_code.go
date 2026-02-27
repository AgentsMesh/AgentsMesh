package agent

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

const ClaudeCodeSlug = "claude-code"

// Built-in Skills embedded from runner's canonical source.
// These are platform-level skills always available in Claude Code pods.
//
//go:embed builtin_skills/am-delegate.md
var builtinSkillAmDelegate string

//go:embed builtin_skills/am-channel.md
var builtinSkillAmChannel string

// ClaudeCodeBuilder is the builder for Claude Code agent.
// Claude Code CLI syntax: claude [prompt] [options]
// The prompt must come BEFORE options.
type ClaudeCodeBuilder struct {
	*BaseAgentBuilder
}

// NewClaudeCodeBuilder creates a new ClaudeCodeBuilder
func NewClaudeCodeBuilder() *ClaudeCodeBuilder {
	return &ClaudeCodeBuilder{
		BaseAgentBuilder: NewBaseAgentBuilder(ClaudeCodeSlug),
	}
}

// Slug returns the agent type identifier
func (b *ClaudeCodeBuilder) Slug() string {
	return ClaudeCodeSlug
}

// HandleInitialPrompt prepends the initial prompt to launch arguments.
// Claude Code syntax: claude [prompt] [options]
func (b *ClaudeCodeBuilder) HandleInitialPrompt(ctx *BuildContext, args []string) []string {
	if ctx.Request.InitialPrompt != "" {
		return append([]string{ctx.Request.InitialPrompt}, args...)
	}
	return args
}

// BuildLaunchArgs builds launch arguments, appending --plugin-dir if MCP is enabled.
// The plugin-dir flag is a platform mechanism — always injected by code, not stored in DB.
func (b *ClaudeCodeBuilder) BuildLaunchArgs(ctx *BuildContext) ([]string, error) {
	args, err := b.BaseAgentBuilder.BuildLaunchArgs(ctx)
	if err != nil {
		return nil, err
	}

	// Append --plugin-dir (platform mechanism, not user-configurable)
	args = append(args, "--plugin-dir", "{{.sandbox.root_path}}/agentsmesh-plugin")

	return args, nil
}

// SupportsPlugin returns true - Claude Code supports plugin directory mode
func (b *ClaudeCodeBuilder) SupportsPlugin() bool { return true }

// SupportsMcp returns true - Claude Code supports MCP servers
func (b *ClaudeCodeBuilder) SupportsMcp() bool { return true }

// SupportsSkills returns true - Claude Code supports Skills
func (b *ClaudeCodeBuilder) SupportsSkills() bool { return true }

// BuildFilesToCreate generates Claude Code specific files including plugin structure and MCP config.
// Built-in Skills are embedded in code (not DB files_template) because they are platform mechanisms.
func (b *ClaudeCodeBuilder) BuildFilesToCreate(ctx *BuildContext) ([]*runnerv1.FileToCreate, error) {
	// Start with base files from template (DB-driven, for any user-customizable files)
	files, err := b.BaseAgentBuilder.BuildFilesToCreate(ctx)
	if err != nil {
		return nil, err
	}

	pluginDir := "{{.sandbox.root_path}}/agentsmesh-plugin"
	skillsDir := pluginDir + "/skills"

	// Create agentsmesh-plugin directory structure
	files = append(files,
		&runnerv1.FileToCreate{Path: pluginDir, IsDirectory: true},
		&runnerv1.FileToCreate{Path: pluginDir + "/.claude-plugin", IsDirectory: true},
		&runnerv1.FileToCreate{Path: skillsDir, IsDirectory: true},
	)

	// plugin.json — required by Claude Code --plugin-dir
	pluginJSON, _ := json.Marshal(map[string]interface{}{
		"name":        "agentsmesh",
		"description": "AgentsMesh collaboration plugin for Claude Code",
		"version":     "1.0.0",
	})
	files = append(files, &runnerv1.FileToCreate{
		Path:    pluginDir + "/.claude-plugin/plugin.json",
		Content: string(pluginJSON),
		Mode:    0644,
	})

	// Built-in Skills — platform-level, always present in Claude Code pods
	builtinSkills := map[string]string{
		"am-delegate": builtinSkillAmDelegate,
		"am-channel":  builtinSkillAmChannel,
	}
	for slug, content := range builtinSkills {
		dir := skillsDir + "/" + slug
		files = append(files,
			&runnerv1.FileToCreate{Path: dir, IsDirectory: true},
			&runnerv1.FileToCreate{
				Path:    dir + "/SKILL.md",
				Content: content,
				Mode:    0644,
			},
		)
	}

	// .mcp.json — merged MCP server configuration
	mcpConfig := b.buildMcpConfig(ctx)
	mcpJSON, err := json.MarshalIndent(mcpConfig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal MCP config: %w", err)
	}
	files = append(files, &runnerv1.FileToCreate{
		Path:    "{{.sandbox.work_dir}}/.mcp.json",
		Content: string(mcpJSON),
		Mode:    0644,
	})

	return files, nil
}

// buildMcpConfig builds the merged MCP server configuration
func (b *ClaudeCodeBuilder) buildMcpConfig(ctx *BuildContext) map[string]interface{} {
	servers := make(map[string]interface{})

	// Add built-in agentsmesh MCP server if port is configured
	if mcpPort, ok := ctx.TemplateCtx["mcp_port"]; ok && mcpPort != nil {
		if port, ok := mcpPort.(int); ok && port > 0 {
			agentsmeshCfg := map[string]interface{}{
				"type": "http",
				"url":  fmt.Sprintf("http://127.0.0.1:%d/mcp", port),
			}
			// Include X-Pod-Key header so the MCP proxy can identify this pod
			if podKey, ok := ctx.TemplateCtx["pod_key"]; ok && podKey != nil {
				if key, ok := podKey.(string); ok && key != "" {
					agentsmeshCfg["headers"] = map[string]string{
						"X-Pod-Key": key,
					}
				}
			}
			servers["agentsmesh"] = agentsmeshCfg
		}
	}

	// Add installed MCP servers using ToMcpConfig() which handles
	// MarketItem fallback for command/args/http_url.
	for _, srv := range ctx.McpServers {
		if !srv.IsEnabled {
			continue
		}

		// Only support known transport types
		switch srv.TransportType {
		case "stdio", "http", "sse":
			// supported
		default:
			slog.Warn("Unknown MCP transport type", "slug", srv.Slug, "type", srv.TransportType)
			continue
		}

		serverConfig := srv.ToMcpConfig()
		if len(serverConfig) == 0 {
			slog.Warn("Empty MCP config from ToMcpConfig", "slug", srv.Slug)
			continue
		}

		// Ensure "type" is always set for Claude Code's .mcp.json format
		if _, hasType := serverConfig["type"]; !hasType {
			serverConfig["type"] = srv.TransportType
		}

		servers[srv.Slug] = serverConfig
	}

	return map[string]interface{}{
		"mcpServers": servers,
	}
}

// BuildResourcesToDownload converts resolved skills to ResourceToDownload proto messages
func (b *ClaudeCodeBuilder) BuildResourcesToDownload(ctx *BuildContext) ([]*runnerv1.ResourceToDownload, error) {
	if len(ctx.ResolvedSkills) == 0 {
		return nil, nil
	}

	resources := make([]*runnerv1.ResourceToDownload, 0, len(ctx.ResolvedSkills))
	for _, skill := range ctx.ResolvedSkills {
		targetPath := fmt.Sprintf("{{.sandbox.root_path}}/agentsmesh-plugin/%s", skill.TargetDir)
		resources = append(resources, &runnerv1.ResourceToDownload{
			Sha:          skill.ContentSha,
			DownloadUrl:  skill.DownloadURL,
			TargetPath:   targetPath,
			ResourceType: "skill_package",
			SizeBytes:    skill.PackageSize,
		})
	}

	return resources, nil
}

// BuildEnvVars uses the base implementation
func (b *ClaudeCodeBuilder) BuildEnvVars(ctx *BuildContext) (map[string]string, error) {
	return b.BaseAgentBuilder.BuildEnvVars(ctx)
}

// PostProcess uses the base implementation
func (b *ClaudeCodeBuilder) PostProcess(ctx *BuildContext, cmd *runnerv1.CreatePodCommand) error {
	return b.BaseAgentBuilder.PostProcess(ctx, cmd)
}
