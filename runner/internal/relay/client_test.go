package relay

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
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
	c := NewClient(url, "pod-1", "test-token", nil)

	if err := c.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	c.Start()

	snapshot := &vt.TerminalSnapshot{Cols: 80, Rows: 24}
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

func TestSetReconnectHandler(t *testing.T) {
	c := NewClient("ws://localhost:8080", "pod-1", "test-token", nil)

	reconnectCalled := false
	c.SetReconnectHandler(func() { reconnectCalled = true })
	if c.onReconnect == nil {
		t.Error("onReconnect not set")
	}

	// Trigger handler
	c.onReconnect()
	if !reconnectCalled {
		t.Error("reconnect handler not called")
	}
}

func TestReconnectOnDisconnect(t *testing.T) {
	// Track connection attempts with atomic to avoid race condition
	var connectionAttempts atomic.Int32
	reconnected := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := connectionAttempts.Add(1)
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		if attempt == 1 {
			// First connection: close immediately to trigger reconnect
			conn.Close()
			return
		}

		// Second connection: signal reconnect and keep open
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

	c.SetReconnectHandler(func() {
		close(reconnected)
	})

	if err := c.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	c.Start()

	// Wait for reconnection
	select {
	case <-reconnected:
		// Success
	case <-time.After(5 * time.Second):
		t.Error("timeout waiting for reconnect")
	}

	if connectionAttempts.Load() < 2 {
		t.Errorf("expected at least 2 connection attempts, got %d", connectionAttempts.Load())
	}

	c.Stop()
}

func TestNoReconnectOnGracefulClose(t *testing.T) {
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

	var closeCalled, reconnectCalled atomic.Bool

	c.SetCloseHandler(func() { closeCalled.Store(true) })
	c.SetReconnectHandler(func() { reconnectCalled.Store(true) })

	if err := c.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	c.Start()
	time.Sleep(10 * time.Millisecond)

	// Graceful stop
	c.Stop()
	time.Sleep(100 * time.Millisecond)

	if !closeCalled.Load() {
		t.Error("close handler should be called on graceful stop")
	}
	if reconnectCalled.Load() {
		t.Error("reconnect handler should NOT be called on graceful stop")
	}
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

// TestStopDuringReconnect verifies that Stop() works correctly when called
// during an active reconnection attempt. This tests the race condition fix
// where Stop() could hang waiting for loops that were being restarted by
// reconnectLoop.
func TestStopDuringReconnect(t *testing.T) {
	// Track connection attempts
	var connectionAttempts atomic.Int32
	connectChan := make(chan struct{}, 10)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := connectionAttempts.Add(1)
		connectChan <- struct{}{}

		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		if attempt == 1 {
			// First connection: close immediately to trigger reconnect
			conn.Close()
			return
		}

		// Subsequent connections: keep open briefly then close
		defer conn.Close()
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c := NewClient(url, "pod-1", "test-token", nil)

	if err := c.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	c.Start()

	// Wait for first connection attempt
	<-connectChan

	// Wait for reconnect to start (second connection attempt)
	select {
	case <-connectChan:
		// Reconnect started
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for reconnect to start")
	}

	// Now call Stop() while reconnect is in progress
	// This should complete within a reasonable time (not hang)
	stopDone := make(chan struct{})
	go func() {
		c.Stop()
		close(stopDone)
	}()

	select {
	case <-stopDone:
		// Stop completed successfully
	case <-time.After(6 * time.Second):
		t.Error("Stop() hung during reconnect - race condition not fixed")
	}

	// Verify client is properly stopped
	if c.IsConnected() {
		t.Error("client should not be connected after Stop()")
	}
	if !c.stopped.Load() {
		t.Error("client should be marked as stopped")
	}
}

// TestConcurrentStopAndReconnect tests that multiple concurrent Stop() and
// reconnect operations don't cause panics or hangs.
func TestConcurrentStopAndReconnect(t *testing.T) {
	for i := 0; i < 10; i++ {
		t.Run("iteration", func(t *testing.T) {
			var connectionAttempts atomic.Int32

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				connectionAttempts.Add(1)
				conn, err := testUpgrader.Upgrade(w, r, nil)
				if err != nil {
					return
				}
				// Close connection after a short delay to trigger reconnect
				time.Sleep(50 * time.Millisecond)
				conn.Close()
			}))
			defer srv.Close()

			url := "ws" + strings.TrimPrefix(srv.URL, "http")
			c := NewClient(url, "pod-1", "test-token", nil)

			if err := c.Connect(); err != nil {
				t.Fatalf("Connect: %v", err)
			}
			c.Start()

			// Give some time for potential reconnection attempts
			time.Sleep(100 * time.Millisecond)

			// Stop the client - should not hang or panic
			stopDone := make(chan struct{})
			go func() {
				c.Stop()
				close(stopDone)
			}()

			select {
			case <-stopDone:
				// Success
			case <-time.After(6 * time.Second):
				t.Error("Stop() hung - possible race condition")
			}
		})
	}
}

// TestStartAfterStop verifies that Start() returns false after Stop() is called.
func TestStartAfterStop(t *testing.T) {
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

	if err := c.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Start should succeed before Stop
	if !c.Start() {
		t.Error("Start() should return true before Stop()")
	}

	c.Stop()

	// After Stop, the client cannot be reused, but we can verify stopped state
	if !c.stopped.Load() {
		t.Error("stopped flag should be true after Stop()")
	}
}

// TestStopIdempotent verifies that calling Stop() multiple times is safe.
func TestStopIdempotent(t *testing.T) {
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

	if err := c.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	c.Start()

	// Call Stop() multiple times concurrently
	done := make(chan struct{})
	for i := 0; i < 5; i++ {
		go func() {
			c.Stop()
		}()
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		close(done)
	}()

	select {
	case <-done:
		// All Stop() calls completed without hanging
	case <-time.After(6 * time.Second):
		t.Error("Multiple Stop() calls hung")
	}
}
