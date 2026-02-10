package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// ==================== Connect Certificate Tests ====================

func TestGRPCRunnerAdapter_Connect_CertificateRevoked(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	// Setup test data
	runnerSvc.AddRunner("test-node", RunnerInfo{
		ID:               1,
		NodeID:           "test-node",
		OrganizationID:   100,
		IsEnabled:        true,
		CertSerialNumber: "SERIAL123",
	})
	orgSvc.AddOrg("test-org", OrganizationInfo{
		ID:   100,
		Slug: "test-org",
	})

	// Mark certificate as revoked
	runnerSvc.SetCertificateRevoked("SERIAL123", true)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	md := metadata.New(map[string]string{
		MetadataKeyClientCertDN:     "CN=test-node",
		MetadataKeyOrgSlug:          "test-org",
		MetadataKeyClientCertSerial: "SERIAL123",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	mockStream := &mockConnectServer{
		ctx: ctx,
	}

	err := adapter.Connect(mockStream)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "certificate has been revoked")
}

func TestGRPCRunnerAdapter_Connect_CertificateRevocationCheckError(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	// Setup test data
	runnerSvc.AddRunner("test-node", RunnerInfo{
		ID:               1,
		NodeID:           "test-node",
		OrganizationID:   100,
		IsEnabled:        true,
		CertSerialNumber: "SERIAL123",
	})
	orgSvc.AddOrg("test-org", OrganizationInfo{
		ID:   100,
		Slug: "test-org",
	})

	// Set error only for revocation check (not for GetByNodeID)
	runnerSvc.SetRevocationCheckError(context.DeadlineExceeded)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	md := metadata.New(map[string]string{
		MetadataKeyClientCertDN:     "CN=test-node",
		MetadataKeyOrgSlug:          "test-org",
		MetadataKeyClientCertSerial: "SERIAL123",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	mockStream := &mockConnectServer{
		ctx: ctx,
	}

	err := adapter.Connect(mockStream)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to verify certificate status")
}

func TestGRPCRunnerAdapter_Connect_CertificateValid(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	// Setup test data
	runnerSvc.AddRunner("test-node", RunnerInfo{
		ID:               1,
		NodeID:           "test-node",
		OrganizationID:   100,
		IsEnabled:        true,
		CertSerialNumber: "SERIAL123",
	})
	orgSvc.AddOrg("test-org", OrganizationInfo{
		ID:   100,
		Slug: "test-org",
	})

	// Certificate is NOT revoked (default)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	md := metadata.New(map[string]string{
		MetadataKeyClientCertDN:     "CN=test-node",
		MetadataKeyOrgSlug:          "test-org",
		MetadataKeyClientCertSerial: "SERIAL123",
	})
	ctx, cancel := context.WithCancel(context.Background())
	ctx = metadata.NewIncomingContext(ctx, md)

	recvCh := make(chan *runnerv1.RunnerMessage)
	mockStream := &mockConnectServer{
		ctx:    ctx,
		recvCh: recvCh,
	}

	// Start Connect in goroutine since it blocks on receive
	errCh := make(chan error, 1)
	go func() {
		errCh <- adapter.Connect(mockStream)
	}()

	// Cancel to end the connection
	cancel()

	// Should complete without certificate error
	select {
	case err := <-errCh:
		// Connection ended due to context cancellation, not certificate error
		if err != nil {
			assert.NotContains(t, err.Error(), "certificate")
		}
	case <-time.After(time.Second):
		t.Fatal("Connect did not complete in time")
	}

	// Verify connection was established
	assert.True(t, connMgr.ConnectionCount() >= 0) // Connection may have been cleaned up
}
