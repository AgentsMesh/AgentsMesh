//go:build integration

package relay

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRelay_CloseCallbackFiresOnce_Integration verifies that the onClose
// handler fires exactly once even when Stop() triggers multiple code paths.
func TestRelay_CloseCallbackFiresOnce_Integration(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := integrationUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	c := NewClient(wsURL(srv), "pod-close", "tok", nil)

	var closeCount atomic.Int32
	c.SetCloseHandler(func() {
		closeCount.Add(1)
	})

	require.NoError(t, c.Connect())
	require.True(t, c.Start())

	// Stop triggers close; calling Stop again should be a no-op.
	c.Stop()
	c.Stop()

	// Give a moment for any stray goroutines that might fire the handler.
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(1), closeCount.Load(), "onClose must fire exactly once")
}

// TestRelay_PingPongHeartbeat_Integration verifies the client responds with
// Pong when it receives a protocol-level Ping (MsgTypePing=0x05).
func TestRelay_PingPongHeartbeat_Integration(t *testing.T) {
	pongReceived := make(chan struct{}, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := integrationUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Send a protocol-level Ping message
		err = conn.WriteMessage(websocket.BinaryMessage, EncodePing())
		require.NoError(t, err)

		// Read the response — expect a Pong message
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		msg, _ := DecodeMessage(data)
		if msg != nil && msg.Type == MsgTypePong {
			pongReceived <- struct{}{}
		}

		// Keep connection alive until client disconnects
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	c := NewClient(wsURL(srv), "pod-ping", "tok", nil)
	require.NoError(t, c.Connect())
	require.True(t, c.Start())
	defer c.Stop()

	select {
	case <-pongReceived:
		// success
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for Pong response")
	}
}

// TestRelay_SendWhileDisconnected_Integration verifies Send returns an error
// when the client is not connected, and does not panic.
func TestRelay_SendWhileDisconnected_Integration(t *testing.T) {
	c := NewClient("ws://127.0.0.1:0/unused", "pod-disc", "tok", nil)

	err := c.Send(MsgTypeOutput, []byte("should fail"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

// TestRelay_ConcurrentHandlerRegistration_Integration registers and
// deregisters handlers from many goroutines while messages are being
// dispatched, verifying no race condition or panic occurs.
func TestRelay_ConcurrentHandlerRegistration_Integration(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := integrationUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		// Echo every message back as MsgTypeInput
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			conn.WriteMessage(websocket.BinaryMessage, data)
		}
	}))
	defer srv.Close()

	c := NewClient(wsURL(srv), "pod-race", "tok", nil)
	require.NoError(t, c.Connect())
	require.True(t, c.Start())
	defer c.Stop()

	const goroutines = 10
	const iterations = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		msgType := byte(MsgTypeInput + g%4) // spread across several types
		go func(mt byte) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				c.SetMessageHandler(mt, func([]byte) {})
				// Interleave with sends to exercise handleMessage path
				_ = c.Send(MsgTypeOutput, []byte{mt, byte(i)})
			}
		}(msgType)
	}

	wg.Wait()
	// If we reach here without panic or race detector failure, the test passes.
}
