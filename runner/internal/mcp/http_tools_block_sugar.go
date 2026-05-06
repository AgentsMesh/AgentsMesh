package mcp

import (
	"context"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// Sugar tools wrap a createBlock of a specific type behind a friendlier name.
// Backend dispatch expands them into the equivalent block.create payload so
// the write path, validation, and op-log semantics match a hand-written
// createBlock exactly.

func (s *HTTPServer) createIndicatorDefineTool() *MCPTool {
	return &MCPTool{
		Name:        "indicator.define",
		Description: "Register a schema-driven indicator type (e.g. 'okr', 'incident'). Records of this type can then be created via block.create. If you don't know the workspace_id, call block.list_workspaces or block.get_default_workspace first.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"workspace_id":    blockstoreStringProp("Workspace UUID. Use block.list_workspaces / block.get_default_workspace to obtain it."),
				"idempotency_key": blockstoreStringProp("Optional retry key."),
				"arguments":       blockstoreObjectProp("{type_key, label?, description?, revision?, default_view?, supported_views?, allowed_children?, columns[]}"),
			},
			"required": []string{"workspace_id", "arguments"},
		},
		Handler: blockstoreHandler(func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			return client.IndicatorDefine(ctx, args)
		}),
	}
}

func (s *HTTPServer) createTriggerDefineTool() *MCPTool {
	return &MCPTool{
		Name:        "trigger.define",
		Description: "Register a reactive rule: 'when create/update/delete happens on type X, fire action'. Webhook targets are SSRF-checked backend-side. If you don't know the workspace_id, call block.list_workspaces or block.get_default_workspace first.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"workspace_id":    blockstoreStringProp("Workspace UUID. Use block.list_workspaces / block.get_default_workspace to obtain it."),
				"idempotency_key": blockstoreStringProp("Optional retry key."),
				"arguments":       blockstoreObjectProp("{name, target_type, on ('create'|'update'|'delete'), predicate?, action: {kind, url?, agent_slug?, headers?}, enabled?}"),
			},
			"required": []string{"workspace_id", "arguments"},
		},
		Handler: blockstoreHandler(func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			return client.TriggerDefine(ctx, args)
		}),
	}
}
