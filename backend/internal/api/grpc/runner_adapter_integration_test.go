package grpc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// TestGRPCRunnerAdapter_Connect_Integration tests the full Connect flow.
func TestGRPCRunnerAdapter_Connect_Integration(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	runnerSvc.AddRunner("test-node", RunnerInfo{
		ID: 1, NodeID: "test-node", OrganizationID: 100, IsEnabled: true,
	})
	orgSvc.AddOrg("test-org", OrganizationInfo{ID: 100, Slug: "test-org"})

	agentProvider := &mockAgentsProvider{
		agents: []interfaces.AgentInfo{
			{Slug: "claude-code", Name: "Claude Code", Executable: "claude"},
		},
	}
	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, agentProvider, connMgr, nil)

	addr, cleanup := setupTestServer(t, adapter)
	defer cleanup()

	// Use channel to synchronize callback from gRPC goroutine
	initDone := make(chan struct{}, 1)
	connMgr.SetInitializedCallback(func(runnerID int64, agents []string) {
		initDone <- struct{}{}
	})

	stream, conn, cancel := connectRunner(t, addr, "test-node", "test-org")
	defer cancel()
	defer conn.Close()

	// Complete handshake
	completeHandshake(t, stream, []string{"claude-code"})

	// Wait for callback with timeout
	select {
	case <-initDone:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for initialized callback")
	}
	assert.True(t, connMgr.IsConnected(1))

	// Close
	_ = stream.CloseSend()
}

// TestGRPCRunnerAdapter_SendCommands_Integration tests sending commands.
func TestGRPCRunnerAdapter_SendCommands_Integration(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	runnerSvc.AddRunner("cmd-node", RunnerInfo{
		ID: 2, NodeID: "cmd-node", OrganizationID: 100, IsEnabled: true,
	})
	orgSvc.AddOrg("test-org", OrganizationInfo{ID: 100, Slug: "test-org"})

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	addr, cleanup := setupTestServer(t, adapter)
	defer cleanup()

	stream, conn, cancel := connectRunner(t, addr, "cmd-node", "test-org")
	defer cancel()
	defer conn.Close()

	completeHandshake(t, stream, []string{"claude-code"})
	time.Sleep(50 * time.Millisecond)
	require.True(t, connMgr.IsConnected(2))

	// Test SendCreatePod
	err := adapter.SendCreatePod(2, &runnerv1.CreatePodCommand{
		PodKey: "pod-1", LaunchCommand: "claude",
	})
	require.NoError(t, err)

	msg, err := stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, "pod-1", msg.GetCreatePod().PodKey)

	// Test SendPodInput
	err = adapter.SendPodInput(2, "pod-1", []byte("hello"))
	require.NoError(t, err)

	msg, err = stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), msg.GetPodInput().Data)

	// Test SendTerminatePod
	err = adapter.SendTerminatePod(2, "pod-1", true)
	require.NoError(t, err)

	msg, err = stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, "pod-1", msg.GetTerminatePod().PodKey)

	_ = stream.CloseSend()
}

// NOTE: Additional integration tests (RunnerEvents, Disconnect) are in
// runner_adapter_integration_events_test.go
