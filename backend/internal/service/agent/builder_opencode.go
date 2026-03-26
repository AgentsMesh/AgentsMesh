package agent

import (
	"encoding/json"
	"fmt"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

const OpenCodeSlug = "opencode"

// OpenCodeBuilder is the builder for OpenCode agent.
// OpenCode CLI syntax: opencode [options]
// MCP and permissions are configured via OPENCODE_CONFIG_CONTENT env var.
type OpenCodeBuilder struct {
	*BaseAgentBuilder
}

// NewOpenCodeBuilder creates a new OpenCodeBuilder
func NewOpenCodeBuilder() *OpenCodeBuilder {
	return &OpenCodeBuilder{
		BaseAgentBuilder: NewBaseAgentBuilder(OpenCodeSlug),
	}
}

// Slug returns the agent type identifier
func (b *OpenCodeBuilder) Slug() string {
	return OpenCodeSlug
}

// HandleInitialPrompt prepends --prompt flag for TUI mode.
// OpenCode syntax: opencode --prompt="message" [options]
// This starts the TUI with the prompt pre-filled and stays running.
// Using = format for shell-safe argument passing.
// BuildLaunchArgs builds launch args and adds --model flag if configured.
// If model is empty but models list exists, uses the first model.
func (b *OpenCodeBuilder) BuildLaunchArgs(ctx *BuildContext) ([]string, error) {
	args, err := b.BaseAgentBuilder.BuildLaunchArgs(ctx)
	if err != nil {
		return nil, err
	}

	model := b.getConfigString(ctx.Config, "model")
	if model != "" {
		args = append(args, "--model", model)
		return args, nil
	}

	if models, ok := ctx.Config["models"]; ok && models != nil {
		if modelsList, ok := models.([]interface{}); ok && len(modelsList) > 0 {
			if firstModel, ok := modelsList[0].(string); ok && firstModel != "" {
				args = append(args, "--model", firstModel)
			}
		}
	}

	return args, nil
}

// HandleInitialPrompt passes the prompt via --prompt flag.
// OpenCode syntax: opencode --prompt "message" [options]
func (b *OpenCodeBuilder) HandleInitialPrompt(ctx *BuildContext, args []string) []string {
	if ctx.Request.InitialPrompt != "" {
		return append([]string{"--prompt", ctx.Request.InitialPrompt}, args...)
	}
	return args
}

// SupportsMcp returns true - OpenCode supports MCP servers
func (b *OpenCodeBuilder) SupportsMcp() bool { return true }

// BuildFilesToCreate uses the base implementation
func (b *OpenCodeBuilder) BuildFilesToCreate(ctx *BuildContext) ([]*runnerv1.FileToCreate, error) {
	return b.BaseAgentBuilder.BuildFilesToCreate(ctx)
}

// BuildEnvVars creates env vars including OPENCODE_CONFIG_CONTENT for MCP and permissions.
func (b *OpenCodeBuilder) BuildEnvVars(ctx *BuildContext) (map[string]string, error) {
	envVars, err := b.BaseAgentBuilder.BuildEnvVars(ctx)
	if err != nil {
		return nil, err
	}

	config := make(map[string]interface{})

	if b.getConfigBool(ctx.Config, "mcp_enabled") {
		mcpServers := b.buildMcpServers(ctx)
		if len(mcpServers) > 0 {
			config["mcp"] = mcpServers
		}
	}

	if b.getConfigBool(ctx.Config, "skip_permissions") {
		config["permission"] = map[string]string{"*": "allow"}
	}

	if len(config) > 0 {
		configJSON, err := json.Marshal(config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal opencode config: %w", err)
		}
		envVars["OPENCODE_CONFIG_CONTENT"] = string(configJSON)
	}

	return envVars, nil
}

// buildMcpServers builds the MCP server configurations.
func (b *OpenCodeBuilder) buildMcpServers(ctx *BuildContext) map[string]interface{} {
	servers := make(map[string]interface{})

	if mcpPort, ok := ctx.TemplateCtx["mcp_port"]; ok && mcpPort != nil {
		if port, ok := mcpPort.(int); ok && port > 0 {
			agentsmeshCfg := map[string]interface{}{
				"type":    "remote",
				"url":     fmt.Sprintf("http://127.0.0.1:%d/mcp", port),
				"enabled": true,
			}
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

	for _, srv := range ctx.McpServers {
		if !srv.IsEnabled {
			continue
		}

		switch srv.TransportType {
		case "stdio", "http", "sse":
		default:
			continue
		}

		serverConfig := srv.ToMcpConfig()
		if len(serverConfig) == 0 {
			continue
		}

		if _, hasType := serverConfig["type"]; !hasType {
			serverConfig["type"] = srv.TransportType
		}

		servers[srv.Slug] = serverConfig
	}

	return servers
}

// PostProcess uses the base implementation
func (b *OpenCodeBuilder) PostProcess(ctx *BuildContext, cmd *runnerv1.CreatePodCommand) error {
	return b.BaseAgentBuilder.PostProcess(ctx, cmd)
}
