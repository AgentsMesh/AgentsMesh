//go:build integration

package grpc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// TestGRPCRunnerAdapter_RunnerEvents_Integration tests runner events.
func TestGRPCRunnerAdapter_RunnerEvents_Integration(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	runnerSvc.AddRunner("event-node", RunnerInfo{
		ID: 3, NodeID: "event-node", OrganizationID: 100, IsEnabled: true,
	})
	orgSvc.AddOrg("test-org", OrganizationInfo{ID: 100, Slug: "test-org"})

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	// Track events
	var podCreatedKey string
	connMgr.SetPodCreatedCallback(func(runnerID int64, data *runnerv1.PodCreatedEvent) {
		podCreatedKey = data.PodKey
	})

	addr, cleanup := setupTestServer(t, adapter)
	defer cleanup()

	stream, conn, cancel := connectRunner(t, addr, "event-node", "test-org")
	defer cancel()
	defer conn.Close()

	completeHandshake(t, stream, []string{})
	time.Sleep(50 * time.Millisecond)

	// Send PodCreated event
	err := stream.Send(&runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_PodCreated{
			PodCreated: &runnerv1.PodCreatedEvent{PodKey: "pod-123", Pid: 12345},
		},
	})
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, "pod-123", podCreatedKey)

	// NOTE: TerminalOutput test removed - terminal output is exclusively streamed via Relay.
	// Runner no longer sends TerminalOutputEvent via gRPC.

	_ = stream.CloseSend()
}

// TestGRPCRunnerAdapter_Disconnect_Integration tests disconnect handling.
func TestGRPCRunnerAdapter_Disconnect_Integration(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	runnerSvc.AddRunner("disconnect-node", RunnerInfo{
		ID: 4, NodeID: "disconnect-node", OrganizationID: 100, IsEnabled: true,
	})
	orgSvc.AddOrg("test-org", OrganizationInfo{ID: 100, Slug: "test-org"})

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	var disconnectCalled bool
	connMgr.SetDisconnectCallback(func(runnerID int64) {
		disconnectCalled = true
	})

	addr, cleanup := setupTestServer(t, adapter)
	defer cleanup()

	stream, conn, cancel := connectRunner(t, addr, "disconnect-node", "test-org")
	defer cancel()

	completeHandshake(t, stream, []string{})
	time.Sleep(50 * time.Millisecond)
	assert.True(t, connMgr.IsConnected(4))

	// Close connection
	_ = stream.CloseSend()
	conn.Close()

	// Wait for disconnect
	time.Sleep(100 * time.Millisecond)
	assert.True(t, disconnectCalled)
	assert.False(t, connMgr.IsConnected(4))
}
