package mcp

import (
	"context"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// Block Store read-side tools. memory.retrieve is the semantic search surface
// ("long-term memory"); block.list_types is the schema catalog discovery.

func (s *HTTPServer) createMemoryRetrieveTool() *MCPTool {
	return &MCPTool{
		Name:        "memory.retrieve",
		Description: "Semantic search over the workspace: returns blocks (notes/tasks/comments) ranked by relevance to the query. Use for long-term memory lookup before drafting a response. If you don't know the workspace_id, call block.list_workspaces or block.get_default_workspace first.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"workspace_id": blockstoreStringProp("Workspace UUID. Use block.list_workspaces / block.get_default_workspace to obtain it."),
				"query":        blockstoreStringProp("Free-text query."),
				"k":            map[string]interface{}{"type": "integer", "description": "Top-K hits (default 5, max 100)."},
				"min_score":    map[string]interface{}{"type": "number", "description": "Cosine similarity threshold (0-1)."},
				"type":         blockstoreStringProp("Optional: filter to one block type (e.g. 'task')."),
			},
			"required": []string{"workspace_id", "query"},
		},
		Handler: blockstoreHandler(func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			return client.MemoryRetrieve(ctx, args)
		}),
	}
}

// createBlockListTypesTool exposes the workspace's registered block types so
// agents can discover dynamic indicator types (those registered via
// indicator.define) beyond the static bootstrap set. Call this before a
// block.create when the type_key is not well-known — the response includes
// columns/required fields agents need to populate.
func (s *HTTPServer) createBlockListTypesTool() *MCPTool {
	return &MCPTool{
		Name:        "block.list_types",
		Description: "List every block type available in the workspace (bootstrap + indicator.define registrations). Returns the hydrated type spec for each (columns, required keys, default view, etc.). Call this first when the target type might have been defined dynamically by an agent. If you don't know the workspace_id, call block.list_workspaces or block.get_default_workspace first.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"workspace_id": blockstoreStringProp("Workspace UUID. Use block.list_workspaces / block.get_default_workspace to obtain it."),
			},
			"required": []string{"workspace_id"},
		},
		Handler: blockstoreHandler(func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			return client.BlockListTypes(ctx, args)
		}),
	}
}
