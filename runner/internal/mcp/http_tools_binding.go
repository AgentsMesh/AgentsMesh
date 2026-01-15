package mcp

import (
	"context"
	"fmt"

	"github.com/anthropics/agentmesh/runner/internal/mcp/tools"
)

// Binding Tools

func (s *HTTPServer) createBindPodTool() *MCPTool {
	return &MCPTool{
		Name:        "bind_pod",
		Description: "Request to bind with another agent pod. The target pod must accept the binding request.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"target_pod": map[string]interface{}{
					"type":        "string",
					"description": "The pod key of the target pod to bind with",
				},
				"scopes": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string", "enum": []string{"terminal:read", "terminal:write"}},
					"description": "Permission scopes to request (terminal:read, terminal:write)",
				},
			},
			"required": []string{"target_pod", "scopes"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			targetPod := getStringArg(args, "target_pod")
			scopeStrs := getStringSliceArg(args, "scopes")

			if targetPod == "" || len(scopeStrs) == 0 {
				return nil, fmt.Errorf("target_pod and scopes are required")
			}

			scopes := make([]tools.BindingScope, len(scopeStrs))
			for i, s := range scopeStrs {
				scopes[i] = tools.BindingScope(s)
			}

			return client.RequestBinding(ctx, targetPod, scopes)
		},
	}
}

func (s *HTTPServer) createAcceptBindingTool() *MCPTool {
	return &MCPTool{
		Name:        "accept_binding",
		Description: "Accept a pending binding request from another agent pod.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"binding_id": map[string]interface{}{
					"type":        "integer",
					"description": "The ID of the binding request to accept",
				},
			},
			"required": []string{"binding_id"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			bindingID := getIntArg(args, "binding_id")
			if bindingID == 0 {
				return nil, fmt.Errorf("binding_id is required")
			}
			return client.AcceptBinding(ctx, bindingID)
		},
	}
}

func (s *HTTPServer) createRejectBindingTool() *MCPTool {
	return &MCPTool{
		Name:        "reject_binding",
		Description: "Reject a pending binding request from another agent pod.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"binding_id": map[string]interface{}{
					"type":        "integer",
					"description": "The ID of the binding request to reject",
				},
				"reason": map[string]interface{}{
					"type":        "string",
					"description": "Optional reason for rejection",
				},
			},
			"required": []string{"binding_id"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			bindingID := getIntArg(args, "binding_id")
			reason := getStringArg(args, "reason")

			if bindingID == 0 {
				return nil, fmt.Errorf("binding_id is required")
			}

			return client.RejectBinding(ctx, bindingID, reason)
		},
	}
}

func (s *HTTPServer) createUnbindPodTool() *MCPTool {
	return &MCPTool{
		Name:        "unbind_pod",
		Description: "Unbind from a previously bound pod, revoking all permissions.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"target_pod": map[string]interface{}{
					"type":        "string",
					"description": "The pod key of the pod to unbind from",
				},
			},
			"required": []string{"target_pod"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			targetPod := getStringArg(args, "target_pod")
			if targetPod == "" {
				return nil, fmt.Errorf("target_pod is required")
			}

			err := client.UnbindPod(ctx, targetPod)
			if err != nil {
				return nil, err
			}
			return "Pod unbound successfully", nil
		},
	}
}

func (s *HTTPServer) createGetBindingsTool() *MCPTool {
	return &MCPTool{
		Name:        "get_bindings",
		Description: "Get all bindings for this pod, optionally filtered by status.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"status": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"pending", "active", "rejected", "inactive", "expired"},
					"description": "Filter by binding status (optional)",
				},
			},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			var status *tools.BindingStatus
			if s := getStringArg(args, "status"); s != "" {
				bs := tools.BindingStatus(s)
				status = &bs
			}
			return client.GetBindings(ctx, status)
		},
	}
}

func (s *HTTPServer) createGetBoundPodsTool() *MCPTool {
	return &MCPTool{
		Name:        "get_bound_pods",
		Description: "Get list of pods that are currently bound to this pod with active permissions.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			return client.GetBoundPods(ctx)
		},
	}
}
