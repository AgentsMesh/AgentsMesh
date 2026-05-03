package grpc

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	blockstoreservice "github.com/anthropics/agentsmesh/backend/internal/service/blockstore"
	"github.com/google/uuid"
)

// mcpMemoryRetrieve runs semantic search over the workspace — the agent's
// "long-term memory" tool. Backed by pgvector when a production embedder is
// wired; falls back to hash-BOW in dev (see service.WarnIfDefaultEmbedder).
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

// mcpBlockListTypes lets an agent discover which block types exist in its
// workspace — including indicator types it (or a peer agent) registered via
// indicator.define. Without this, the tools/list catalog can only advertise
// the static bootstrap set and an agent would have to guess dynamic type
// keys. Returns the hydrated BlockTypeSpec slice (type, label, description,
// columns, etc.) so callers can introspect the schema for createBlock.
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
