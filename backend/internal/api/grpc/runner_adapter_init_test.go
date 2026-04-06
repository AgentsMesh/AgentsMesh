package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// ==================== handleInitialize Tests ====================

func TestGRPCRunnerAdapter_HandleInitialize(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	t.Run("without agents provider", func(t *testing.T) {
		adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

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
			assert.Empty(t, initResult.Agents)
			assert.Contains(t, initResult.Features, "files_to_create")
			assert.Contains(t, initResult.Features, "work_dir_config")
			assert.Contains(t, initResult.Features, "prompt")
		default:
			t.Fatal("expected message to be sent to conn.Send channel")
		}
	})

	t.Run("with agents provider", func(t *testing.T) {
		agentProvider := &mockAgentsProvider{
			agents: []interfaces.AgentInfo{
				{Slug: "claude-code", Name: "Claude Code", Executable: "claude"},
				{Slug: "aider", Name: "Aider", Executable: "aider"},
			},
		}
		adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, agentProvider, connMgr, nil)

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
			require.Len(t, initResult.Agents, 2)
			assert.Equal(t, "claude-code", initResult.Agents[0].Slug)
			assert.Equal(t, "Claude Code", initResult.Agents[0].Name)
			assert.Equal(t, "claude", initResult.Agents[0].Command)
		default:
			t.Fatal("expected message to be sent to conn.Send channel")
		}
	})

	t.Run("send message failure", func(t *testing.T) {
		adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

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

// mockAgentsProvider implements AgentsProvider for testing
type mockAgentsProvider struct {
	agents []interfaces.AgentInfo
}

func (m *mockAgentsProvider) GetAgentsForRunner() []interfaces.AgentInfo {
	return m.agents
}

// ==================== handleInitialized Tests ====================

func TestGRPCRunnerAdapter_HandleInitialized(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

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
	adapter := NewGRPCRunnerAdapter(logger, nil, nil, nil, nil, nil, connMgr, nil)

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

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, nil, nil, nil, connMgr, nil)

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
