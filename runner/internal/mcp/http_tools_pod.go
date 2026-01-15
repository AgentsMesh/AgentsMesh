package mcp

import (
	"context"

	"github.com/anthropics/agentmesh/runner/internal/mcp/tools"
)

// Pod Tools

func (s *HTTPServer) createCreatePodTool() *MCPTool {
	return &MCPTool{
		Name:        "create_pod",
		Description: "Create a new agent pod. The new pod will automatically have terminal:read and terminal:write permissions to the creator via binding.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"runner_id": map[string]interface{}{
					"type":        "integer",
					"description": "ID of the runner to create the pod on (optional, uses available runner)",
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
			},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			req := &tools.PodCreateRequest{
				InitialPrompt: getStringArg(args, "initial_prompt"),
				Model:         getStringArg(args, "model"),
			}

			if v := getIntArg(args, "runner_id"); v != 0 {
				req.RunnerID = v
			}
			if v := getIntPtrArg(args, "ticket_id"); v != nil {
				req.TicketID = v
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
