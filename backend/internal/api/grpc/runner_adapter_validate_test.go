package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
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

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	identity := &ClientIdentity{
		NodeID:  "non-existent",
		OrgSlug: "test-org",
	}

	_, err := adapter.validateRunner(context.Background(), identity)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "runner not found")
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
	assert.Contains(t, err.Error(), "does not belong to this organization")
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
