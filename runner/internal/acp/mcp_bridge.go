package acp

import "fmt"

// BuildMCPServersConfig creates the mcpServers configuration
// to pass to session/new so the ACP agent can access AgentsMesh tools.
func BuildMCPServersConfig(mcpPort int) map[string]any {
	return map[string]any{
		"agentsmesh": map[string]any{
			"type":    "http",
			"url":     fmt.Sprintf("http://127.0.0.1:%d/mcp", mcpPort),
			"headers": []any{}, // required by ACP spec, empty for local
		},
	}
}
