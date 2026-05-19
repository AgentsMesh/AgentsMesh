package grpc

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	blockstoreservice "github.com/anthropics/agentsmesh/backend/internal/service/blockstore"
	"github.com/google/uuid"
)

func (a *GRPCRunnerAdapter) mcpMemoryRetrieve(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	if a.blockstoreService == nil {
		return nil, newMcpError(501, "blockstore service not configured")
	}
	var params struct {
		WorkspaceID string  `json:"workspace_id"`
		Query       string  `json:"query"`
		TopK        int     `json:"k,omitempty"`
		MinScore    float32 `json:"min_score,omitempty"`
		TypeFilter  string  `json:"type,omitempty"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(params.WorkspaceID)
	if err != nil {
		return nil, newMcpError(400, "workspace_id must be a uuid")
	}
	hits, svcErr := a.blockstoreService.SemanticSearch(ctx, actorFromTenant(ctx, tc), blockstoreservice.SearchInput{
		WorkspaceID: wsID,
		Query:       params.Query,
		TopK:        params.TopK,
		MinScore:    params.MinScore,
		TypeFilter:  params.TypeFilter,
	})
	if svcErr != nil {
		return nil, blockstoreErrToMcp(svcErr)
	}
	return map[string]any{"hits": hits}, nil
}

func (a *GRPCRunnerAdapter) mcpBlockListTypes(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	if a.blockstoreService == nil {
		return nil, newMcpError(501, "blockstore service not configured")
	}
	var params struct {
		WorkspaceID string `json:"workspace_id"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(params.WorkspaceID)
	if err != nil {
		return nil, newMcpError(400, "workspace_id must be a uuid")
	}
	specs, svcErr := a.blockstoreService.ListRegisteredTypes(ctx, actorFromTenant(ctx, tc), wsID)
	if svcErr != nil {
		return nil, blockstoreErrToMcp(svcErr)
	}
	return map[string]any{"types": specs}, nil
}
