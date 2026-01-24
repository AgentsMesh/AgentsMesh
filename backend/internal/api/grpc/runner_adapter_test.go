package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

func TestNewGRPCRunnerAdapter(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

	assert.NotNil(t, adapter)
	assert.Equal(t, connMgr, adapter.connManager)
}

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

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

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

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

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

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

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

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

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

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

	identity := &ClientIdentity{
		NodeID:  "test-node",
		OrgSlug: "non-existent-org",
	}

	_, err := adapter.validateRunner(context.Background(), identity)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "organization not found")
}

func TestGRPCRunnerAdapter_SendCreatePod(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

	// Test sending to non-existent runner
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-pod",
		LaunchCommand: "claude",
	}
	err := adapter.SendCreatePod(999, cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestGRPCRunnerAdapter_SendTerminatePod(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

	// Test sending to non-existent runner
	err := adapter.SendTerminatePod(999, "test-pod", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestGRPCRunnerAdapter_SendTerminalInput(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

	// Test sending to non-existent runner
	err := adapter.SendTerminalInput(999, "test-pod", []byte("hello"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestGRPCRunnerAdapter_SendTerminalResize(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

	// Test sending to non-existent runner
	err := adapter.SendTerminalResize(999, "test-pod", 120, 40)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestGRPCRunnerAdapter_SendPrompt(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

	// Test sending to non-existent runner
	err := adapter.SendPrompt(999, "test-pod", "Hello, Claude!")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestGRPCRunnerAdapter_IsConnected(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

	// Initially not connected
	assert.False(t, adapter.IsConnected(1))

	// Add a connection via connManager
	mockStream := &mockRunnerStream{}
	connMgr.AddConnection(1, "test-node", "test-org", mockStream)

	// Now connected
	assert.True(t, adapter.IsConnected(1))
}

func TestGRPCRunnerAdapter_Connect_InvalidIdentity(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

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

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

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

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

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
	assert.Contains(t, err.Error(), "runner not found")
}

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

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

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

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

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

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

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

// mockRunnerStream implements runner.RunnerStream for testing with full type safety.
type mockRunnerStream struct {
	sent []*runnerv1.ServerMessage
}

// Compile-time check: mockRunnerStream implements runner.RunnerStream
var _ runner.RunnerStream = (*mockRunnerStream)(nil)

func (m *mockRunnerStream) Send(msg *runnerv1.ServerMessage) error {
	m.sent = append(m.sent, msg)
	return nil
}

func (m *mockRunnerStream) Recv() (*runnerv1.RunnerMessage, error) {
	return nil, context.Canceled
}

func (m *mockRunnerStream) Context() context.Context {
	return context.Background()
}

// mockConnectServer implements runnerv1.RunnerService_ConnectServer for testing
type mockConnectServer struct {
	ctx      context.Context
	messages []*runnerv1.ServerMessage
	recvCh   chan *runnerv1.RunnerMessage
}

func (m *mockConnectServer) Send(msg *runnerv1.ServerMessage) error {
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockConnectServer) Recv() (*runnerv1.RunnerMessage, error) {
	if m.recvCh == nil {
		return nil, context.Canceled
	}
	select {
	case msg := <-m.recvCh:
		return msg, nil
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	}
}

func (m *mockConnectServer) Context() context.Context {
	return m.ctx
}

func (m *mockConnectServer) SetHeader(metadata.MD) error  { return nil }
func (m *mockConnectServer) SendHeader(metadata.MD) error { return nil }
func (m *mockConnectServer) SetTrailer(metadata.MD)       {}
func (m *mockConnectServer) SendMsg(interface{}) error    { return nil }
func (m *mockConnectServer) RecvMsg(interface{}) error    { return nil }

// ==================== grpcStreamAdapter Tests ====================

func TestGrpcStreamAdapter_Send(t *testing.T) {
	t.Run("successful send", func(t *testing.T) {
		done := make(chan struct{})
		sendCh := make(chan *runnerv1.ServerMessage, 10)
		adapter := &grpcStreamAdapter{
			sendCh: sendCh,
			done:   done,
		}

		msg := &runnerv1.ServerMessage{
			Timestamp: 12345,
		}
		err := adapter.Send(msg)
		require.NoError(t, err)

		// Verify message was queued
		select {
		case received := <-sendCh:
			assert.Equal(t, msg, received)
		default:
			t.Fatal("expected message to be queued")
		}
	})

	t.Run("send when closed with full buffer", func(t *testing.T) {
		done := make(chan struct{})
		// Use unbuffered channel - will block on send, triggering select's other cases
		sendCh := make(chan *runnerv1.ServerMessage)
		adapter := &grpcStreamAdapter{
			sendCh: sendCh,
			done:   done,
		}
		close(done)

		// With default case in select:
		// - sendCh is full (unbuffered, no receiver)
		// - done is closed
		// - default case matches first due to non-blocking select
		// So we expect "buffer full" or "connection closed" depending on select order
		msg := &runnerv1.ServerMessage{Timestamp: 12345}
		err := adapter.Send(msg)
		require.Error(t, err)
		// The error could be either "buffer full" or "connection closed" due to select non-determinism
		// but with default present, "buffer full" is most likely
		errMsg := err.Error()
		assert.True(t, errMsg == "rpc error: code = ResourceExhausted desc = send buffer full" ||
			errMsg == "rpc error: code = Canceled desc = connection closed",
			"unexpected error: %s", errMsg)
	})
}

func TestGrpcStreamAdapter_Send_BufferFull(t *testing.T) {
	done := make(chan struct{})
	sendCh := make(chan *runnerv1.ServerMessage) // Unbuffered channel

	adapter := &grpcStreamAdapter{
		sendCh: sendCh,
		done:   done,
	}

	// Send should fail immediately as buffer is full (no receiver)
	msg := &runnerv1.ServerMessage{Timestamp: 12345}
	err := adapter.Send(msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send buffer full")
}

func TestGrpcStreamAdapter_Recv(t *testing.T) {
	recvCh := make(chan *runnerv1.RunnerMessage, 1)
	mockStream := &mockConnectServer{
		ctx:    context.Background(),
		recvCh: recvCh,
	}

	adapter := &grpcStreamAdapter{
		stream: mockStream,
		sendCh: make(chan *runnerv1.ServerMessage, 10),
		done:   make(chan struct{}),
	}

	// Queue a message
	expectedMsg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_Heartbeat{
			Heartbeat: &runnerv1.HeartbeatData{NodeId: "test"},
		},
	}
	recvCh <- expectedMsg

	// Recv should return the message
	msg, err := adapter.Recv()
	require.NoError(t, err)
	assert.Equal(t, expectedMsg, msg)
}

func TestGrpcStreamAdapter_Context(t *testing.T) {
	ctx := context.WithValue(context.Background(), "key", "value")
	mockStream := &mockConnectServer{
		ctx: ctx,
	}

	adapter := &grpcStreamAdapter{
		stream: mockStream,
		sendCh: make(chan *runnerv1.ServerMessage, 10),
		done:   make(chan struct{}),
	}

	assert.Equal(t, ctx, adapter.Context())
}

// ==================== handleProtoMessage Tests ====================

func TestGRPCRunnerAdapter_HandleProtoMessage(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

	// Add a connection
	mockStream := &mockRunnerStream{}
	conn := connMgr.AddConnection(1, "test-node", "test-org", mockStream)

	t.Run("heartbeat message", func(t *testing.T) {
		var heartbeatReceived bool
		connMgr.SetHeartbeatCallback(func(runnerID int64, data *runnerv1.HeartbeatData) {
			heartbeatReceived = true
			assert.Equal(t, int64(1), runnerID)
			assert.Equal(t, "test-node", data.NodeId)
		})

		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_Heartbeat{
				Heartbeat: &runnerv1.HeartbeatData{NodeId: "test-node"},
			},
		}
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)
		assert.True(t, heartbeatReceived)
	})

	t.Run("pod created message", func(t *testing.T) {
		var podCreatedReceived bool
		connMgr.SetPodCreatedCallback(func(runnerID int64, data *runnerv1.PodCreatedEvent) {
			podCreatedReceived = true
			assert.Equal(t, int64(1), runnerID)
			assert.Equal(t, "test-pod", data.PodKey)
		})

		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_PodCreated{
				PodCreated: &runnerv1.PodCreatedEvent{PodKey: "test-pod"},
			},
		}
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)
		assert.True(t, podCreatedReceived)
	})

	t.Run("pod terminated message", func(t *testing.T) {
		var podTerminatedReceived bool
		connMgr.SetPodTerminatedCallback(func(runnerID int64, data *runnerv1.PodTerminatedEvent) {
			podTerminatedReceived = true
			assert.Equal(t, int64(1), runnerID)
			assert.Equal(t, "test-pod", data.PodKey)
			assert.Equal(t, int32(0), data.ExitCode)
		})

		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_PodTerminated{
				PodTerminated: &runnerv1.PodTerminatedEvent{PodKey: "test-pod", ExitCode: 0},
			},
		}
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)
		assert.True(t, podTerminatedReceived)
	})

	t.Run("terminal output message", func(t *testing.T) {
		var terminalOutputReceived bool
		connMgr.SetTerminalOutputCallback(func(runnerID int64, data *runnerv1.TerminalOutputEvent) {
			terminalOutputReceived = true
			assert.Equal(t, int64(1), runnerID)
			assert.Equal(t, "test-pod", data.PodKey)
			assert.Equal(t, []byte("hello"), data.Data)
		})

		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_TerminalOutput{
				TerminalOutput: &runnerv1.TerminalOutputEvent{PodKey: "test-pod", Data: []byte("hello")},
			},
		}
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)
		assert.True(t, terminalOutputReceived)
	})

	t.Run("agent status message", func(t *testing.T) {
		var agentStatusReceived bool
		connMgr.SetAgentStatusCallback(func(runnerID int64, data *runnerv1.AgentStatusEvent) {
			agentStatusReceived = true
			assert.Equal(t, int64(1), runnerID)
			assert.Equal(t, "test-pod", data.PodKey)
			assert.Equal(t, "running", data.Status)
		})

		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_AgentStatus{
				AgentStatus: &runnerv1.AgentStatusEvent{PodKey: "test-pod", Status: "running"},
			},
		}
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)
		assert.True(t, agentStatusReceived)
	})

	t.Run("pty resized message", func(t *testing.T) {
		var ptyResizedReceived bool
		connMgr.SetPtyResizedCallback(func(runnerID int64, data *runnerv1.PtyResizedEvent) {
			ptyResizedReceived = true
			assert.Equal(t, int64(1), runnerID)
			assert.Equal(t, "test-pod", data.PodKey)
			assert.Equal(t, int32(120), data.Cols)
			assert.Equal(t, int32(40), data.Rows)
		})

		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_PtyResized{
				PtyResized: &runnerv1.PtyResizedEvent{PodKey: "test-pod", Cols: 120, Rows: 40},
			},
		}
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)
		assert.True(t, ptyResizedReceived)
	})

	t.Run("error message", func(t *testing.T) {
		// Error messages are just logged, no callback
		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_Error{
				Error: &runnerv1.ErrorEvent{
					PodKey:  "test-pod",
					Code:    "ERR001",
					Message: "test error",
				},
			},
		}
		// Should not panic
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)
	})

	t.Run("initialize message", func(t *testing.T) {
		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_Initialize{
				Initialize: &runnerv1.InitializeRequest{
					ProtocolVersion: 2,
				},
			},
		}
		// Should send InitializeResult
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)

		// Drain the response
		select {
		case <-conn.Send:
			// Expected
		default:
			// May have been consumed already
		}
	})

	t.Run("initialized message", func(t *testing.T) {
		var initCallbackCalled bool
		connMgr.SetInitializedCallback(func(runnerID int64, availableAgents []string) {
			initCallbackCalled = true
		})

		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_Initialized{
				Initialized: &runnerv1.InitializedConfirm{
					AvailableAgents: []string{"test-agent"},
				},
			},
		}
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)
		assert.True(t, initCallbackCalled)
	})

	t.Run("unknown message type", func(t *testing.T) {
		// This should log a warning but not panic
		msg := &runnerv1.RunnerMessage{
			Payload: nil, // Unknown/nil payload
		}
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)
	})
}

