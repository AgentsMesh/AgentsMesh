//go:build integration

package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// setupTestServer creates a gRPC server for testing and returns cleanup function.
func setupTestServer(t *testing.T, adapter *GRPCRunnerAdapter) (string, func()) {
	grpcServer := grpc.NewServer()
	adapter.Register(grpcServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	cleanup := func() {
		grpcServer.Stop()
		listener.Close()
	}

	return listener.Addr().String(), cleanup
}

// connectRunner creates a client connection and stream with timeout.
func connectRunner(t *testing.T, addr, nodeID, orgSlug string) (runnerv1.RunnerService_ConnectClient, *grpc.ClientConn, context.CancelFunc) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	client := runnerv1.NewRunnerServiceClient(conn)

	md := metadata.New(map[string]string{
		MetadataKeyClientCertDN: "CN=" + nodeID,
		MetadataKeyOrgSlug:      orgSlug,
	})

	// Use timeout context to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	ctx = metadata.NewOutgoingContext(ctx, md)

	stream, err := client.Connect(ctx)
	require.NoError(t, err)

	return stream, conn, cancel
}

// completeHandshake performs the initialization handshake.
func completeHandshake(t *testing.T, stream runnerv1.RunnerService_ConnectClient, agents []string) {
	// Send Initialize
	err := stream.Send(&runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_Initialize{
			Initialize: &runnerv1.InitializeRequest{ProtocolVersion: 2},
		},
	})
	require.NoError(t, err)

	// Receive InitializeResult (with timeout via context)
	msg, err := stream.Recv()
	require.NoError(t, err)
	require.NotNil(t, msg.GetInitializeResult())

	// Send Initialized
	err = stream.Send(&runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_Initialized{
			Initialized: &runnerv1.InitializedConfirm{AvailableAgents: agents},
		},
	})
	require.NoError(t, err)
}

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

	agentProvider := &mockAgentTypesProvider{
		agentTypes: []interfaces.AgentTypeInfo{
			{Slug: "claude-code", Name: "Claude Code", Executable: "claude"},
		},
	}
	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, agentProvider, connMgr)

	addr, cleanup := setupTestServer(t, adapter)
	defer cleanup()

	// Track callbacks
	var initializedCalled bool
	connMgr.SetInitializedCallback(func(runnerID int64, agents []string) {
		initializedCalled = true
	})

	stream, conn, cancel := connectRunner(t, addr, "test-node", "test-org")
	defer cancel()
	defer conn.Close()

	// Complete handshake
	completeHandshake(t, stream, []string{"claude-code"})

	// Wait for callback
	time.Sleep(50 * time.Millisecond)
	assert.True(t, initializedCalled)
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

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

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

	// Test SendTerminalInput
	err = adapter.SendTerminalInput(2, "pod-1", []byte("hello"))
	require.NoError(t, err)

	msg, err = stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), msg.GetTerminalInput().Data)

	// Test SendTerminatePod
	err = adapter.SendTerminatePod(2, "pod-1", true)
	require.NoError(t, err)

	msg, err = stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, "pod-1", msg.GetTerminatePod().PodKey)

	_ = stream.CloseSend()
}

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

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

	// Track events
	var podCreatedKey string
	connMgr.SetPodCreatedCallback(func(runnerID int64, data *runnerv1.PodCreatedEvent) {
		podCreatedKey = data.PodKey
	})

	var terminalOutput []byte
	connMgr.SetTerminalOutputCallback(func(runnerID int64, data *runnerv1.TerminalOutputEvent) {
		terminalOutput = data.Data
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

	// Send TerminalOutput event
	err = stream.Send(&runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_TerminalOutput{
			TerminalOutput: &runnerv1.TerminalOutputEvent{PodKey: "pod-123", Data: []byte("output")},
		},
	})
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, []byte("output"), terminalOutput)

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

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

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
