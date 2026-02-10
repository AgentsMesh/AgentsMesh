package grpc

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
)

// ==================== Binding MCP Methods ====================

// mcpRequestBinding handles the "request_binding" MCP method.
func (a *GRPCRunnerAdapter) mcpRequestBinding(ctx context.Context, tc *middleware.TenantContext, podKey string, payload []byte) (interface{}, *mcpError) {
	var params struct {
		TargetPod string   `json:"target_pod"`
		Scopes    []string `json:"scopes"`
		Policy    string   `json:"policy"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.TargetPod == "" {
		return nil, newMcpError(400, "target_pod is required")
	}
	if len(params.Scopes) == 0 {
		return nil, newMcpError(400, "scopes is required")
	}

	binding, err := a.bindingService.RequestBinding(ctx, tc.OrganizationID, podKey, params.TargetPod, params.Scopes, params.Policy)
	if err != nil {
		return nil, newMcpErrorf(400, "failed to request binding: %v", err)
	}

	return map[string]interface{}{"binding": binding}, nil
}

// mcpAcceptBinding handles the "accept_binding" MCP method.
func (a *GRPCRunnerAdapter) mcpAcceptBinding(ctx context.Context, tc *middleware.TenantContext, podKey string, payload []byte) (interface{}, *mcpError) {
	var params struct {
		BindingID int64 `json:"binding_id"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.BindingID == 0 {
		return nil, newMcpError(400, "binding_id is required")
	}

	binding, err := a.bindingService.AcceptBinding(ctx, params.BindingID, podKey)
	if err != nil {
		return nil, newMcpErrorf(400, "failed to accept binding: %v", err)
	}

	return map[string]interface{}{"binding": binding}, nil
}

// mcpRejectBinding handles the "reject_binding" MCP method.
func (a *GRPCRunnerAdapter) mcpRejectBinding(ctx context.Context, tc *middleware.TenantContext, podKey string, payload []byte) (interface{}, *mcpError) {
	var params struct {
		BindingID int64  `json:"binding_id"`
		Reason    string `json:"reason"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.BindingID == 0 {
		return nil, newMcpError(400, "binding_id is required")
	}

	binding, err := a.bindingService.RejectBinding(ctx, params.BindingID, podKey, params.Reason)
	if err != nil {
		return nil, newMcpErrorf(400, "failed to reject binding: %v", err)
	}

	return map[string]interface{}{"binding": binding}, nil
}

// mcpUnbindPod handles the "unbind_pod" MCP method.
func (a *GRPCRunnerAdapter) mcpUnbindPod(ctx context.Context, tc *middleware.TenantContext, podKey string, payload []byte) (interface{}, *mcpError) {
	var params struct {
		TargetPod string `json:"target_pod"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.TargetPod == "" {
		return nil, newMcpError(400, "target_pod is required")
	}

	removed, err := a.bindingService.Unbind(ctx, podKey, params.TargetPod)
	if err != nil {
		return nil, newMcpErrorf(500, "failed to unbind: %v", err)
	}
	if !removed {
		return nil, newMcpError(404, "no active binding found")
	}

	return map[string]interface{}{"message": "unbound successfully"}, nil
}

// mcpGetBindings handles the "get_bindings" MCP method.
func (a *GRPCRunnerAdapter) mcpGetBindings(ctx context.Context, tc *middleware.TenantContext, podKey string, payload []byte) (interface{}, *mcpError) {
	var params struct {
		Status *string `json:"status"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	bindings, err := a.bindingService.GetBindingsForPod(ctx, podKey, params.Status)
	if err != nil {
		return nil, newMcpErrorf(500, "failed to get bindings: %v", err)
	}

	return map[string]interface{}{"bindings": bindings}, nil
}

// mcpGetBoundPods handles the "get_bound_pods" MCP method.
func (a *GRPCRunnerAdapter) mcpGetBoundPods(ctx context.Context, tc *middleware.TenantContext, podKey string) (interface{}, *mcpError) {
	pods, err := a.bindingService.GetBoundPods(ctx, podKey)
	if err != nil {
		return nil, newMcpErrorf(500, "failed to get bound pods: %v", err)
	}

	return map[string]interface{}{"pods": pods}, nil
}