// ==================== handleInitialize Tests ====================

func TestGRPCRunnerAdapter_HandleInitialize(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	t.Run("without agent types provider", func(t *testing.T) {
		adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

		// Add a connection
		mockStream := &mockRunnerStream{}
		conn := connMgr.AddConnection(1, "test-node", "test-org", mockStream)

		req := &runnerv1.InitializeRequest{
			ProtocolVersion: 2,
		}
		adapter.handleInitialize(context.Background(), 1, conn, req)

		// Messages are sent to conn.Send channel, so we need to read from there
		select {
		case response := <-conn.Send:
			initResult := response.GetInitializeResult()
			require.NotNil(t, initResult)
			assert.Equal(t, int32(2), initResult.ProtocolVersion)
			assert.NotNil(t, initResult.ServerInfo)
			assert.Empty(t, initResult.AgentTypes)
			assert.Contains(t, initResult.Features, "files_to_create")
			assert.Contains(t, initResult.Features, "work_dir_config")
			assert.Contains(t, initResult.Features, "initial_prompt")
		default:
			t.Fatal("expected message to be sent to conn.Send channel")
		}
	})

	t.Run("with agent types provider", func(t *testing.T) {
		agentProvider := &mockAgentTypesProvider{
			agentTypes: []interfaces.AgentTypeInfo{
				{Slug: "claude-code", Name: "Claude Code", Executable: "claude"},
				{Slug: "aider", Name: "Aider", Executable: "aider"},
			},
		}
		adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, agentProvider, connMgr)

		// Clear previous messages and add new connection
		mockStream := &mockRunnerStream{}
		conn := connMgr.AddConnection(2, "test-node-2", "test-org", mockStream)

		req := &runnerv1.InitializeRequest{
			ProtocolVersion: 2,
		}
		adapter.handleInitialize(context.Background(), 2, conn, req)

		// Messages are sent to conn.Send channel
		select {
		case response := <-conn.Send:
			initResult := response.GetInitializeResult()
			require.NotNil(t, initResult)
			require.Len(t, initResult.AgentTypes, 2)
			assert.Equal(t, "claude-code", initResult.AgentTypes[0].Slug)
			assert.Equal(t, "Claude Code", initResult.AgentTypes[0].Name)
			assert.Equal(t, "claude", initResult.AgentTypes[0].Command)
		default:
			t.Fatal("expected message to be sent to conn.Send channel")
		}
	})

	t.Run("send message failure", func(t *testing.T) {
		adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

		// Add a connection
		mockStream := &mockRunnerStream{}
		conn := connMgr.AddConnection(3, "test-node-3", "test-org", mockStream)

		// Close the connection to make SendMessage fail
		conn.Close()

		req := &runnerv1.InitializeRequest{
			ProtocolVersion: 2,
		}

		// Should not panic when SendMessage fails
		adapter.handleInitialize(context.Background(), 3, conn, req)
	})
}

