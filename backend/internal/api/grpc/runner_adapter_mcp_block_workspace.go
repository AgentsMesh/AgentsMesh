package grpc

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
)

// Workspace discovery handlers — surface the org's block_workspaces to agents
// so they can obtain a workspace UUID before any block.* / memory.retrieve /
// indicator.define / trigger.define call. See blockstore_ops.go for the
// equivalent REST routes; these gRPC dispatch entries mirror them for the
// runner→backend MCP bridge.
//
// Both handlers take an empty payload. unmarshalPayload on len==0 is a no-op
// (runner_adapter_mcp_response.go:66), so we don't even need to declare a
// param struct — but keeping the call documents the parsing contract.

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
