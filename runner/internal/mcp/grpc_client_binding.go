package mcp

import (
	"context"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// ==================== BindingClient ====================

// RequestBinding requests a binding with another pod.
func (c *GRPCCollaborationClient) RequestBinding(ctx context.Context, targetPod string, scopes []tools.BindingScope) (*tools.Binding, error) {
	params := map[string]interface{}{
		"target_pod": targetPod,
		"scopes":     scopes,
	}
	var result struct {
		Binding tools.Binding `json:"binding"`
	}
	if err := c.call(ctx, "request_binding", params, &result); err != nil {
		return nil, err
	}
	return &result.Binding, nil
}

// AcceptBinding accepts a binding request.
func (c *GRPCCollaborationClient) AcceptBinding(ctx context.Context, bindingID int) (*tools.Binding, error) {
	params := map[string]interface{}{
		"binding_id": bindingID,
	}
	var result struct {
		Binding tools.Binding `json:"binding"`
	}
	if err := c.call(ctx, "accept_binding", params, &result); err != nil {
		return nil, err
	}
	return &result.Binding, nil
}

// RejectBinding rejects a binding request.
func (c *GRPCCollaborationClient) RejectBinding(ctx context.Context, bindingID int, reason string) (*tools.Binding, error) {
	params := map[string]interface{}{
		"binding_id": bindingID,
		"reason":     reason,
	}
	var result struct {
		Binding tools.Binding `json:"binding"`
	}
	if err := c.call(ctx, "reject_binding", params, &result); err != nil {
		return nil, err
	}
	return &result.Binding, nil
}

// UnbindPod unbinds from another pod.
func (c *GRPCCollaborationClient) UnbindPod(ctx context.Context, targetPod string) error {
	params := map[string]interface{}{
		"target_pod": targetPod,
	}
	return c.call(ctx, "unbind_pod", params, nil)
}

// GetBindings gets all bindings for the current pod.
func (c *GRPCCollaborationClient) GetBindings(ctx context.Context, status *tools.BindingStatus) ([]tools.Binding, error) {
	params := map[string]interface{}{}
	if status != nil {
		params["status"] = string(*status)
	}
	var result struct {
		Bindings []tools.Binding `json:"bindings"`
	}
	if err := c.call(ctx, "get_bindings", params, &result); err != nil {
		return nil, err
	}
	return result.Bindings, nil
}

// GetBoundPods gets pods that are bound to the current pod.
func (c *GRPCCollaborationClient) GetBoundPods(ctx context.Context) ([]string, error) {
	var result struct {
		Pods []string `json:"pods"`
	}
	if err := c.call(ctx, "get_bound_pods", nil, &result); err != nil {
		return nil, err
	}
	return result.Pods, nil
}
