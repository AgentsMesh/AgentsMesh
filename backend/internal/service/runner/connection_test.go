package runner

import (
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGRPCConnection(t *testing.T) {
	stream := newMockRunnerStream()
	defer stream.Close()

	conn := NewGRPCConnection(1, 1, "test-node", "test-org", stream)

	assert.Equal(t, int64(1), conn.RunnerID)
	assert.Equal(t, "test-node", conn.NodeID)
	assert.Equal(t, "test-org", conn.OrgSlug)
	assert.Equal(t, stream, conn.Stream)
	assert.NotNil(t, conn.Send)
	assert.NotNil(t, conn.closeChan)
	assert.False(t, conn.closed)
	assert.False(t, conn.initialized)
	assert.WithinDuration(t, time.Now(), conn.ConnectedAt, time.Second)
	assert.WithinDuration(t, time.Now(), conn.LastPing, time.Second)
}

func TestGRPCConnection_IsInitialized(t *testing.T) {
	stream := newMockRunnerStream()
	defer stream.Close()

	conn := NewGRPCConnection(1, 1, "test-node", "test-org", stream)

	// Initially not initialized
	assert.False(t, conn.IsInitialized())

	// Set initialized
	conn.SetInitialized(true, []string{"claude-code"})
	assert.True(t, conn.IsInitialized())

	// Set back to not initialized
	conn.SetInitialized(false, nil)
	assert.False(t, conn.IsInitialized())
}

func TestGRPCConnection_SetInitialized(t *testing.T) {
	stream := newMockRunnerStream()
	defer stream.Close()

	conn := NewGRPCConnection(1, 1, "test-node", "test-org", stream)

	agents := []string{"claude-code", "aider"}
	conn.SetInitialized(true, agents)

	assert.True(t, conn.IsInitialized())
	assert.Equal(t, agents, conn.GetAvailableAgents())
}

func TestGRPCConnection_GetAvailableAgents(t *testing.T) {
	stream := newMockRunnerStream()
	defer stream.Close()

	conn := NewGRPCConnection(1, 1, "test-node", "test-org", stream)

	// Initially nil
	assert.Nil(t, conn.GetAvailableAgents())

	// Set agents
	agents := []string{"claude-code", "aider", "gemini"}
	conn.SetInitialized(true, agents)

	result := conn.GetAvailableAgents()
	assert.Equal(t, agents, result)

	// Verify it's a copy (modify result should not affect internal state)
	result[0] = "modified"
	assert.NotEqual(t, result[0], conn.GetAvailableAgents()[0])
}

func TestGRPCConnection_UpdateLastPing(t *testing.T) {
	stream := newMockRunnerStream()
	defer stream.Close()

	conn := NewGRPCConnection(1, 1, "test-node", "test-org", stream)
	initialPing := conn.GetLastPing()

	time.Sleep(10 * time.Millisecond)
	conn.UpdateLastPing()

	assert.True(t, conn.GetLastPing().After(initialPing))
}

func TestGRPCConnection_GetLastPing(t *testing.T) {
	stream := newMockRunnerStream()
	defer stream.Close()

	conn := NewGRPCConnection(1, 1, "test-node", "test-org", stream)

	ping := conn.GetLastPing()
	assert.WithinDuration(t, time.Now(), ping, time.Second)
}

func TestGRPCConnection_LastPong_InitiallyZero(t *testing.T) {
	stream := newMockRunnerStream()
	defer stream.Close()

	conn := NewGRPCConnection(1, 1, "test-node", "test-org", stream)

	// Initially zero (no pong received yet)
	assert.True(t, conn.GetLastPong().IsZero())
}

func TestGRPCConnection_UpdateLastPong(t *testing.T) {
	stream := newMockRunnerStream()
	defer stream.Close()

	conn := NewGRPCConnection(1, 1, "test-node", "test-org", stream)
	assert.True(t, conn.GetLastPong().IsZero())

	// Update lastPong
	conn.UpdateLastPong()

	lastPong := conn.GetLastPong()
	assert.False(t, lastPong.IsZero())
	assert.WithinDuration(t, time.Now(), lastPong, time.Second)
}

func TestGRPCConnection_UpdateLastPong_Advances(t *testing.T) {
	stream := newMockRunnerStream()
	defer stream.Close()

	conn := NewGRPCConnection(1, 1, "test-node", "test-org", stream)

	conn.UpdateLastPong()
	first := conn.GetLastPong()

	time.Sleep(10 * time.Millisecond)
	conn.UpdateLastPong()
	second := conn.GetLastPong()

	assert.True(t, second.After(first), "second UpdateLastPong should advance the timestamp")
}

func TestGRPCConnection_IsClosed(t *testing.T) {
	stream := newMockRunnerStream()
	defer stream.Close()

	conn := NewGRPCConnection(1, 1, "test-node", "test-org", stream)

	assert.False(t, conn.IsClosed())

	conn.Close()

	assert.True(t, conn.IsClosed())
}

func TestGRPCConnection_Close(t *testing.T) {
	stream := newMockRunnerStream()

	conn := NewGRPCConnection(1, 1, "test-node", "test-org", stream)

	// Close should work
	conn.Close()
	assert.True(t, conn.IsClosed())

	// Double close should not panic (idempotent)
	conn.Close()
	assert.True(t, conn.IsClosed())
}

func TestGRPCConnection_CloseChan(t *testing.T) {
	stream := newMockRunnerStream()

	conn := NewGRPCConnection(1, 1, "test-node", "test-org", stream)

	closeChan := conn.CloseChan()
	assert.NotNil(t, closeChan)

	// Channel should not be closed initially
	select {
	case <-closeChan:
		t.Fatal("close channel should not be closed yet")
	default:
		// expected
	}

	// Close the connection
	conn.Close()

	// Channel should now be closed
	select {
	case <-closeChan:
		// expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("close channel should be closed after conn.Close()")
	}
}

func TestGRPCConnection_SendMessage(t *testing.T) {
	stream := newMockRunnerStream()
	defer stream.Close()

	conn := NewGRPCConnection(1, 1, "test-node", "test-org", stream)

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_InitializeResult{
			InitializeResult: &runnerv1.InitializeResult{
				ProtocolVersion: 1,
			},
		},
	}

	// Send message should succeed
	err := conn.SendMessage(msg)
	require.NoError(t, err)

	// Verify message was queued
	select {
	case received := <-conn.Send:
		assert.Equal(t, msg, received)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("message should be in send channel")
	}
}

func TestGRPCConnection_SendMessage_Closed(t *testing.T) {
	stream := newMockRunnerStream()

	conn := NewGRPCConnection(1, 1, "test-node", "test-org", stream)
	conn.Close()

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_InitializeResult{
			InitializeResult: &runnerv1.InitializeResult{
				ProtocolVersion: 1,
			},
		},
	}

	// Send message should fail when connection is closed
	err := conn.SendMessage(msg)
	assert.Equal(t, ErrConnectionClosed, err)
}

func TestGRPCConnection_SendMessage_BufferFull(t *testing.T) {
	stream := newMockRunnerStream()
	defer stream.Close()

	conn := NewGRPCConnection(1, 1, "test-node", "test-org", stream)

	// Fill up the send buffer (buffer size is 256)
	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_InitializeResult{
			InitializeResult: &runnerv1.InitializeResult{
				ProtocolVersion: 1,
			},
		},
	}

	for i := 0; i < 256; i++ {
		err := conn.SendMessage(msg)
		require.NoError(t, err)
	}

	// Next message should fail with buffer full
	err := conn.SendMessage(msg)
	assert.Equal(t, ErrSendBufferFull, err)
}
