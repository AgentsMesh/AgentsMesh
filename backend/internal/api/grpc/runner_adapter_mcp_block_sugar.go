package grpc

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
)

func (a *GRPCRunnerAdapter) mcpIndicatorDefine(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		WorkspaceID    string         `json:"workspace_id"`
		IdempotencyKey string         `json:"idempotency_key,omitempty"`
		Arguments      map[string]any `json:"arguments"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}
	typeKey, _ := params.Arguments["type_key"].(string)
	if typeKey == "" {
		return nil, newMcpError(400, "type_key is required")
	}
	data := make(map[string]any, len(params.Arguments))
	for k, v := range params.Arguments {
		data[k] = v
	}
	return a.applyOneWithParams(ctx, tc, blockstore.OpCreateBlock, applyOpsPayload{
		WorkspaceID:    params.WorkspaceID,
		IdempotencyKey: params.IdempotencyKey,
		Payload: map[string]any{
			"type": blockstore.BlockTypeTypeDef,
			"data": data,
			"text": typeKey,
		},
	})
}

func (a *GRPCRunnerAdapter) mcpTriggerDefine(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		WorkspaceID    string         `json:"workspace_id"`
		IdempotencyKey string         `json:"idempotency_key,omitempty"`
		Arguments      map[string]any `json:"arguments"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}
	name, _ := params.Arguments["name"].(string)
	if name == "" {
		return nil, newMcpError(400, "name is required")
	}
	data := make(map[string]any, len(params.Arguments))
	for k, v := range params.Arguments {
		data[k] = v
	}
	if _, has := data["enabled"]; !has {
		data["enabled"] = true
	}
	return a.applyOneWithParams(ctx, tc, blockstore.OpCreateBlock, applyOpsPayload{
		WorkspaceID:    params.WorkspaceID,
		IdempotencyKey: params.IdempotencyKey,
		Payload: map[string]any{
			"type": blockstore.BlockTypeTriggerDef,
			"data": data,
			"text": name,
		},
	})
}
