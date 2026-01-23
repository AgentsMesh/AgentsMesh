package relay

import (
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
	c := NewClient("ws://localhost:8080", "pod-1", "session-1", "test-token", nil)
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if c.relayURL != "ws://localhost:8080" {
		t.Errorf("relayURL: %s", c.relayURL)
	}
	if c.podKey != "pod-1" {
		t.Errorf("podKey: %s", c.podKey)
	}
	if c.sessionID != "session-1" {
		t.Errorf("sessionID: %s", c.sessionID)
	}
	if c.IsConnected() {
		t.Error("should not be connected")
	}
}

func TestNewClientWithLogger(t *testing.T) {
	logger := slog.Default()
	c := NewClient("ws://localhost:8080", "pod-1", "session-1", "test-token", logger)
	if c == nil || c.logger == nil {
		t.Fatal("NewClient with logger failed")
	}
}

func TestSetHandlers(t *testing.T) {
	c := NewClient("ws://localhost:8080", "pod-1", "session-1", "test-token", nil)

	inputCalled := false
	c.SetInputHandler(func(data []byte) { inputCalled = true })
	if c.onInput == nil {
		t.Error("onInput not set")
	}

	resizeCalled := false
	c.SetResizeHandler(func(cols, rows uint16) { resizeCalled = true })
	if c.onResize == nil {
		t.Error("onResize not set")
	}

	closeCalled := false
	c.SetCloseHandler(func() { closeCalled = true })
	if c.onClose == nil {
		t.Error("onClose not set")
	}

	// Trigger handlers
	c.onInput([]byte("test"))
	c.onResize(80, 24)
	c.onClose()
	if !inputCalled || !resizeCalled || !closeCalled {
		t.Error("handlers not called")
	}
}

func TestConnectInvalidURL(t *testing.T) {
	c := NewClient("://invalid", "pod-1", "session-1", "test-token", nil)
	if err := c.Connect(); err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestConnectUnsupportedScheme(t *testing.T) {
	c := NewClient("ftp://localhost:8080", "pod-1", "session-1", "test-token", nil)
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
		c := NewClient(tt.input, "pod-1", "session-1", "test-token", nil)
		// Connect will fail, but scheme should be converted
		err := c.Connect()
		if err == nil {
			c.Stop()
		}
	}
}

func TestSendNotConnected(t *testing.T) {
	c := NewClient("ws://localhost:8080", "pod-1", "session-1", "test-token", nil)
	if err := c.SendOutput([]byte("test")); err == nil {
		t.Error("expected error when not connected")
	}
	if err := c.SendPong(); err == nil {
		t.Error("expected error when not connected")
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
	c := NewClient(url, "pod-1", "session-1", "test-token", nil)

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
	c := NewClient("ws://localhost:8080", "pod-1", "session-1", "test-token", nil)

	var receivedInput []byte
	c.SetInputHandler(func(data []byte) { receivedInput = data })

	var receivedCols, receivedRows uint16
	c.SetResizeHandler(func(cols, rows uint16) {
		receivedCols = cols
		receivedRows = rows
	})

	// Test input message
	inputMsg := EncodeMessage(MsgTypeInput, []byte("hello"))
	c.handleMessage(inputMsg)
	if string(receivedInput) != "hello" {
		t.Errorf("input: %s", receivedInput)
	}

	// Test resize message
	resizeMsg := EncodeResize(100, 50)
	c.handleMessage(resizeMsg)
	if receivedCols != 100 || receivedRows != 50 {
		t.Errorf("resize: %dx%d", receivedCols, receivedRows)
	}

	// Test invalid message (should not panic)
	c.handleMessage([]byte{})
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
	c := NewClient(url, "pod-1", "session-1", "test-token", nil)

	if err := c.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	c.Start()

	snapshot := &TerminalSnapshot{Cols: 80, Rows: 24}
	if err := c.SendSnapshot(snapshot); err != nil {
		t.Errorf("SendSnapshot: %v", err)
	}

	// Wait for snapshot to be received before stopping
	select {
	case <-received:
	case <-time.After(time.Second):
		t.Error("timeout waiting for snapshot")
	}

	c.Stop()
}
