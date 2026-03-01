package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/runner"
)

// ==================== ValidateRunner Tests ====================

func TestGRPCRunnerAdapter_ValidateRunner(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	// Setup test data
	runnerSvc.AddRunner("test-node", RunnerInfo{
		ID:             1,
		NodeID:         "test-node",
		OrganizationID: 100,
		IsEnabled:      true,
	})
	orgSvc.AddOrg("test-org", OrganizationInfo{
		ID:   100,
		Slug: "test-org",
	})

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	// Test valid runner
	identity := &ClientIdentity{
		NodeID:  "test-node",
		OrgSlug: "test-org",
	}

	runnerInfo, err := adapter.validateRunner(context.Background(), identity)
	require.NoError(t, err)
	assert.Equal(t, int64(1), runnerInfo.ID)
	assert.Equal(t, "test-node", runnerInfo.NodeID)
}

func TestGRPCRunnerAdapter_ValidateRunner_NotFound(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	orgSvc.AddOrg("test-org", OrganizationInfo{
		ID:   100,
		Slug: "test-org",
	})

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	identity := &ClientIdentity{
		NodeID:  "non-existent",
		OrgSlug: "test-org",
	}

	_, err := adapter.validateRunner(context.Background(), identity)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "runner not found for this organization")
}

func TestGRPCRunnerAdapter_ValidateRunner_Disabled(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	runnerSvc.AddRunner("disabled-node", RunnerInfo{
		ID:             1,
		NodeID:         "disabled-node",
		OrganizationID: 100,
		IsEnabled:      false, // Disabled
	})
	orgSvc.AddOrg("test-org", OrganizationInfo{
		ID:   100,
		Slug: "test-org",
	})

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	identity := &ClientIdentity{
		NodeID:  "disabled-node",
		OrgSlug: "test-org",
	}

	_, err := adapter.validateRunner(context.Background(), identity)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "runner is disabled")
}

func TestGRPCRunnerAdapter_ValidateRunner_WrongOrg(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	runnerSvc.AddRunner("test-node", RunnerInfo{
		ID:             1,
		NodeID:         "test-node",
		OrganizationID: 100, // Belongs to org 100
		IsEnabled:      true,
	})
	orgSvc.AddOrg("other-org", OrganizationInfo{
		ID:   200, // Different org ID
		Slug: "other-org",
	})

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	identity := &ClientIdentity{
		NodeID:  "test-node",
		OrgSlug: "other-org", // Trying to connect to wrong org
	}

	_, err := adapter.validateRunner(context.Background(), identity)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "runner not found for this organization")
}

func TestGRPCRunnerAdapter_ValidateRunner_OrgNotFound(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	runnerSvc.AddRunner("test-node", RunnerInfo{
		ID:             1,
		NodeID:         "test-node",
		OrganizationID: 100,
		IsEnabled:      true,
	})
	// No org added

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	identity := &ClientIdentity{
		NodeID:  "test-node",
		OrgSlug: "non-existent-org",
	}

	_, err := adapter.validateRunner(context.Background(), identity)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "organization not found")
}

func TestGRPCRunnerAdapter_ValidateRunner_SameNodeDifferentOrgs(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	// Same node_id registered in two different organizations
	runnerSvc.AddRunner("shared-node", RunnerInfo{
		ID:             1,
		NodeID:         "shared-node",
		OrganizationID: 100,
		IsEnabled:      true,
	})
	// AddRunner with nodeID "shared-node" will overwrite the nodeID key,
	// but composite keys "shared-node:100" and "shared-node:200" remain separate
	runnerSvc.runners["shared-node:200"] = RunnerInfo{
		ID:             2,
		NodeID:         "shared-node",
		OrganizationID: 200,
		IsEnabled:      true,
	}

	orgSvc.AddOrg("org-a", OrganizationInfo{ID: 100, Slug: "org-a"})
	orgSvc.AddOrg("org-b", OrganizationInfo{ID: 200, Slug: "org-b"})

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	// Connect with org-a credentials → should get runner 1
	identityA := &ClientIdentity{NodeID: "shared-node", OrgSlug: "org-a"}
	infoA, err := adapter.validateRunner(context.Background(), identityA)
	require.NoError(t, err)
	assert.Equal(t, int64(1), infoA.ID)
	assert.Equal(t, int64(100), infoA.OrganizationID)

	// Connect with org-b credentials → should get runner 2
	identityB := &ClientIdentity{NodeID: "shared-node", OrgSlug: "org-b"}
	infoB, err := adapter.validateRunner(context.Background(), identityB)
	require.NoError(t, err)
	assert.Equal(t, int64(2), infoB.ID)
	assert.Equal(t, int64(200), infoB.OrganizationID)
}
