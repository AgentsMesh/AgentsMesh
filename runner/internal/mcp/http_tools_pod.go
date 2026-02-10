package mcp

import (
	"context"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// Pod Tools

func (s *HTTPServer) createCreatePodTool() *MCPTool {
	return &MCPTool{
		Name:        "create_pod",
		Description: "Create a new agent pod. IMPORTANT: Before calling this tool, you MUST first call list_runners to get the runner_id and agent_type_id. The new pod will automatically have terminal:read and terminal:write permissions to the creator via binding.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"runner_id": map[string]interface{}{
					"type":        "integer",
					"description": "ID of the runner to create the pod on (required). Call list_runners first to get available runner IDs.",
				},
				"agent_type_id": map[string]interface{}{
					"type":        "integer",
					"description": "ID of the agent type to use for the pod (required). Call list_runners first to see available agent types and their IDs.",
				},
				"ticket_id": map[string]interface{}{
					"type":        "integer",
					"description": "ID of the ticket to associate with the pod",
				},
				"initial_prompt": map[string]interface{}{
					"type":        "string",
					"description": "Initial prompt to send to the new agent pod",
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
					"description": "Override agent type default configuration. Keys depend on the agent type's config schema.",
				},
				"permission_mode": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"plan", "default", "bypassPermissions"},
					"description": "Permission mode for the pod: 'plan' (default, requires approval), 'default' (normal permissions), or 'bypassPermissions' (auto-approve all).",
				},
			},
			"required": []string{"runner_id", "agent_type_id"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			req := &tools.PodCreateRequest{
				InitialPrompt: getStringArg(args, "initial_prompt"),
				Model:         getStringArg(args, "model"),
			}

			if v := getIntArg(args, "runner_id"); v != 0 {
				req.RunnerID = v
			}
			if v := getInt64PtrArg(args, "agent_type_id"); v != nil {
				req.AgentTypeID = v
			}
			if v := getIntPtrArg(args, "ticket_id"); v != nil {
				req.TicketID = v
			}
			if v := getInt64PtrArg(args, "repository_id"); v != nil {
				req.RepositoryID = v
			}
			if v := getStringArg(args, "repository_url"); v != "" {
				req.RepositoryURL = &v
			}
			if v := getStringArg(args, "branch_name"); v != "" {
				req.BranchName = &v
			}
			if v := getInt64PtrArg(args, "credential_profile_id"); v != nil {
				req.CredentialProfileID = v
			}
			if v := getMapArg(args, "config_overrides"); v != nil {
				req.ConfigOverrides = v
			}
			if v := getStringArg(args, "permission_mode"); v != "" {
				req.PermissionMode = &v
			}

			// Create the pod
			resp, err := client.CreatePod(ctx, req)
			if err != nil {
				return nil, err
			}

			// Auto-bind to the new pod with terminal permissions
			// This allows the creator to observe and control the new pod's terminal
			scopes := []tools.BindingScope{tools.ScopeTerminalRead, tools.ScopeTerminalWrite}
			binding, err := client.RequestBinding(ctx, resp.PodKey, scopes)
			if err != nil {
				// Pod created but binding failed - return both info
				return map[string]interface{}{
					"pod_key":       resp.PodKey,
					"status":        resp.Status,
					"binding_error": err.Error(),
				}, nil
			}

			return map[string]interface{}{
				"pod_key":        resp.PodKey,
				"status":         resp.Status,
				"binding_id":     binding.ID,
				"binding_status": binding.Status,
			}, nil
		},
	}
}