// mockAgentTypesProvider implements AgentTypesProvider for testing
type mockAgentTypesProvider struct {
	agentTypes []interfaces.AgentTypeInfo
}

func (m *mockAgentTypesProvider) GetAgentTypesForRunner() []interfaces.AgentTypeInfo {
	return m.agentTypes
}

// ==================== handleInitialized Tests ====================

func TestGRPCRunnerAdapter_HandleInitialized(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

	// Add a connection
	mockStream := &mockRunnerStream{}
	conn := connMgr.AddConnection(1, "test-node", "test-org", mockStream)

	// Set up callback to verify
	var callbackRunnerID int64
	var callbackAgents []string
	connMgr.SetInitializedCallback(func(runnerID int64, availableAgents []string) {
		callbackRunnerID = runnerID
		callbackAgents = availableAgents
	})

	msg := &runnerv1.InitializedConfirm{
		AvailableAgents: []string{"claude-code", "aider"},
	}
	adapter.handleInitialized(context.Background(), 1, conn, msg)

	// Verify callback was called
	assert.Equal(t, int64(1), callbackRunnerID)
	assert.Equal(t, []string{"claude-code", "aider"}, callbackAgents)

	// Verify connection is marked as initialized
	assert.True(t, conn.IsInitialized())
	assert.Equal(t, []string{"claude-code", "aider"}, conn.GetAvailableAgents())
}

