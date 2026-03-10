package mcp

import (
	"context"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// ==================== DiscoveryClient ====================

// ListAvailablePods lists pods available for collaboration.
func (c *GRPCCollaborationClient) ListAvailablePods(ctx context.Context) ([]tools.AvailablePod, error) {
	var result struct {
		Pods []tools.AvailablePod `json:"pods"`
	}
	if err := c.call(ctx, "list_available_pods", nil, &result); err != nil {
		return nil, err
	}
	return result.Pods, nil
}

// ListRunners returns simplified Runner list with nested Agent info.
func (c *GRPCCollaborationClient) ListRunners(ctx context.Context) ([]tools.RunnerSummary, error) {
	var result struct {
		Runners []tools.RunnerSummary `json:"runners"`
	}
	if err := c.call(ctx, "list_runners", nil, &result); err != nil {
		return nil, err
	}
	return result.Runners, nil
}

// ListRepositories lists repositories configured in the organization.
func (c *GRPCCollaborationClient) ListRepositories(ctx context.Context) ([]tools.Repository, error) {
	var result struct {
		Repositories []tools.Repository `json:"repositories"`
	}
	if err := c.call(ctx, "list_repositories", nil, &result); err != nil {
		return nil, err
	}
	return result.Repositories, nil
}
