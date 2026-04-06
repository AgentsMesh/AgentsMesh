package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// buildAgentfileLayerFromArgs generates an AgentFile Layer from MCP tool arguments.
// Converts scattered config fields into unified AgentFile CONFIG declarations.
func buildAgentfileLayerFromArgs(
	model, permissionMode, prompt string,
	configOverrides map[string]interface{},
	repositoryURL, branchName string,
) string {
	var lines []string

	if permissionMode != "" {
		lines = append(lines, fmt.Sprintf(`CONFIG permission_mode = "%s"`, permissionMode))
	}
	if model != "" {
		lines = append(lines, fmt.Sprintf(`CONFIG model = "%s"`, model))
	}
	for k, v := range configOverrides {
		if k == "model" || k == "permission_mode" {
			continue // already handled above
		}
		lines = append(lines, fmt.Sprintf("CONFIG %s = %s", k, formatMCPConfigValue(v)))
	}
	if prompt != "" {
		escaped := strings.ReplaceAll(prompt, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		escaped = strings.ReplaceAll(escaped, "\n", `\n`)
		escaped = strings.ReplaceAll(escaped, "\t", `\t`)
		lines = append(lines, fmt.Sprintf(`PROMPT "%s"`, escaped))
	}
	if repositoryURL != "" {
		lines = append(lines, fmt.Sprintf(`REPO "%s"`, repositoryURL))
	}
	if branchName != "" {
		lines = append(lines, fmt.Sprintf(`BRANCH "%s"`, branchName))
	}

	return strings.Join(lines, "\n")
}

func formatMCPConfigValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		escaped := strings.ReplaceAll(val, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		escaped = strings.ReplaceAll(escaped, "\n", `\n`)
		escaped = strings.ReplaceAll(escaped, "\t", `\t`)
		return fmt.Sprintf(`"%s"`, escaped)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	default:
		return fmt.Sprintf(`"%v"`, val)
	}
}

// Pod Tools

func (s *HTTPServer) createCreatePodTool() *MCPTool {
	return &MCPTool{
		Name:        "create_pod",
		Description: "Create a new agent pod. IMPORTANT: Before calling this tool, you MUST first call list_runners to get the runner_id and agent_slug. The new pod will automatically have pod:read and pod:write permissions to the creator via binding.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"runner_id": map[string]interface{}{
					"type":        "integer",
					"description": "ID of the runner to create the pod on (required). Call list_runners first to get available runner IDs.",
				},
				"agent_slug": map[string]interface{}{
					"type":        "string",
					"description": "Slug of the agent to use for the pod (required, e.g. 'claude-code'). Call list_runners first to see available agents.",
				},
				"ticket_slug": map[string]interface{}{
					"type":        "string",
					"description": "Ticket slug to associate with the pod (e.g., 'AM-123'). Use search_tickets to find tickets.",
				},
				"prompt": map[string]interface{}{
					"type":        "string",
					"description": "[Deprecated: prefer AgentFile PROMPT declaration] Initial prompt to send to the new agent pod",
				},
				"alias": map[string]interface{}{
					"type":        "string",
					"description": "User-defined display name for the pod (max 100 characters). Shown in sidebar instead of auto-generated title.",
				},
				"model": map[string]interface{}{
					"type":        "string",
					"description": "AI model to use for the pod",
				},
				"repository_id": map[string]interface{}{
					"type":        "integer",
					"description": "ID of the repository to work with (mutually exclusive with repository_url). Use list_repositories to see available repositories.",
				},
				"repository_url": map[string]interface{}{
					"type":        "string",
					"description": "Direct repository URL to clone (takes precedence over repository_id). Use this for repositories not registered in the system.",
				},
				"branch_name": map[string]interface{}{
					"type":        "string",
					"description": "Git branch name to checkout. If not specified, uses repository's default branch.",
				},
				"credential_profile_id": map[string]interface{}{
					"type":        "integer",
					"description": "ID of the credential profile to use. If not specified or 0, uses RunnerHost mode (runner's local environment).",
				},
				"config_overrides": map[string]interface{}{
					"type":        "object",
					"description": "[Deprecated: prefer AgentFile Layer CONFIG declarations] Override agent default configuration. Keys depend on the agent's config schema.",
				},
				"permission_mode": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"plan", "default", "bypassPermissions"},
					"description": "[Deprecated: prefer AgentFile CONFIG permission_mode] Permission mode for the pod: 'plan' (default, requires approval), 'default' (normal permissions), or 'bypassPermissions' (auto-approve all).",
				},
			},
			"required": []string{"runner_id", "agent_slug"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			// Build AgentFile Layer from scattered arguments (AgentFile SSOT).
			model := getStringArg(args, "model")
			permissionMode := getStringArg(args, "permission_mode")
			prompt := getStringArg(args, "prompt")
			configOverrides := getMapArg(args, "config_overrides")
			repositoryURL := getStringArg(args, "repository_url")
			branchName := getStringArg(args, "branch_name")
			agentfileLayer := buildAgentfileLayerFromArgs(model, permissionMode, prompt, configOverrides, repositoryURL, branchName)

			req := &tools.PodCreateRequest{}

			if v := getStringArg(args, "alias"); v != "" {
				req.Alias = &v
			}
			if v := getIntArg(args, "runner_id"); v != 0 {
				req.RunnerID = v
			}
			if v := getStringArg(args, "agent_slug"); v != "" {
				req.AgentSlug = v
			}
			if v := getStringArg(args, "ticket_slug"); v != "" {
				req.TicketSlug = &v
			}
			// repository_id and credential_profile_id remain as separate fields
			// because they are platform-level ID references that cannot be resolved
			// to AgentFile slugs/names on the Runner side (no DB access).
			if v := getInt64PtrArg(args, "repository_id"); v != nil {
				req.RepositoryID = v
			}
			if v := getInt64PtrArg(args, "credential_profile_id"); v != nil {
				req.CredentialProfileID = v
			}
			if agentfileLayer != "" {
				req.AgentfileLayer = &agentfileLayer
			}

			// Create the pod
			resp, err := client.CreatePod(ctx, req)
			if err != nil {
				return nil, err
			}

			// Auto-bind to the new pod with pod interaction permissions
			scopes := []tools.BindingScope{tools.ScopePodRead, tools.ScopePodWrite}
			binding, err := client.RequestBinding(ctx, resp.PodKey, scopes)
			if err != nil {
				return fmt.Sprintf("Pod: %s | Status: %s | Binding Error: %s", resp.PodKey, resp.Status, err.Error()), nil
			}

			return fmt.Sprintf("Pod: %s | Status: %s | Binding: #%d (%s)", resp.PodKey, resp.Status, binding.ID, binding.Status), nil
		},
	}
}