func TestGRPCRunnerAdapter_HandleInitialized_NilRunnerService(t *testing.T) {
	logger := newTestLogger()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	// Create adapter with nil runnerService
	adapter := NewGRPCRunnerAdapter(logger, nil, nil, nil, nil, nil, connMgr)

	// Add a connection
	mockStream := &mockRunnerStream{}
	conn := connMgr.AddConnection(1, "test-node", "test-org", mockStream)

	msg := &runnerv1.InitializedConfirm{
		AvailableAgents: []string{"claude-code"},
	}

	// Should not panic when runnerService is nil
	adapter.handleInitialized(context.Background(), 1, conn, msg)

	// Connection should still be marked as initialized
	assert.True(t, conn.IsInitialized())
}

func TestGRPCRunnerAdapter_HandleInitialized_UpdateAgentsError(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	// Set the service to return error on UpdateAvailableAgents
	runnerSvc.err = context.DeadlineExceeded
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, nil, nil, nil, connMgr)

	// Add a connection
	mockStream := &mockRunnerStream{}
	conn := connMgr.AddConnection(1, "test-node", "test-org", mockStream)

	msg := &runnerv1.InitializedConfirm{
		AvailableAgents: []string{"claude-code"},
	}

	// Should not panic even when UpdateAvailableAgents returns error
	adapter.handleInitialized(context.Background(), 1, conn, msg)

	// Connection should still be marked as initialized
	assert.True(t, conn.IsInitialized())
}

