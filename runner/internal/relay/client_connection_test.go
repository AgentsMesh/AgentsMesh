package relay

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

var testUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func TestConnectClosesDialedSocketWhenStopStateWins(t *testing.T) {
	disconnected := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		if _, _, err := conn.ReadMessage(); err != nil {
			close(disconnected)
		}
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	client := NewClient(context.Background(), url, "pod-1", "test-token", nil)
	// Model Stop after it publishes the stopped state but before it cancels the
	// dial context. Connect must not publish the socket as a new generation.
	client.stopped.Store(true)
	defer client.Stop()

	if err := client.Connect(); !errors.Is(err, errRelayClientStopped) {
		t.Fatalf("Connect after stopped state = %v, want %v", err, errRelayClientStopped)
	}
	client.connMu.RLock()
	conn := client.conn
	client.connMu.RUnlock()
	if conn != nil {
		t.Fatal("Connect retained a socket after outbound activation was rejected")
	}
	select {
	case <-disconnected:
	case <-time.After(time.Second):
		t.Fatal("rejected connection was not closed")
	}
}

func TestConnectAndStop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		// Keep connection open
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c := NewClient(context.TODO(), url, "pod-1", "test-token", nil)

	if err := c.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if !c.IsConnected() {
		t.Error("should be connected")
	}

	c.Start()
	time.Sleep(10 * time.Millisecond)

	c.Stop()
	time.Sleep(10 * time.Millisecond)
	if c.IsConnected() {
		t.Error("should not be connected after stop")
	}
}

func TestHandleMessage(t *testing.T) {
	c := NewClient(context.TODO(), "ws://localhost:8080", "pod-1", "test-token", nil)

	var receivedInput []byte
	c.SetMessageHandler(MsgTypeInput, func(payload []byte) { receivedInput = payload })

	var receivedCols, receivedRows uint16
	c.SetMessageHandler(MsgTypeResize, func(payload []byte) {
		if len(payload) < 4 {
			return
		}
		receivedCols = binary.BigEndian.Uint16(payload[0:2])
		receivedRows = binary.BigEndian.Uint16(payload[2:4])
	})

	// Test input message
	inputMsg := EncodeMessage(MsgTypeInput, []byte("hello"))
	c.handleMessage(inputMsg)
	if string(receivedInput) != "hello" {
		t.Errorf("input: %s", receivedInput)
	}

	// Test resize message (4-byte big-endian: cols=100, rows=50)
	resizePayload := make([]byte, 4)
	binary.BigEndian.PutUint16(resizePayload[0:2], 100)
	binary.BigEndian.PutUint16(resizePayload[2:4], 50)
	resizeMsg := EncodeMessage(MsgTypeResize, resizePayload)
	c.handleMessage(resizeMsg)
	if receivedCols != 100 || receivedRows != 50 {
		t.Errorf("resize: %dx%d", receivedCols, receivedRows)
	}

	// Test invalid message (should not panic)
	c.handleMessage([]byte{})

	// Test message with no registered handler (should not panic)
	unknownMsg := EncodeMessage(0xFF, []byte("unknown"))
	c.handleMessage(unknownMsg)

	// Test pong message (no handler, just alive marker)
	pongMsg := EncodeMessage(MsgTypePong, nil)
	c.handleMessage(pongMsg)
}

func TestSendSnapshot(t *testing.T) {
	received := make(chan struct{})
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
			if len(data) > 0 && data[0] == MsgTypeSnapshot {
				close(received)
				return
			}
		}
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c := NewClient(context.TODO(), url, "pod-1", "test-token", nil)

	if err := c.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	c.Start()

	// Serialize snapshot payload inline (no vt dependency)
	snapshotPayload, _ := json.Marshal(map[string]any{"cols": 80, "rows": 24})
	if err := c.Send(MsgTypeSnapshot, snapshotPayload); err != nil {
		t.Errorf("Send snapshot: %v", err)
	}

	// Wait for snapshot to be received before stopping
	select {
	case <-received:
	case <-time.After(time.Second):
		t.Error("timeout waiting for snapshot")
	}

	c.Stop()
}

func TestGetConnectedAtBeforeConnect(t *testing.T) {
	c := NewClient(context.TODO(), "ws://localhost:8080", "pod-1", "test-token", nil)

	// Before connection, ConnectedAt should be 0
	if c.GetConnectedAt() != 0 {
		t.Errorf("expected 0 before connection, got %d", c.GetConnectedAt())
	}
}

func TestGetConnectedAtAfterConnect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
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

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c := NewClient(context.TODO(), url, "pod-1", "test-token", nil)

	// Before connection
	if c.GetConnectedAt() != 0 {
		t.Errorf("expected 0 before connection, got %d", c.GetConnectedAt())
	}

	beforeConnect := time.Now().UnixMilli()
	if err := c.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	afterConnect := time.Now().UnixMilli()

	// After connection, ConnectedAt should be set
	connectedAt := c.GetConnectedAt()
	if connectedAt == 0 {
		t.Error("ConnectedAt should not be 0 after connection")
	}

	// ConnectedAt should be between beforeConnect and afterConnect
	if connectedAt < beforeConnect || connectedAt > afterConnect {
		t.Errorf("ConnectedAt (%d) should be between %d and %d", connectedAt, beforeConnect, afterConnect)
	}

	c.Stop()
}

func TestGetRelayURL(t *testing.T) {
	c := NewClient(context.TODO(), "wss://relay.example.com", "pod-1", "test-token", nil)

	if c.GetRelayURL() != "wss://relay.example.com" {
		t.Errorf("GetRelayURL: expected wss://relay.example.com, got %s", c.GetRelayURL())
	}
}
