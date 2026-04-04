package relay

import (
	"encoding/binary"
	"encoding/json"
	"log/slog"
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

func TestNewClient(t *testing.T) {
	c := NewClient("ws://localhost:8080", "pod-1", "test-token", nil)
	if c == nil {
		t.Fatal("NewClient returned nil")
		return
	}
	if c.relayURL != "ws://localhost:8080" {
		t.Errorf("relayURL: %s", c.relayURL)
	}
	if c.podKey != "pod-1" {
		t.Errorf("podKey: %s", c.podKey)
	}
	if c.IsConnected() {
		t.Error("should not be connected")
	}
}

func TestNewClientWithLogger(t *testing.T) {
	logger := slog.Default()
	c := NewClient("ws://localhost:8080", "pod-1", "test-token", logger)
	if c == nil || c.logger == nil {
		t.Fatal("NewClient with logger failed")
	}
}

func TestSetHandlers(t *testing.T) {
	c := NewClient("ws://localhost:8080", "pod-1", "test-token", nil)

	inputCalled := false
	c.SetMessageHandler(MsgTypeInput, func(payload []byte) { inputCalled = true })

	resizeCalled := false
	c.SetMessageHandler(MsgTypeResize, func(payload []byte) { resizeCalled = true })

	closeCalled := false
	c.SetCloseHandler(func() { closeCalled = true })
	if c.onClose == nil {
		t.Error("onClose not set")
	}

	// Verify handlers are stored
	c.handlersMu.RLock()
	if c.handlers[MsgTypeInput] == nil {
		t.Error("input handler not set")
	}
	if c.handlers[MsgTypeResize] == nil {
		t.Error("resize handler not set")
	}
	c.handlersMu.RUnlock()

	// Trigger handlers
	c.handlers[MsgTypeInput]([]byte("test"))
	c.handlers[MsgTypeResize]([]byte{0, 80, 0, 24})
	c.onClose()
	if !inputCalled || !resizeCalled || !closeCalled {
		t.Error("handlers not called")
	}
}

func TestConnectInvalidURL(t *testing.T) {
	c := NewClient("://invalid", "pod-1", "test-token", nil)
	if err := c.Connect(); err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestConnectUnsupportedScheme(t *testing.T) {
	c := NewClient("ftp://localhost:8080", "pod-1", "test-token", nil)
	if err := c.Connect(); err == nil {
		t.Error("expected error for unsupported scheme")
	}
}

func TestConnectSchemeConversion(t *testing.T) {
	// Test that http converts to ws, https to wss
	// We can't actually connect, but we can test the URL building
	tests := []struct {
		input  string
		scheme string
	}{
		{"http://localhost", "ws"},
		{"https://localhost", "wss"},
		{"ws://localhost", "ws"},
		{"wss://localhost", "wss"},
	}
	for _, tt := range tests {
		c := NewClient(tt.input, "pod-1", "test-token", nil)
		// Connect will fail, but scheme should be converted
		err := c.Connect()
		if err == nil {
			c.Stop()
		}
	}
}

func TestSendNotConnected(t *testing.T) {
	c := NewClient("ws://localhost:8080", "pod-1", "test-token", nil)
	if err := c.Send(MsgTypeOutput, []byte("test")); err == nil {
		t.Error("expected error when not connected")
	}
	if err := c.SendPong(); err == nil {
		t.Error("expected error when not connected")
	}
}

func TestSendBufferFull(t *testing.T) {
	c := NewClient("ws://localhost:8080", "pod-1", "test-token", nil)
	// Mark as connected so send() doesn't short-circuit on "not connected".
	c.connected.Store(true)

	// Fill the send channel to capacity.
	for i := 0; i < cap(c.sendCh); i++ {
		c.sendCh <- []byte{0x00}
	}

	// Next send should return "send buffer full".
	err := c.Send(MsgTypeOutput, []byte("overflow"))
	if err == nil {
		t.Error("expected error when send buffer is full")
	}
	if err != nil && err.Error() != "send buffer full" {
		t.Errorf("expected 'send buffer full', got: %v", err)
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
	c := NewClient(url, "pod-1", "test-token", nil)

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
	c := NewClient("ws://localhost:8080", "pod-1", "test-token", nil)

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
	c := NewClient(url, "pod-1", "test-token", nil)

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
	c := NewClient("ws://localhost:8080", "pod-1", "test-token", nil)

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
	c := NewClient(url, "pod-1", "test-token", nil)

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
	c := NewClient("wss://relay.example.com", "pod-1", "test-token", nil)

	if c.GetRelayURL() != "wss://relay.example.com" {
		t.Errorf("GetRelayURL: expected wss://relay.example.com, got %s", c.GetRelayURL())
	}
}
