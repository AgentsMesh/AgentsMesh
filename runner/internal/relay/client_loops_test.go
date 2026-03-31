package relay

import (
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReadLoop_PingRespondsWithPong verifies readLoop sends pong when receiving ping.
func TestReadLoop_PingRespondsWithPong(t *testing.T) {
	gotPong := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Send a ping message
		pingMsg := EncodePing()
		conn.WriteMessage(websocket.BinaryMessage, pingMsg)

		// Wait for pong response
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if len(data) > 0 && data[0] == MsgTypePong {
				close(gotPong)
				return
			}
		}
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c := NewClient(url, "pod-1", "test-token", nil)
	require.NoError(t, c.Connect())
	c.Start()

	select {
	case <-gotPong:
		// Success
	case <-time.After(3 * time.Second):
		t.Error("timeout waiting for pong response to ping")
	}
	c.Stop()
}

// TestReadLoop_RoutesMessageToHandler verifies readLoop dispatches to registered handler.
func TestReadLoop_RoutesMessageToHandler(t *testing.T) {
	var received atomic.Value

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Send an ACP event message
		msg := EncodeMessage(MsgTypeAcpEvent, []byte(`{"event":"test"}`))
		conn.WriteMessage(websocket.BinaryMessage, msg)

		// Keep connection alive briefly
		time.Sleep(500 * time.Millisecond)
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c := NewClient(url, "pod-1", "test-token", nil)
	c.SetMessageHandler(MsgTypeAcpEvent, func(payload []byte) {
		received.Store(string(payload))
	})

	require.NoError(t, c.Connect())
	c.Start()

	// Poll for message delivery
	deadline := time.After(2 * time.Second)
	for {
		if v := received.Load(); v != nil {
			assert.Equal(t, `{"event":"test"}`, v.(string))
			break
		}
		select {
		case <-deadline:
			t.Fatal("timeout waiting for message delivery")
		case <-time.After(10 * time.Millisecond):
		}
	}
	c.Stop()
}

// TestWriteLoop_SendsQueuedMessages verifies writeLoop delivers queued messages.
func TestWriteLoop_SendsQueuedMessages(t *testing.T) {
	var mu sync.Mutex
	var serverReceived []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if len(data) > 0 && data[0] == MsgTypeOutput {
				mu.Lock()
				serverReceived = data[1:] // strip type byte
				mu.Unlock()
				return
			}
		}
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c := NewClient(url, "pod-1", "test-token", nil)
	require.NoError(t, c.Connect())
	c.Start()

	require.NoError(t, c.Send(MsgTypeOutput, []byte("hello world")))

	deadline := time.After(2 * time.Second)
	for {
		mu.Lock()
		got := serverReceived
		mu.Unlock()
		if got != nil {
			assert.Equal(t, "hello world", string(got))
			break
		}
		select {
		case <-deadline:
			t.Fatal("timeout waiting for server to receive message")
		case <-time.After(10 * time.Millisecond):
		}
	}
	c.Stop()
}

// TestReadLoop_ResizeMessage verifies resize messages are decoded correctly.
func TestReadLoop_ResizeMessage(t *testing.T) {
	var gotCols, gotRows atomic.Uint32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		payload := make([]byte, 4)
		binary.BigEndian.PutUint16(payload[0:2], 120)
		binary.BigEndian.PutUint16(payload[2:4], 40)
		conn.WriteMessage(websocket.BinaryMessage, EncodeMessage(MsgTypeResize, payload))
		time.Sleep(500 * time.Millisecond)
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c := NewClient(url, "pod-1", "test-token", nil)
	c.SetMessageHandler(MsgTypeResize, func(p []byte) {
		if len(p) >= 4 {
			gotCols.Store(uint32(binary.BigEndian.Uint16(p[0:2])))
			gotRows.Store(uint32(binary.BigEndian.Uint16(p[2:4])))
		}
	})
	require.NoError(t, c.Connect())
	c.Start()

	deadline := time.After(2 * time.Second)
	for {
		if gotCols.Load() != 0 {
			assert.Equal(t, uint32(120), gotCols.Load())
			assert.Equal(t, uint32(40), gotRows.Load())
			break
		}
		select {
		case <-deadline:
			t.Fatal("timeout")
		case <-time.After(10 * time.Millisecond):
		}
	}
	c.Stop()
}
