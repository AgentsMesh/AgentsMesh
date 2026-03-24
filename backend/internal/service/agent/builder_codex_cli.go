package agent

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

const CodexCLISlug = "codex-cli"

// codexMinimumVersion is the minimum supported Codex CLI version (Rust rewrite).
// Node.js versions (< 0.100.0) are no longer supported.
const codexMinimumVersion = "0.100.0"

// codexApprovalFlagMap maps internal approval mode names to Codex CLI Rust flags.
// Codex CLI Rust has three distinct modes with different flags:
//   - suggest   → --ask-for-approval on-request  (user confirms each action)
//   - auto-edit → --full-auto                    (auto within workspace, ask for external)
//   - full-auto → --yolo                         (fully autonomous, no sandbox)
//
// Our pods run in isolated sandboxes, so Codex's built-in sandbox is redundant.
// "full-auto" maps to --yolo because the pod IS the sandbox.
var codexApprovalFlagMap = map[string][]string{
	"suggest":   {"--ask-for-approval", "on-request"},
	"auto-edit": {"--full-auto"},
	"full-auto": {"--yolo"},
}

// CodexCLIBuilder is the builder for Codex CLI agent (Rust rewrite, >= 0.100.0).
// Codex CLI syntax: codex [prompt] [options]
// MCP injection uses CODEX_HOME per-pod isolation with TOML config.
type CodexCLIBuilder struct {
	*BaseAgentBuilder
}

// NewCodexCLIBuilder creates a new CodexCLIBuilder
func NewCodexCLIBuilder() *CodexCLIBuilder {
	return &CodexCLIBuilder{
		BaseAgentBuilder: NewBaseAgentBuilder(CodexCLISlug),
	}
}

// Slug returns the agent type identifier
func (b *CodexCLIBuilder) Slug() string {
	return CodexCLISlug
}

// SupportsMcp returns true - Codex CLI supports MCP servers via config.toml
func (b *CodexCLIBuilder) SupportsMcp() bool { return true }

// SupportsSkills returns false - Codex CLI has no plugin/skill system like Claude Code
func (b *CodexCLIBuilder) SupportsSkills() bool { return false }

// HandleInitialPrompt prepends the initial prompt to launch arguments.
// Codex CLI syntax: codex [prompt] [options]
func (b *CodexCLIBuilder) HandleInitialPrompt(ctx *BuildContext, args []string) []string {
	if ctx.Request.InitialPrompt != "" {
		return append([]string{ctx.Request.InitialPrompt}, args...)
	}
	return args
}

// BuildLaunchArgs builds launch arguments for Codex CLI Rust version.
// Enforces minimum version and maps approval values to Rust CLI format.
func (b *CodexCLIBuilder) BuildLaunchArgs(ctx *BuildContext) ([]string, error) {
	// Enforce minimum version: only Rust rewrite (>= 0.100.0) is supported
	if ctx.AgentVersion != "" && CompareVersions(ctx.AgentVersion, codexMinimumVersion) < 0 {
		return nil, fmt.Errorf(
			"codex CLI %s is not supported (minimum: %s), please upgrade: npm install -g @openai/codex@latest",
			ctx.AgentVersion, codexMinimumVersion,
		)
	}

	args, err := b.BaseAgentBuilder.BuildLaunchArgs(ctx)
	if err != nil {
		return nil, err
	}

	// Map internal approval mode names to Codex CLI Rust flags
	args = mapCodexApprovalFlags(args)

	return args, nil
}

// mapCodexApprovalFlags replaces --ask-for-approval <internal-value> with the
// correct Codex CLI Rust flags. Some modes use entirely different flags
// (e.g., --full-auto, --yolo) rather than --ask-for-approval values.
func mapCodexApprovalFlags(args []string) []string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--ask-for-approval" {
			internalValue := args[i+1]
			if mapped, ok := codexApprovalFlagMap[internalValue]; ok {
				// Replace the two-element pair with the mapped flag(s)
				result := make([]string, 0, len(args))
				result = append(result, args[:i]...)
				result = append(result, mapped...)
				result = append(result, args[i+2:]...)
				return result
			}
		}
	}
	return args
}

// BuildFilesToCreate generates Codex CLI config files for MCP injection.
// Creates a per-pod CODEX_HOME directory with TOML-format config.toml
// containing platform + repo-bound MCP servers.
func (b *CodexCLIBuilder) BuildFilesToCreate(ctx *BuildContext) ([]*runnerv1.FileToCreate, error) {
	// Start with base files from template (DB-driven, should be empty after migration 082)
	files, err := b.BaseAgentBuilder.BuildFilesToCreate(ctx)
	if err != nil {
		return nil, err
	}

	codexHome := "{{.sandbox.root_path}}/codex-home"

	// Create codex-home directory
	files = append(files, &runnerv1.FileToCreate{
		Path:        codexHome,
		IsDirectory: true,
	})

	// Generate TOML config with MCP servers
	tomlContent := b.buildCodexTomlMcpConfig(ctx)
	if tomlContent != "" {
		files = append(files, &runnerv1.FileToCreate{
			Path:    codexHome + "/config.toml",
			Content: tomlContent,
			Mode:    0644,
		})
	}

	return files, nil
}