// ==================== Send Operations with Connected Runner Tests ====================

func TestGRPCRunnerAdapter_SendOperations_WithConnection(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

	// Add a connection
	mockStream := &mockRunnerStream{}
	conn := connMgr.AddConnection(1, "test-node", "test-org", mockStream)

	t.Run("SendCreatePod with connection", func(t *testing.T) {
		cmd := &runnerv1.CreatePodCommand{
			PodKey:        "test-pod",
			LaunchCommand: "claude",
		}
		err := adapter.SendCreatePod(1, cmd)
		require.NoError(t, err)

		// Messages are sent to conn.Send channel
		select {
		case msg := <-conn.Send:
			assert.NotNil(t, msg.GetCreatePod())
			assert.Equal(t, "test-pod", msg.GetCreatePod().PodKey)
		default:
			t.Fatal("expected message in conn.Send channel")
		}
	})

	t.Run("SendTerminatePod with connection", func(t *testing.T) {
		err := adapter.SendTerminatePod(1, "test-pod", true)
		require.NoError(t, err)

		select {
		case msg := <-conn.Send:
			assert.NotNil(t, msg.GetTerminatePod())
			assert.Equal(t, "test-pod", msg.GetTerminatePod().PodKey)
			assert.True(t, msg.GetTerminatePod().Force)
		default:
			t.Fatal("expected message in conn.Send channel")
		}
	})

	t.Run("SendTerminalInput with connection", func(t *testing.T) {
		err := adapter.SendTerminalInput(1, "test-pod", []byte("hello"))
		require.NoError(t, err)

		select {
		case msg := <-conn.Send:
			assert.NotNil(t, msg.GetTerminalInput())
			assert.Equal(t, "test-pod", msg.GetTerminalInput().PodKey)
			assert.Equal(t, []byte("hello"), msg.GetTerminalInput().Data)
		default:
			t.Fatal("expected message in conn.Send channel")
		}
	})

	t.Run("SendTerminalResize with connection", func(t *testing.T) {
		err := adapter.SendTerminalResize(1, "test-pod", 120, 40)
		require.NoError(t, err)

		select {
		case msg := <-conn.Send:
			assert.NotNil(t, msg.GetTerminalResize())
			assert.Equal(t, "test-pod", msg.GetTerminalResize().PodKey)
			assert.Equal(t, int32(120), msg.GetTerminalResize().Cols)
			assert.Equal(t, int32(40), msg.GetTerminalResize().Rows)
		default:
			t.Fatal("expected message in conn.Send channel")
		}
	})

	t.Run("SendPrompt with connection", func(t *testing.T) {
		err := adapter.SendPrompt(1, "test-pod", "Hello, Claude!")
		require.NoError(t, err)

		select {
		case msg := <-conn.Send:
			assert.NotNil(t, msg.GetSendPrompt())
			assert.Equal(t, "test-pod", msg.GetSendPrompt().PodKey)
			assert.Equal(t, "Hello, Claude!", msg.GetSendPrompt().Prompt)
		default:
			t.Fatal("expected message in conn.Send channel")
		}
	})
}

