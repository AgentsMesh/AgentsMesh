package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/anthropics/agentsmesh/relay/internal/config"
)

func testConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Host:         "0.0.0.0",
			Port:         8090,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Issuer: "test-issuer",
		},
		Backend: config.BackendConfig{
			URL:               "http://localhost:8080",
			InternalAPISecret: "internal-secret",
			HeartbeatInterval: 10 * time.Second,
		},
		Session: config.SessionConfig{
			KeepAliveDuration: 30 * time.Second,
			MaxBrowsersPerPod: 10,
		},
		Relay: config.RelayConfig{
			ID:       "relay-1",
			URL:      "ws://localhost:8090",
			Region:   "us-west",
			Capacity: 1000,
		},
	}
}

// findFreePort returns a free TCP port on localhost
func findFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("findFreePort: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port
}

// testWSPair holds a WebSocket connection pair for testing
type testWSPair struct {
	serverConn *websocket.Conn
	clientConn *websocket.Conn
}

// createTestWSPair creates a WebSocket pair for testing in the server package
func createTestWSPair(t *testing.T) *testWSPair {
	t.Helper()
	var serverConn *websocket.Conn
	var wg sync.WaitGroup
	wg.Add(1)

	wsUpgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("ws upgrade: %v", err)
		}
		serverConn = c
		wg.Done()
	}))

	wsURL := "ws" + srv.URL[4:]
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		srv.Close()
		t.Fatalf("ws dial: %v", err)
	}

	wg.Wait()
	srv.Close()

	t.Cleanup(func() {
		_ = clientConn.Close()
		_ = serverConn.Close()
	})

	return &testWSPair{serverConn: serverConn, clientConn: clientConn}
}

func TestNew(t *testing.T) {
	cfg := testConfig()
	server := New(cfg)

	if server == nil {
		t.Fatal("New returned nil")
	}
	if server.cfg != cfg {
		t.Error("cfg not set correctly")
	}
	if server.channelManager == nil {
		t.Error("channelManager should not be nil")
	}
	if server.backendClient == nil {
		t.Error("backendClient should not be nil")
	}
	if server.handler == nil {
		t.Error("handler should not be nil")
	}
	if server.logger == nil {
		t.Error("logger should not be nil")
	}
}

func TestServer_Stats(t *testing.T) {
	server := New(testConfig())
	stats := server.Stats()

	if stats.ActiveChannels != 0 {
		t.Errorf("ActiveChannels: expected 0, got %d", stats.ActiveChannels)
	}
	if stats.TotalSubscribers != 0 {
		t.Errorf("TotalSubscribers: expected 0, got %d", stats.TotalSubscribers)
	}
	if stats.PendingPublishers != 0 {
		t.Errorf("PendingPublishers: expected 0, got %d", stats.PendingPublishers)
	}
	if stats.PendingSubscribers != 0 {
		t.Errorf("PendingSubscribers: expected 0, got %d", stats.PendingSubscribers)
	}
}

func TestServer_IsAcceptingConnections(t *testing.T) {
	server := New(testConfig())
	if !server.IsAcceptingConnections() {
		t.Error("new server should accept connections")
	}
}

func TestServer_Start_RegisterFails(t *testing.T) {
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockBackend.Close()

	port := findFreePort(t)
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "127.0.0.1", Port: port,
			ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second,
		},
		JWT:     config.JWTConfig{Secret: "test-secret", Issuer: "test-issuer"},
		Backend: config.BackendConfig{URL: mockBackend.URL, InternalAPISecret: "test-internal", HeartbeatInterval: 50 * time.Millisecond},
		Session: config.SessionConfig{KeepAliveDuration: 5 * time.Second, MaxBrowsersPerPod: 10},
		Relay:   config.RelayConfig{ID: "relay-test", URL: fmt.Sprintf("ws://127.0.0.1:%d", port), Region: "test", Capacity: 100},
	}

	s := New(cfg)
	err := s.Start(context.Background())
	if err == nil {
		t.Fatal("expected error when register fails")
	}
	if !strings.Contains(err.Error(), "failed to register") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestServer_StartAndShutdown(t *testing.T) {
	var registerCalled atomic.Int32
	var heartbeatCount atomic.Int32
	var unregisterCalled atomic.Int32

	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/internal/relays/register":
			registerCalled.Add(1)
			w.WriteHeader(http.StatusOK)
		case "/api/internal/relays/heartbeat":
			heartbeatCount.Add(1)
			w.WriteHeader(http.StatusOK)
		case "/api/internal/relays/unregister":
			unregisterCalled.Add(1)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockBackend.Close()

	port := findFreePort(t)
	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "127.0.0.1", Port: port, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second},
		JWT:     config.JWTConfig{Secret: "test-secret", Issuer: "test-issuer"},
		Backend: config.BackendConfig{URL: mockBackend.URL, InternalAPISecret: "test-internal", HeartbeatInterval: 50 * time.Millisecond},
		Session: config.SessionConfig{KeepAliveDuration: 5 * time.Second, MaxBrowsersPerPod: 10},
		Relay:   config.RelayConfig{ID: "relay-test", URL: fmt.Sprintf("ws://127.0.0.1:%d", port), Region: "test", Capacity: 100},
	}

	s := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- s.Start(ctx) }()

	// Wait for server to be ready
	deadline := time.Now().Add(3 * time.Second)
	var healthOK bool
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				healthOK = true
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !healthOK {
		cancel()
		t.Fatal("server did not become ready within timeout")
	}

	if registerCalled.Load() < 1 {
		t.Error("register should have been called")
	}

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/stats", port))
	if err != nil {
		t.Fatalf("stats request failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if !strings.Contains(string(body), "active_channels") {
		t.Errorf("unexpected stats body: %s", body)
	}

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if heartbeatCount.Load() >= 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if heartbeatCount.Load() < 1 {
		t.Error("no heartbeat received")
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Start returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}

	if unregisterCalled.Load() < 1 {
		t.Error("unregister should have been called during graceful shutdown")
	}
	if s.IsAcceptingConnections() {
		t.Error("should not be accepting connections after shutdown")
	}
}

func TestServer_Start_PortInUse(t *testing.T) {
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockBackend.Close()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	defer func() { _ = l.Close() }()

	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "127.0.0.1", Port: port, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second},
		JWT:     config.JWTConfig{Secret: "test-secret", Issuer: "test-issuer"},
		Backend: config.BackendConfig{URL: mockBackend.URL, InternalAPISecret: "test-internal", HeartbeatInterval: 10 * time.Second},
		Session: config.SessionConfig{KeepAliveDuration: 5 * time.Second, MaxBrowsersPerPod: 10},
		Relay:   config.RelayConfig{ID: "relay-test", URL: fmt.Sprintf("ws://127.0.0.1:%d", port), Region: "test", Capacity: 100},
	}

	s := New(cfg)

	errCh := make(chan error, 1)
	go func() { errCh <- s.Start(context.Background()) }()

	select {
	case err := <-errCh:
		if err == nil {
			t.Error("expected error when port is in use")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after port bind failure")
	}
}
