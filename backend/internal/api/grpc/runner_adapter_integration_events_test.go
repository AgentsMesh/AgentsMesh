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

	// Use channel to synchronize callback
	podCreatedCh := make(chan string, 1)
	connMgr.SetPodCreatedCallback(func(runnerID int64, data *runnerv1.PodCreatedEvent) {
		podCreatedCh <- data.PodKey
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

	// Wait for callback with timeout
	select {
	case key := <-podCreatedCh:
		assert.Equal(t, "pod-123", key)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for pod_created callback")
	}

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

	// Use channel to synchronize callback
	disconnectCh := make(chan struct{}, 1)
	connMgr.SetDisconnectCallback(func(runnerID int64) {
		disconnectCh <- struct{}{}
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

	// Wait for disconnect callback with timeout
	select {
	case <-disconnectCh:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for disconnect callback")
	}
	assert.False(t, connMgr.IsConnected(4))
}
