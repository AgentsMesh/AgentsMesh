package grpc

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	blockstoreservice "github.com/anthropics/agentsmesh/backend/internal/service/blockstore"
)

// One tool call → one op. Batch writes stay REST-only.
type applyOpsPayload struct {
	WorkspaceID    string         `json:"workspace_id"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	Payload        map[string]any `json:"payload"`
}

func (a *GRPCRunnerAdapter) applySingleOp(
	ctx context.Context,
	tc *middleware.TenantContext,
	opKind string,
	raw []byte,
) (interface{}, *mcpError) {
	if a.blockstoreService == nil {
		return nil, newMcpError(501, "blockstore service not configured")
	}
	var params applyOpsPayload
	if err := unmarshalPayload(raw, &params); err != nil {
		return nil, err
	}
	return a.applyOneWithParams(ctx, tc, opKind, params)
}

func (a *GRPCRunnerAdapter) applyOneWithParams(
	ctx context.Context,
	tc *middleware.TenantContext,
	opKind string,
	params applyOpsPayload,
) (interface{}, *mcpError) {
	if a.blockstoreService == nil {
		return nil, newMcpError(501, "blockstore service not configured")
	}
	if params.WorkspaceID == "" {
		return nil, newMcpError(400, "workspace_id is required")
	}
	res, svcErr := a.blockstoreService.ApplyOps(ctx, actorFromTenant(ctx, tc), blockstoreservice.ApplyOpsInput{
		WorkspaceID:    params.WorkspaceID,
		IdempotencyKey: params.IdempotencyKey,
		Ops: []blockstoreservice.OpEnvelope{
			{Op: opKind, Payload: params.Payload},
		},
	})
	if svcErr != nil {
		return nil, blockstoreErrToMcp(svcErr)
	}
	return res, nil
}

func (a *GRPCRunnerAdapter) mcpBlockCreate(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	return a.applySingleOp(ctx, tc, blockstore.OpCreateBlock, payload)
}
func (a *GRPCRunnerAdapter) mcpBlockUpdate(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	return a.applySingleOp(ctx, tc, blockstore.OpUpdateBlock, payload)
}
func (a *GRPCRunnerAdapter) mcpBlockDelete(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	return a.applySingleOp(ctx, tc, blockstore.OpDeleteBlock, payload)
}
func (a *GRPCRunnerAdapter) mcpBlockAddRef(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	return a.applySingleOp(ctx, tc, blockstore.OpAddRef, payload)
}
func (a *GRPCRunnerAdapter) mcpBlockRemoveRef(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	return a.applySingleOp(ctx, tc, blockstore.OpRemoveRef, payload)
}
func (a *GRPCRunnerAdapter) mcpBlockUpdateRef(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	return a.applySingleOp(ctx, tc, blockstore.OpUpdateRef, payload)
}

// Mirrors translateErr in REST layer — both transports MUST produce consistent 4xx/5xx.
func blockstoreErrToMcp(err error) *mcpError {
	switch {
	case errors.Is(err, blockstore.ErrWorkspaceNotFound),
		errors.Is(err, blockstore.ErrBlockNotFound),
		errors.Is(err, blockstore.ErrRefNotFound):
		return newMcpError(404, err.Error())
	case errors.Is(err, blockstore.ErrOrgMismatch),
		errors.Is(err, blockstore.ErrBlockForbidden):
		return newMcpError(403, err.Error())
	case errors.Is(err, blockstore.ErrSingleNestParent),
		errors.Is(err, blockstore.ErrNestCycle),
		errors.Is(err, blockstore.ErrStaleUpdate),
		errors.Is(err, blockstore.ErrWorkspaceAlreadyExists):
		return newMcpError(409, err.Error())
	case errors.Is(err, blockstore.ErrUnknownBlockType),
		errors.Is(err, blockstore.ErrUnknownOpKind),
		errors.Is(err, blockstore.ErrInvalidRel),
		errors.Is(err, blockstore.ErrOrderKeyRequired),
		errors.Is(err, blockstore.ErrMissingRequiredKey),
		errors.Is(err, blockstore.ErrColumnValueInvalid),
		errors.Is(err, blockstore.ErrChildNotAllowed),
		errors.Is(err, blockstore.ErrCrossWorkspaceRef),
		errors.Is(err, blockstore.ErrApplyOpsEmpty),
		errors.Is(err, blockstore.ErrEmbeddingDisabled):
		return newMcpError(400, err.Error())
	default:
		return newMcpError(500, err.Error())
	}
}
