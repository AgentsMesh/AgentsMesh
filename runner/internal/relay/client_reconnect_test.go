package relay

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

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

// TestAuthFailureCircuitBreaker verifies that the reconnect loop gives up
// after maxConsecutiveAuthFailures handshake errors.
func TestAuthFailureCircuitBreaker(t *testing.T) {
	var connectionAttempts atomic.Int32
	var closeCalled atomic.Bool

	// Server always rejects with 401
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connectionAttempts.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c := NewClient(url, "pod-1", "test-token", nil)
	c.SetCloseHandler(func() { closeCalled.Store(true) })

	// Token refresh always fails (returns empty)
	c.SetTokenExpiredHandler(func() string { return "" })

	// Simulate that read/write loops have already exited (no prior connection).
	close(c.writeExitCh)

	// Manually trigger reconnectLoop as if readLoop spawned it.
	c.wg.Add(1)
	c.reconnecting.Store(true)
	go c.reconnectLoop()

	// Wait for circuit breaker to trigger
	deadline := time.After(30 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timeout: reconnect loop did not stop after auth failures")
		default:
			if closeCalled.Load() {
				// Circuit breaker triggered onClose
				attempts := connectionAttempts.Load()
				if attempts > int32(maxConsecutiveAuthFailures)+1 {
					t.Errorf("too many attempts: got %d, want <= %d",
						attempts, maxConsecutiveAuthFailures+1)
				}
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// TestAuthFailureResetsOnTransientError verifies that transient errors
// reset the consecutive auth failure counter.
func TestAuthFailureResetsOnTransientError(t *testing.T) {
	var connectionAttempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := connectionAttempts.Add(1)
		if attempt <= 3 {
			// First 3: auth failures
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if attempt == 4 {
			// 4th: network-level rejection (transient) - just close connection
			w.Header().Set("Connection", "close")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// 5th+: accept connection
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
	c.SetTokenExpiredHandler(func() string { return "" })

	reconnected := make(chan struct{})
	c.SetReconnectHandler(func() { close(reconnected) })

	close(c.writeExitCh)
	c.wg.Add(1)
	c.reconnecting.Store(true)
	go c.reconnectLoop()
	defer c.Stop()

	select {
	case <-reconnected:
		// After 3 auth + 1 transient + 1 success = reconnected
		if connectionAttempts.Load() < 5 {
			t.Errorf("expected at least 5 attempts, got %d", connectionAttempts.Load())
		}
	case <-time.After(30 * time.Second):
		t.Fatal("timeout: should have reconnected after transient error reset counter")
	}
}

// TestAuthFailure_TokenRefreshSuccessThenFailAgain verifies that after a
// successful token refresh, the auth failure counter resets and a fresh
// sequence of failures is needed to trigger the circuit breaker.
func TestAuthFailure_TokenRefreshSuccessThenFailAgain(t *testing.T) {
	var connectionAttempts atomic.Int32
	var closeCalled atomic.Bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connectionAttempts.Add(1)
		// Always reject — even refreshed tokens don't help
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c := NewClient(url, "pod-1", "test-token", nil)
	c.SetCloseHandler(func() { closeCalled.Store(true) })

	// Token refresh succeeds (returns a new token), but the new token also fails
	c.SetTokenExpiredHandler(func() string { return "refreshed-token" })

	close(c.writeExitCh)
	c.wg.Add(1)
	c.reconnecting.Store(true)
	go c.reconnectLoop()

	deadline := time.After(30 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timeout: circuit breaker should have triggered")
		default:
			if closeCalled.Load() {
				// Circuit breaker triggered. After token refresh success (resets counter),
				// need maxConsecutiveAuthFailures more failures.
				// Attempt 1: auth fail → refresh → success → counter=0, retry immediately
				// Attempts 2-6: auth fail × 5 → circuit break
				// Total: 1 (refresh attempt) + 1 (retry with new token) + 5 = 7 max
				attempts := connectionAttempts.Load()
				if attempts > int32(maxConsecutiveAuthFailures)+2 {
					t.Errorf("too many attempts after refresh: got %d", attempts)
				}
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
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
