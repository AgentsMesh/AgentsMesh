package mcp

import (
	"context"
)

// Block Store client methods — implements BlockStoreClient for the gRPC
// transport. Each method sends the args through as-is to a matching MCP
// method on backend (runner_adapter_mcp.go:dispatchMcpMethod). The
// Runner-side tool handlers (http_tools_block.go) are the authoritative
// place where agent input is shaped; here we just forward.

func (c *GRPCCollaborationClient) BlockCreate(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	return c.blockstoreCall(ctx, "block.create", args)
}

func (c *GRPCCollaborationClient) BlockUpdate(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	return c.blockstoreCall(ctx, "block.update", args)
}

func (c *GRPCCollaborationClient) BlockDelete(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	return c.blockstoreCall(ctx, "block.delete", args)
}

func (c *GRPCCollaborationClient) BlockAddRef(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	return c.blockstoreCall(ctx, "block.add_ref", args)
}

func (c *GRPCCollaborationClient) BlockRemoveRef(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	return c.blockstoreCall(ctx, "block.remove_ref", args)
}

func (c *GRPCCollaborationClient) BlockUpdateRef(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	return c.blockstoreCall(ctx, "block.update_ref", args)
}

func (c *GRPCCollaborationClient) IndicatorDefine(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	return c.blockstoreCall(ctx, "indicator.define", args)
}

func (c *GRPCCollaborationClient) TriggerDefine(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	return c.blockstoreCall(ctx, "trigger.define", args)
}

func (c *GRPCCollaborationClient) MemoryRetrieve(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	return c.blockstoreCall(ctx, "memory.retrieve", args)
}

func (c *GRPCCollaborationClient) BlockListTypes(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	return c.blockstoreCall(ctx, "block.list_types", args)
}

func (c *GRPCCollaborationClient) BlockListWorkspaces(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	return c.blockstoreCall(ctx, "block.list_workspaces", args)
}

func (c *GRPCCollaborationClient) BlockGetDefaultWorkspace(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	return c.blockstoreCall(ctx, "block.get_default_workspace", args)
}

// blockstoreCall is a thin wrapper over c.call that unmarshals into a
// map[string]interface{} for the agent-facing tool layer. Block Store
// responses are heterogeneous (ApplyOps returns {op_ids, was_replay},
// memory.retrieve returns {hits: [...]}), so a generic map keeps this
// client small at the cost of losing strongly-typed returns.
func (c *GRPCCollaborationClient) blockstoreCall(ctx context.Context, method string, args map[string]interface{}) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := c.call(ctx, method, args, &result); err != nil {
		return nil, err
	}
	if result == nil {
		result = map[string]interface{}{}
	}
	return result, nil
}
