package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/runner"
)

// ==================== Connect Tests ====================

func TestGRPCRunnerAdapter_Connect_InvalidIdentity(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	// Create mock stream with no metadata
	mockStream := &mockConnectServer{
		ctx: context.Background(),
	}

	err := adapter.Connect(mockStream)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no metadata in context")
}

func TestGRPCRunnerAdapter_Connect_MissingNodeID(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	// Create mock stream with metadata missing node_id
	md := metadata.New(map[string]string{
		MetadataKeyOrgSlug: "test-org",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	mockStream := &mockConnectServer{
		ctx: ctx,
	}

	err := adapter.Connect(mockStream)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing client certificate CN")
}

func TestGRPCRunnerAdapter_Connect_RunnerNotFound(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService() // No runners added
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	// Add org so validateRunner can find it, but no runner is registered
	orgSvc.AddOrg("test-org", OrganizationInfo{
		ID:   100,
		Slug: "test-org",
	})

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	md := metadata.New(map[string]string{
		MetadataKeyClientCertDN: "CN=non-existent-node",
		MetadataKeyOrgSlug:      "test-org",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	mockStream := &mockConnectServer{
		ctx: ctx,
	}

	err := adapter.Connect(mockStream)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "runner not found for this organization")
}

// NOTE: Certificate-related Connect tests are in runner_adapter_connect_cert_test.go
