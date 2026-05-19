package grpc

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
)

// Cross-bundle: mirrors REST routes in blockstore_ops.go for the runner→backend MCP bridge.
// Empty payload: unmarshalPayload on len==0 is no-op (see runner_adapter_mcp_response.go).

func (a *GRPCRunnerAdapter) mcpBlockListWorkspaces(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	if a.blockstoreService == nil {
		return nil, newMcpError(501, "blockstore service not configured")
	}
	if mcpErr := unmarshalPayload(payload, &struct{}{}); mcpErr != nil {
		return nil, mcpErr
	}
	views, err := a.blockstoreService.ListWorkspaces(ctx, actorFromTenant(ctx, tc))
	if err != nil {
		return nil, blockstoreErrToMcp(err)
	}
	return map[string]any{"workspaces": views}, nil
}

func (a *GRPCRunnerAdapter) mcpBlockGetDefaultWorkspace(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	if a.blockstoreService == nil {
		return nil, newMcpError(501, "blockstore service not configured")
	}
	if mcpErr := unmarshalPayload(payload, &struct{}{}); mcpErr != nil {
		return nil, mcpErr
	}
	view, err := a.blockstoreService.EnsureDefaultWorkspace(ctx, actorFromTenant(ctx, tc))
	if err != nil {
		return nil, blockstoreErrToMcp(err)
	}
	return view, nil
}
