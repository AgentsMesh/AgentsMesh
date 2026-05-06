package mcp

import (
	"context"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// Workspace discovery tools — give agents a way to obtain the workspace UUID
// without guessing. Every block.* / memory.retrieve / indicator.define /
// trigger.define call requires a workspace_id, and there's no other surface
// that returns one (Pod registration carries org/ticket/project but not
// workspace). Backend dispatch lives in runner_adapter_mcp_block_workspace.go.

func (s *HTTPServer) createBlockListWorkspacesTool() *MCPTool {
	return &MCPTool{
		Name:        "block.list_workspaces",
		Description: "List all block-store workspaces in the current organization. Each workspace is a namespaced container for blocks; cross-workspace refs are not allowed. Returns {workspaces:[{id, slug, name, root_block_id, created_at}, ...]}. Call this first to discover the workspace UUID before any block.* / memory.retrieve / indicator.define / trigger.define call.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: blockstoreHandler(func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			return client.BlockListWorkspaces(ctx, args)
		}),
	}
}

func (s *HTTPServer) createBlockGetDefaultWorkspaceTool() *MCPTool {
	return &MCPTool{
		Name:        "block.get_default_workspace",
		Description: "Get (or auto-create) the organization's default workspace. Returns {id, slug:'default', name, root_block_id, created_at}. Use this when no specific workspace is required — most agents work here unless explicitly told otherwise.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: blockstoreHandler(func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			return client.BlockGetDefaultWorkspace(ctx, args)
		}),
	}
}
