package grpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
)

// ==================== Additional Send Tests ====================

func TestGRPCRunnerAdapter_SendTerminalRedraw(t *testing.T) {
	logger := newTestLogger()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, nil, nil, nil, nil, connMgr, nil)

	t.Run("runner not connected", func(t *testing.T) {
		err := adapter.SendTerminalRedraw(999, "pod-1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})

	t.Run("successful send", func(t *testing.T) {
		mockStream := &mockRunnerStream{}
		connMgr.AddConnection(1, "test-node", "test-org", mockStream)

		err := adapter.SendTerminalRedraw(1, "pod-1")
		require.NoError(t, err)
	})
}

func TestGRPCRunnerAdapter_SendSubscribeTerminal(t *testing.T) {
	logger := newTestLogger()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, nil, nil, nil, nil, connMgr, nil)

	t.Run("runner not connected", func(t *testing.T) {
		err := adapter.SendSubscribeTerminal(999, "pod-1", "ws://relay", "token", true, 100)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})

	t.Run("successful send", func(t *testing.T) {
		mockStream := &mockRunnerStream{}
		connMgr.AddConnection(2, "test-node", "test-org", mockStream)

		err := adapter.SendSubscribeTerminal(2, "pod-1", "ws://relay", "token", true, 100)
		require.NoError(t, err)
	})
}

func TestGRPCRunnerAdapter_SendUnsubscribeTerminal(t *testing.T) {
	logger := newTestLogger()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, nil, nil, nil, nil, connMgr, nil)

	t.Run("runner not connected", func(t *testing.T) {
		err := adapter.SendUnsubscribeTerminal(999, "pod-1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})

	t.Run("successful send", func(t *testing.T) {
		mockStream := &mockRunnerStream{}
		connMgr.AddConnection(3, "test-node", "test-org", mockStream)

		err := adapter.SendUnsubscribeTerminal(3, "pod-1")
		require.NoError(t, err)
	})
}

func TestGRPCRunnerAdapter_SendQuerySandboxes(t *testing.T) {
	logger := newTestLogger()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, nil, nil, nil, nil, connMgr, nil)

	t.Run("runner not connected", func(t *testing.T) {
		err := adapter.SendQuerySandboxes(999, "req-1", []string{"pod-1", "pod-2"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})

	t.Run("successful send", func(t *testing.T) {
		mockStream := &mockRunnerStream{}
		connMgr.AddConnection(4, "test-node", "test-org", mockStream)

		err := adapter.SendQuerySandboxes(4, "req-1", []string{"pod-1", "pod-2"})
		require.NoError(t, err)
	})
}
