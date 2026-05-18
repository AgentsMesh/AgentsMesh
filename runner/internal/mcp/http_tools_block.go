package mcp

import (
	"context"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// Block Store MCP tools — the agent-facing surface for structured
// collaboration (notes, tasks, views, indicators, triggers). Each tool
// forwards the agent's JSON args to backend via the gRPC stream; backend
// runs the actual service-layer write / read. Registered by registerTools()
// in http_server_middleware.go.
//
// Tool naming is `block.*` for CRUD/ref ops (6 tools) plus three sugar
// tools (`indicator.define`, `trigger.define`, `memory.retrieve`) that
// backend dispatch expands into createBlock + a type wrapper or a direct
// SemanticSearch call.

// createBlockCreateTool — `block.create` — primitive block creation.
func (s *HTTPServer) createBlockCreateTool() *MCPTool {
	return &MCPTool{
		Name:        "block.create",
		Description: "Create a new block inside a workspace. Use for notes, tasks, paragraphs, views, indicator records, etc. If you don't know the workspace_id, call block.list_workspaces or block.get_default_workspace first.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"workspace_id":    blockstoreStringProp("Workspace UUID. Required. Use block.list_workspaces / block.get_default_workspace to obtain it."),
				"idempotency_key": blockstoreStringProp("Optional client-supplied key; second call with same key is a no-op replay returning the original op_ids."),
				"payload":         blockstoreObjectProp("Block spec: {id?, type, data, text?, meta?}. `type` must be a registered block type key. IMPORTANT — two text fields with DIFFERENT roles: (1) `data.text` is the **UI-rendered content** for paragraph / heading / bulleted_list_item / numbered_list_item / quote / callout — agents MUST put displayable content here, or the UI will render blank. (2) top-level `text` is a plain-text summary used only for full-text search and semantic embedding (memory.retrieve); the server does not derive it from data and the UI does not read it. To make a block both visible AND searchable, populate BOTH fields with the same string. Heading also needs `data.level` (1|2|3)."),
			},
			"required": []string{"workspace_id", "payload"},
		},
		Handler: blockstoreHandler(func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			return client.BlockCreate(ctx, args)
		}),
	}
}

func (s *HTTPServer) createBlockUpdateTool() *MCPTool {
	return &MCPTool{
		Name:        "block.update",
		Description: "Update an existing block's data, text, or meta. Pass expected_updated_at inside payload for optimistic concurrency. If you don't know the workspace_id, call block.list_workspaces or block.get_default_workspace first.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"workspace_id":    blockstoreStringProp("Workspace UUID. Use block.list_workspaces / block.get_default_workspace to obtain it."),
				"idempotency_key": blockstoreStringProp("Optional retry key."),
				"payload":         blockstoreObjectProp("{id, data?, text?, meta?, expected_updated_at?}. SAME `text` semantics as block.create: `data.text` is the UI-rendered content; top-level `text` is only the search/embedding summary. Update both when the visible string changes."),
			},
			"required": []string{"workspace_id", "payload"},
		},
		Handler: blockstoreHandler(func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			return client.BlockUpdate(ctx, args)
		}),
	}
}

func (s *HTTPServer) createBlockDeleteTool() *MCPTool {
	return &MCPTool{
		Name:        "block.delete",
		Description: "Soft-delete a block. Incoming refs are not cascaded; downstream consumers treat dangling targets as tombstones. If you don't know the workspace_id, call block.list_workspaces or block.get_default_workspace first.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"workspace_id":    blockstoreStringProp("Workspace UUID. Use block.list_workspaces / block.get_default_workspace to obtain it."),
				"idempotency_key": blockstoreStringProp("Optional retry key."),
				"payload":         blockstoreObjectProp("{id}"),
			},
			"required": []string{"workspace_id", "payload"},
		},
		Handler: blockstoreHandler(func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			return client.BlockDelete(ctx, args)
		}),
	}
}

func (s *HTTPServer) createBlockAddRefTool() *MCPTool {
	return &MCPTool{
		Name:        "block.add_ref",
		Description: "Create a typed relationship from one block to another. Use rel='nest' (with order_key) for parent→child placement. If you don't know the workspace_id, call block.list_workspaces or block.get_default_workspace first.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"workspace_id":    blockstoreStringProp("Workspace UUID. Use block.list_workspaces / block.get_default_workspace to obtain it."),
				"idempotency_key": blockstoreStringProp("Optional retry key."),
				"payload":         blockstoreObjectProp("{from, to, rel, order_key?, anchor?}"),
			},
			"required": []string{"workspace_id", "payload"},
		},
		Handler: blockstoreHandler(func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			return client.BlockAddRef(ctx, args)
		}),
	}
}

func (s *HTTPServer) createBlockRemoveRefTool() *MCPTool {
	return &MCPTool{
		Name:        "block.remove_ref",
		Description: "Delete a block-to-block ref by its integer id. If you don't know the workspace_id, call block.list_workspaces or block.get_default_workspace first.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"workspace_id":    blockstoreStringProp("Workspace UUID. Use block.list_workspaces / block.get_default_workspace to obtain it."),
				"idempotency_key": blockstoreStringProp("Optional retry key."),
				"payload":         blockstoreObjectProp("{ref_id}"),
			},
			"required": []string{"workspace_id", "payload"},
		},
		Handler: blockstoreHandler(func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			return client.BlockRemoveRef(ctx, args)
		}),
	}
}

func (s *HTTPServer) createBlockUpdateRefTool() *MCPTool {
	return &MCPTool{
		Name:        "block.update_ref",
		Description: "Reposition or re-annotate a ref (change parent, order_key, anchor, or meta). For rel='nest' this is the canonical 'move' operation. If you don't know the workspace_id, call block.list_workspaces or block.get_default_workspace first.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"workspace_id":    blockstoreStringProp("Workspace UUID. Use block.list_workspaces / block.get_default_workspace to obtain it."),
				"idempotency_key": blockstoreStringProp("Optional retry key."),
				"payload":         blockstoreObjectProp("{ref_id, from?, order_key?, anchor?, meta?}"),
			},
			"required": []string{"workspace_id", "payload"},
		},
		Handler: blockstoreHandler(func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			return client.BlockUpdateRef(ctx, args)
		}),
	}
}