// BuildEnvVars injects CODEX_HOME and AGENTSMESH_POD_KEY for per-pod isolation.
func (b *CodexCLIBuilder) BuildEnvVars(ctx *BuildContext) (map[string]string, error) {
	envVars, err := b.BaseAgentBuilder.BuildEnvVars(ctx)
	if err != nil {
		return nil, err
	}

	// CODEX_HOME: per-pod config directory (Runner will copy ~/.codex/ here first)
	envVars["CODEX_HOME"] = "{{.sandbox.root_path}}/codex-home"

	// AGENTSMESH_POD_KEY: read by Codex via env_http_headers in config.toml
	if podKey, ok := ctx.TemplateCtx["pod_key"]; ok {
		if key, ok := podKey.(string); ok && key != "" {
			envVars["AGENTSMESH_POD_KEY"] = key
		}
	}

	return envVars, nil
}

// buildCodexTomlMcpConfig generates TOML-format MCP server configuration.
// Uses env_http_headers for X-Pod-Key so the value is read from environment
// at runtime, enabling per-pod isolation with a shared config format.
func (b *CodexCLIBuilder) buildCodexTomlMcpConfig(ctx *BuildContext) string {
	var sb strings.Builder

	// Platform agentsmesh MCP server
	if mcpPort, ok := ctx.TemplateCtx["mcp_port"]; ok && mcpPort != nil {
		if port, ok := mcpPort.(int); ok && port > 0 {
			sb.WriteString("[mcp_servers.agentsmesh]\n")
			fmt.Fprintf(&sb, "url = \"http://127.0.0.1:%d/mcp\"\n", port)
			sb.WriteString("env_http_headers = { \"X-Pod-Key\" = \"AGENTSMESH_POD_KEY\" }\n")
			sb.WriteString("\n")
		}
	}

	// Repo-bound MCP servers from extension system
	for _, srv := range ctx.McpServers {
		if !srv.IsEnabled {
			continue
		}

		serverConfig := srv.ToMcpConfig()
		if len(serverConfig) == 0 {
			slog.Warn("Empty MCP config from ToMcpConfig", "slug", srv.Slug)
			continue
		}

		// Sanitize slug for TOML key (replace hyphens with underscores)
		tomlKey := strings.ReplaceAll(srv.Slug, "-", "_")

		switch srv.TransportType {
		case "http", "sse":
			fmt.Fprintf(&sb, "[mcp_servers.%s]\n", tomlKey)
			if url, ok := serverConfig["url"].(string); ok && url != "" {
				fmt.Fprintf(&sb, "url = %s\n", tomlQuote(url))
			}
			if headers, ok := serverConfig["headers"].(map[string]string); ok && len(headers) > 0 {
				sb.WriteString("http_headers = { ")
				parts := make([]string, 0, len(headers))
				for k, v := range headers {
					parts = append(parts, fmt.Sprintf("%s = %s", tomlQuote(k), tomlQuote(v)))
				}
				sb.WriteString(strings.Join(parts, ", "))
				sb.WriteString(" }\n")
			}
			if envVars, ok := serverConfig["env"].(map[string]string); ok && len(envVars) > 0 {
				envJSON, _ := json.Marshal(envVars)
				fmt.Fprintf(&sb, "env = %s\n", jsonToInlineToml(string(envJSON)))
			}
		case "stdio":
			fmt.Fprintf(&sb, "[mcp_servers.%s]\n", tomlKey)
			if cmd, ok := serverConfig["command"].(string); ok && cmd != "" {
				fmt.Fprintf(&sb, "command = %s\n", tomlQuote(cmd))
			}
			if args, ok := serverConfig["args"].([]string); ok && len(args) > 0 {
				sb.WriteString("args = [")
				quoted := make([]string, len(args))
				for i, a := range args {
					quoted[i] = tomlQuote(a)
				}
				sb.WriteString(strings.Join(quoted, ", "))
				sb.WriteString("]\n")
			}
			if envVars, ok := serverConfig["env"].(map[string]string); ok && len(envVars) > 0 {
				envJSON, _ := json.Marshal(envVars)
				fmt.Fprintf(&sb, "env = %s\n", jsonToInlineToml(string(envJSON)))
			}
		default:
			slog.Warn("Unsupported Codex MCP transport type", "slug", srv.Slug, "type", srv.TransportType)
			continue
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// tomlQuote wraps a string in double quotes with TOML basic string escaping.
// See: https://toml.io/en/v1.0.0#string
func tomlQuote(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return "\"" + s + "\""
}

// jsonToInlineToml converts a JSON object string to TOML inline table format.
// e.g., {"KEY":"val"} → { "KEY" = "val" }
func jsonToInlineToml(jsonStr string) string {
	var m map[string]string
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return "{}"
	}
	parts := make([]string, 0, len(m))
	for k, v := range m {
		parts = append(parts, fmt.Sprintf("%s = %s", tomlQuote(k), tomlQuote(v)))
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

// PostProcess uses the base implementation
func (b *CodexCLIBuilder) PostProcess(ctx *BuildContext, cmd *runnerv1.CreatePodCommand) error {
	return b.BaseAgentBuilder.PostProcess(ctx, cmd)
}
