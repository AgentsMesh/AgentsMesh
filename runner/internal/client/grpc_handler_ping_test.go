package client

import (
	"context"
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// mockBidiStream is a minimal mock of grpc.BidiStreamingClient for testing.
// Only provides nil-check safety — sendControl only checks stream != nil.
type mockBidiStream struct {
	grpc.ClientStream
}

func (m *mockBidiStream) Send(_ *runnerv1.RunnerMessage) error { return nil }
func (m *mockBidiStream) Recv() (*runnerv1.ServerMessage, error) {
	return nil, nil
}
func (m *mockBidiStream) Header() (metadata.MD, error) { return nil, nil }
func (m *mockBidiStream) Trailer() metadata.MD         { return nil }
func (m *mockBidiStream) CloseSend() error             { return nil }
func (m *mockBidiStream) Context() context.Context     { return context.Background() }
func (m *mockBidiStream) SendMsg(_ interface{}) error  { return nil }
func (m *mockBidiStream) RecvMsg(_ interface{}) error  { return nil }

// setMockStream sets the GRPCConnection stream to a mock for testing,
// allowing sendControl to pass the nil-check.
func setMockStream(c *GRPCConnection) {
	c.mu.Lock()
	c.stream = &mockBidiStream{}
	c.initialized = true
	c.mu.Unlock()
}

// TestHandlePing verifies that receiving a PingCommand sends back a PongEvent
// with the original timestamp echoed.
func TestHandlePing(t *testing.T) {
	conn := NewGRPCConnection("localhost:9443", "test-node", "test-org", "", "", "")
	setMockStream(conn)

	pingTimestamp := time.Now().UnixMilli()
	ping := &runnerv1.PingCommand{
		Timestamp: pingTimestamp,
	}

	conn.handlePing(ping)

	// Verify pong was queued in control channel
	select {
	case msg := <-conn.controlCh:
		pong := msg.GetPong()
		require.NotNil(t, pong, "expected PongEvent payload")
		assert.Equal(t, pingTimestamp, pong.PingTimestamp,
			"pong should echo the original ping timestamp")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected pong message in control channel")
	}
}

// TestHandlePing_StreamNotConnected verifies handlePing gracefully handles
// the case where stream is nil (no crash, no panic).
func TestHandlePing_StreamNotConnected(t *testing.T) {
	conn := NewGRPCConnection("localhost:9443", "test-node", "test-org", "", "", "")
	// stream is nil by default, sendControl should return error but not panic

	ping := &runnerv1.PingCommand{
		Timestamp: time.Now().UnixMilli(),
	}

	// Should not panic
	conn.handlePing(ping)

	// Control channel should be empty (sendControl returns error for nil stream)
	select {
	case <-conn.controlCh:
		t.Fatal("no pong expected when stream is nil")
	default:
		// expected
	}
}

// TestHandleServerMessage_PingDispatch verifies that PingCommand is routed
// to handlePing via handleServerMessage (synchronous, not async).
func TestHandleServerMessage_PingDispatch(t *testing.T) {
	conn := NewGRPCConnection("localhost:9443", "test-node", "test-org", "", "", "")
	setMockStream(conn)

	pingTimestamp := time.Now().UnixMilli()
	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_Ping{
			Ping: &runnerv1.PingCommand{
				Timestamp: pingTimestamp,
			},
		},
	}

	conn.handleServerMessage(msg)

	// Verify pong response
	select {
	case reply := <-conn.controlCh:
		pong := reply.GetPong()
		require.NotNil(t, pong)
		assert.Equal(t, pingTimestamp, pong.PingTimestamp)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected pong in control channel after handleServerMessage")
	}
}
