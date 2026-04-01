package grpc

import (
	"context"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
)

// authenticatePod verifies the Pod exists and belongs to the Runner's organization.
// Authenticates the pod and builds tenant context.
func (a *GRPCRunnerAdapter) authenticatePod(ctx context.Context, podKey, orgSlug string) (*middleware.TenantContext, error) {
	if podKey == "" {
		return nil, fmt.Errorf("pod_key is required")
	}

	// Lookup Pod by key
	pod, err := a.podService.GetPodByKey(ctx, podKey)
	if err != nil {
		return nil, fmt.Errorf("invalid pod key")
	}

	// Lookup Organization by slug
	org, err := a.orgService.GetBySlug(ctx, orgSlug)
	if err != nil {
		return nil, fmt.Errorf("organization not found")
	}

	// Verify Pod belongs to this organization
	if pod.OrganizationID != org.ID {
		return nil, fmt.Errorf("pod does not belong to this organization")
	}

	return &middleware.TenantContext{
		OrganizationID:   org.ID,
		OrganizationSlug: org.Slug,
		UserID:           pod.CreatedByID,
		UserRole:         "pod",
	}, nil
}
