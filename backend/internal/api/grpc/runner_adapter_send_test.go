package grpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// ==================== Send Operations with Connected Runner Tests ====================

func TestGRPCRunnerAdapter_SendOperations_WithConnection(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

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

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	// Create a mock gRPC server to test registration
	grpcServer := grpc.NewServer()
	defer grpcServer.Stop()

	// Register should not panic
	adapter.Register(grpcServer)
}