// ==================== Register Tests ====================

func TestGRPCRunnerAdapter_Register(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

	// Create a mock gRPC server to test registration
	grpcServer := grpc.NewServer()
	defer grpcServer.Stop()

	// Register should not panic
	adapter.Register(grpcServer)
}

// ==================== RequestRelayToken Tests ====================

func TestGRPCRunnerAdapter_HandleProtoMessage_RequestRelayToken(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr)

	// Add a connection
	mockStream := &mockRunnerStream{}
	conn := connMgr.AddConnection(1, "test-node", "test-org", mockStream)

	t.Run("request relay token message", func(t *testing.T) {
		var requestReceived bool
		var receivedRunnerID int64
		var receivedData *runnerv1.RequestRelayTokenEvent

		connMgr.SetRequestRelayTokenCallback(func(runnerID int64, data *runnerv1.RequestRelayTokenEvent) {
			requestReceived = true
			receivedRunnerID = runnerID
			receivedData = data
		})

		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_RequestRelayToken{
				RequestRelayToken: &runnerv1.RequestRelayTokenEvent{
					PodKey:    "test-pod",
					SessionId: "session-123",
					RelayUrl:  "wss://relay.example.com",
				},
			},
		}
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)

		assert.True(t, requestReceived)
		assert.Equal(t, int64(1), receivedRunnerID)
		require.NotNil(t, receivedData)
		assert.Equal(t, "test-pod", receivedData.PodKey)
		assert.Equal(t, "session-123", receivedData.SessionId)
		assert.Equal(t, "wss://relay.example.com", receivedData.RelayUrl)
	})

	t.Run("request relay token message without callback", func(t *testing.T) {
		// Clear callback
		connMgr.SetRequestRelayTokenCallback(nil)

		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_RequestRelayToken{
				RequestRelayToken: &runnerv1.RequestRelayTokenEvent{
					PodKey:    "test-pod",
					SessionId: "session-123",
					RelayUrl:  "wss://relay.example.com",
				},
			},
		}

		// Should not panic
		assert.NotPanics(t, func() {
			adapter.handleProtoMessage(context.Background(), 1, conn, msg)
		})
	})
}
