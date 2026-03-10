package client

import (
	"context"
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendWithTimeout_NoStream(t *testing.T) {
	conn := newTestConnection()
	// stream is nil

	err := conn.sendWithTimeout(&runnerv1.RunnerMessage{}, time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stream not connected")
}

func TestSendWithTimeout_Success(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)

	err := conn.sendWithTimeout(&runnerv1.RunnerMessage{}, time.Second)
	assert.NoError(t, err)
}

func TestPerformInitialization_Timeout(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)
	conn.initTimeout = 100 * time.Millisecond
	conn.agentProbe = NewAgentProbe()

	ctx := context.Background()
	err := conn.performInitialization(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestPerformInitialization_ContextCancelled(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)
	conn.initTimeout = 5 * time.Second
	conn.agentProbe = NewAgentProbe()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancel

	err := conn.performInitialization(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context cancelled")
}

func TestPerformInitialization_Stopped(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)
	conn.initTimeout = 5 * time.Second
	conn.agentProbe = NewAgentProbe()
	close(conn.stopCh)

	ctx := context.Background()
	err := conn.performInitialization(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stopped")
}

func TestPerformInitialization_Success(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)
	conn.initTimeout = 2 * time.Second
	conn.runnerVersion = "test-1.0"
	conn.mcpPort = 19000
	conn.agentProbe = NewAgentProbe()

	// Simulate backend sending InitializeResult after we send the init request
	go func() {
		// Wait for init request to be sent
		time.Sleep(50 * time.Millisecond)
		conn.initResultCh <- &runnerv1.InitializeResult{
			ServerInfo: &runnerv1.ServerInfo{Version: "2.0.0"},
			AgentTypes: []*runnerv1.AgentTypeInfo{},
		}
	}()

	ctx := context.Background()
	err := conn.performInitialization(ctx)
	require.NoError(t, err)
	assert.True(t, conn.IsInitialized())
}

func TestPerformInitialization_DrainsStaleResult(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)
	conn.initTimeout = 2 * time.Second
	conn.agentProbe = NewAgentProbe()

	// Pre-fill with stale result
	conn.initResultCh <- &runnerv1.InitializeResult{
		ServerInfo: &runnerv1.ServerInfo{Version: "stale"},
	}

	// Provide fresh result
	go func() {
		time.Sleep(50 * time.Millisecond)
		conn.initResultCh <- &runnerv1.InitializeResult{
			ServerInfo: &runnerv1.ServerInfo{Version: "fresh"},
			AgentTypes: []*runnerv1.AgentTypeInfo{},
		}
	}()

	ctx := context.Background()
	err := conn.performInitialization(ctx)
	require.NoError(t, err)
	assert.True(t, conn.IsInitialized())
}
