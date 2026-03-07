package server

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
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
			Capacity:    1000,
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
	l.Close()
	return port
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
			Host:         "127.0.0.1",
			Port:         port,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Issuer: "test-issuer",
		},
		Backend: config.BackendConfig{
			URL:               mockBackend.URL,
			InternalAPISecret: "test-internal",
			HeartbeatInterval: 50 * time.Millisecond,
		},
		Session: config.SessionConfig{
			KeepAliveDuration: 5 * time.Second,
			MaxBrowsersPerPod: 10,
		},
		Relay: config.RelayConfig{
			ID:       "relay-test",
			URL:      fmt.Sprintf("ws://127.0.0.1:%d", port),
			Region:   "test",
			Capacity: 100,
		},
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
		Server: config.ServerConfig{
			Host:         "127.0.0.1",
			Port:         port,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Issuer: "test-issuer",
		},
		Backend: config.BackendConfig{
			URL:               mockBackend.URL,
			InternalAPISecret: "test-internal",
			HeartbeatInterval: 50 * time.Millisecond,
		},
		Session: config.SessionConfig{
			KeepAliveDuration: 5 * time.Second,
			MaxBrowsersPerPod: 10,
		},
		Relay: config.RelayConfig{
			ID:       "relay-test",
			URL:      fmt.Sprintf("ws://127.0.0.1:%d", port),
			Region:   "test",
			Capacity: 100,
		},
	}

	s := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start(ctx)
	}()

	// Wait for server to be ready
	deadline := time.Now().Add(3 * time.Second)
	var healthOK bool
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
		if err == nil {
			resp.Body.Close()
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

	// Verify register was called
	if registerCalled.Load() < 1 {
		t.Error("register should have been called")
	}

	// Make stats request
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/stats", port))
	if err != nil {
		t.Fatalf("stats request failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), "active_channels") {
		t.Errorf("unexpected stats body: %s", body)
	}

	// Wait for at least one heartbeat
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

	// Cancel context to trigger graceful shutdown
	cancel()

	// Wait for Start to return
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Start returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}

	// Verify unregister was called during shutdown
	if unregisterCalled.Load() < 1 {
		t.Error("unregister should have been called during graceful shutdown")
	}

	// Verify server is no longer accepting connections
	if s.IsAcceptingConnections() {
		t.Error("should not be accepting connections after shutdown")
	}
}

func TestServer_StartAndShutdown_WithTLS(t *testing.T) {
	// Test the TLS code path in Start() — server creates TLSConfig with GetCertificate callback
	// The actual TLS handshake will fail because we don't have real certs, but the
	// code path for setting up TLSConfig and calling ListenAndServeTLS is exercised.

	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockBackend.Close()

	port := findFreePort(t)
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "127.0.0.1",
			Port:         port,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			TLS: config.TLSConfig{
				Enabled: true,
			},
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Issuer: "test-issuer",
		},
		Backend: config.BackendConfig{
			URL:               mockBackend.URL,
			InternalAPISecret: "test-internal",
			HeartbeatInterval: 1 * time.Second,
		},
		Session: config.SessionConfig{
			KeepAliveDuration: 5 * time.Second,
			MaxBrowsersPerPod: 10,
		},
		Relay: config.RelayConfig{
			ID:       "relay-tls-test",
			URL:      fmt.Sprintf("wss://127.0.0.1:%d", port),
			Region:   "test",
			Capacity: 100,
		},
	}

	s := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start(ctx)
	}()

	// Give server time to start TLS listener
	time.Sleep(200 * time.Millisecond)

	// Cancel context to trigger graceful shutdown
	cancel()

	select {
	case err := <-errCh:
		// Either nil (graceful shutdown) or TLS error — both are acceptable
		// The point is that the TLS code path was exercised
		_ = err
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}

func TestServer_New_OnAllSubscribersGoneCallback(t *testing.T) {
	// Test that the onAllSubscribersGone closure in New() correctly calls backendClient.NotifySessionClosed
	notifyCalled := make(chan string, 1)
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/internal/relays/session-closed" {
			var req struct {
				PodKey string `json:"pod_key"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			notifyCalled <- req.PodKey
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mockBackend.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "127.0.0.1",
			Port:         8090,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Issuer: "test-issuer",
		},
		Backend: config.BackendConfig{
			URL:               mockBackend.URL,
			InternalAPISecret: "test-internal",
			HeartbeatInterval: 10 * time.Second,
		},
		Session: config.SessionConfig{
			KeepAliveDuration:        50 * time.Millisecond, // Very short for testing
			MaxBrowsersPerPod:        10,
			RunnerReconnectTimeout:   200 * time.Millisecond,
			BrowserReconnectTimeout:  200 * time.Millisecond,
			PendingConnectionTimeout: 500 * time.Millisecond,
			OutputBufferSize:         1024,
			OutputBufferCount:        5,
		},
		Relay: config.RelayConfig{
			ID:       "relay-1",
			URL:      "ws://localhost:8090",
			Region:   "test",
			Capacity: 100,
		},
	}

	s := New(cfg)

	// Use the channelManager directly to avoid WebSocket complexity
	// Create a WS pair for publisher
	pubUpgrader := createTestWSPair(t)
	subUpgrader := createTestWSPair(t)

	if err := s.channelManager.HandlePublisherConnect("test-pod", pubUpgrader.serverConn); err != nil {
		t.Fatalf("HandlePublisherConnect: %v", err)
	}
	if err := s.channelManager.HandleSubscriberConnect("test-pod", "sub-1", subUpgrader.serverConn); err != nil {
		t.Fatalf("HandleSubscriberConnect: %v", err)
	}

	// Verify channel exists
	if s.channelManager.GetChannel("test-pod") == nil {
		t.Fatal("expected channel to exist")
	}

	// Close the subscriber client connection to trigger the forwardSubscriberToPublisher
	// goroutine to call RemoveSubscriber, which starts the keep-alive timer → onAllSubscribersGone
	subUpgrader.clientConn.Close()

	// Wait for the onAllSubscribersGone callback
	select {
	case podKey := <-notifyCalled:
		if podKey != "test-pod" {
			t.Errorf("NotifySessionClosed podKey: got %q, want %q", podKey, "test-pod")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("onAllSubscribersGone callback was not triggered within timeout")
	}
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
		clientConn.Close()
		serverConn.Close()
	})

	return &testWSPair{serverConn: serverConn, clientConn: clientConn}
}

func TestServer_GracefulShutdown_WithActiveChannels(t *testing.T) {
	// Test that gracefulShutdown waits for active channels to close
	var unregisterCalled atomic.Int32

	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/internal/relays/unregister":
			unregisterCalled.Add(1)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mockBackend.Close()

	port := findFreePort(t)
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "127.0.0.1",
			Port:         port,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Issuer: "test-issuer",
		},
		Backend: config.BackendConfig{
			URL:               mockBackend.URL,
			InternalAPISecret: "test-internal",
			HeartbeatInterval: 1 * time.Second,
		},
		Session: config.SessionConfig{
			KeepAliveDuration:        5 * time.Second,
			MaxBrowsersPerPod:        10,
			RunnerReconnectTimeout:   200 * time.Millisecond,
			BrowserReconnectTimeout:  200 * time.Millisecond,
			PendingConnectionTimeout: 500 * time.Millisecond,
			OutputBufferSize:         1024,
			OutputBufferCount:        5,
		},
		Relay: config.RelayConfig{
			ID:       "relay-test",
			URL:      fmt.Sprintf("ws://127.0.0.1:%d", port),
			Region:   "test",
			Capacity: 100,
		},
	}

	s := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start(ctx)
	}()

	// Wait for server to be ready
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Create an active channel
	pubPair := createTestWSPair(t)
	subPair := createTestWSPair(t)
	if err := s.channelManager.HandlePublisherConnect("shutdown-pod", pubPair.serverConn); err != nil {
		t.Fatalf("HandlePublisherConnect: %v", err)
	}
	if err := s.channelManager.HandleSubscriberConnect("shutdown-pod", "sub-1", subPair.serverConn); err != nil {
		t.Fatalf("HandleSubscriberConnect: %v", err)
	}

	stats := s.Stats()
	if stats.ActiveChannels != 1 {
		t.Fatalf("expected 1 active channel, got %d", stats.ActiveChannels)
	}

	// Close the channel so graceful shutdown won't wait the full 30s
	s.channelManager.CloseChannel("shutdown-pod")

	// Cancel context to trigger graceful shutdown
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
		t.Error("unregister should have been called")
	}
}

func TestServer_Start_PortInUse(t *testing.T) {
	// Test the errCh path in Start(): server fails to bind port → returns error
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockBackend.Close()

	// Bind a port so the server can't use it
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	defer l.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "127.0.0.1",
			Port:         port, // already in use
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Issuer: "test-issuer",
		},
		Backend: config.BackendConfig{
			URL:               mockBackend.URL,
			InternalAPISecret: "test-internal",
			HeartbeatInterval: 10 * time.Second,
		},
		Session: config.SessionConfig{
			KeepAliveDuration: 5 * time.Second,
			MaxBrowsersPerPod: 10,
		},
		Relay: config.RelayConfig{
			ID:       "relay-test",
			URL:      fmt.Sprintf("ws://127.0.0.1:%d", port),
			Region:   "test",
			Capacity: 100,
		},
	}

	s := New(cfg)

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start(context.Background())
	}()

	select {
	case err := <-errCh:
		if err == nil {
			t.Error("expected error when port is in use")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after port bind failure")
	}
}

func TestServer_GracefulShutdown_WaitsForChannels(t *testing.T) {
	// Test the gracefulShutdown wait loop when active channels exist during shutdown
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockBackend.Close()

	port := findFreePort(t)
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "127.0.0.1",
			Port:         port,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Issuer: "test-issuer",
		},
		Backend: config.BackendConfig{
			URL:               mockBackend.URL,
			InternalAPISecret: "test-internal",
			HeartbeatInterval: 10 * time.Second,
		},
		Session: config.SessionConfig{
			KeepAliveDuration:        5 * time.Second,
			MaxBrowsersPerPod:        10,
			RunnerReconnectTimeout:   5 * time.Second,
			BrowserReconnectTimeout:  5 * time.Second,
			PendingConnectionTimeout: 5 * time.Second,
			OutputBufferSize:         1024,
			OutputBufferCount:        5,
		},
		Relay: config.RelayConfig{
			ID:       "relay-test",
			URL:      fmt.Sprintf("ws://127.0.0.1:%d", port),
			Region:   "test",
			Capacity: 100,
		},
	}

	s := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start(ctx)
	}()

	// Wait for server to be ready
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Create an active channel that persists during shutdown
	pubPair := createTestWSPair(t)
	subPair := createTestWSPair(t)
	if err := s.channelManager.HandlePublisherConnect("wait-pod", pubPair.serverConn); err != nil {
		t.Fatalf("HandlePublisherConnect: %v", err)
	}
	if err := s.channelManager.HandleSubscriberConnect("wait-pod", "sub-1", subPair.serverConn); err != nil {
		t.Fatalf("HandleSubscriberConnect: %v", err)
	}

	// Cancel context to trigger graceful shutdown — channel is still active
	cancel()

	// Close the channel after a short delay to exercise the wait loop body
	go func() {
		time.Sleep(1200 * time.Millisecond)
		s.channelManager.CloseChannel("wait-pod")
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Start returned error: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}

	if s.IsAcceptingConnections() {
		t.Error("should not be accepting connections after shutdown")
	}
}

func TestServer_New_OnAllSubscribersGone_NotifyFails(t *testing.T) {
	// Test the error path in the onAllSubscribersGone closure when NotifySessionClosed fails
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/internal/relays/session-closed" {
			w.WriteHeader(http.StatusInternalServerError) // Notification fails
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mockBackend.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "127.0.0.1",
			Port:         8090,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Issuer: "test-issuer",
		},
		Backend: config.BackendConfig{
			URL:               mockBackend.URL,
			InternalAPISecret: "test-internal",
			HeartbeatInterval: 10 * time.Second,
		},
		Session: config.SessionConfig{
			KeepAliveDuration:        50 * time.Millisecond,
			MaxBrowsersPerPod:        10,
			RunnerReconnectTimeout:   200 * time.Millisecond,
			BrowserReconnectTimeout:  200 * time.Millisecond,
			PendingConnectionTimeout: 500 * time.Millisecond,
			OutputBufferSize:         1024,
			OutputBufferCount:        5,
		},
		Relay: config.RelayConfig{
			ID:       "relay-1",
			URL:      "ws://localhost:8090",
			Region:   "test",
			Capacity: 100,
		},
	}

	s := New(cfg)

	pubPair := createTestWSPair(t)
	subPair := createTestWSPair(t)

	if err := s.channelManager.HandlePublisherConnect("fail-pod", pubPair.serverConn); err != nil {
		t.Fatalf("HandlePublisherConnect: %v", err)
	}
	if err := s.channelManager.HandleSubscriberConnect("fail-pod", "sub-1", subPair.serverConn); err != nil {
		t.Fatalf("HandleSubscriberConnect: %v", err)
	}

	// Close the subscriber client to trigger removal → keep-alive → onAllSubscribersGone → NotifySessionClosed error
	subPair.clientConn.Close()

	// Wait for the callback to fire and the error to be logged (not crash)
	time.Sleep(500 * time.Millisecond)

	// If we reach here without panic, the error path was handled gracefully
}

func TestServer_Start_ContextCancellation(t *testing.T) {
	// Test the context cancellation path in Start() — signal handling is now in main.go
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockBackend.Close()

	port := findFreePort(t)
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "127.0.0.1",
			Port:         port,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Issuer: "test-issuer",
		},
		Backend: config.BackendConfig{
			URL:               mockBackend.URL,
			InternalAPISecret: "test-internal",
			HeartbeatInterval: 10 * time.Second,
		},
		Session: config.SessionConfig{
			KeepAliveDuration: 5 * time.Second,
			MaxBrowsersPerPod: 10,
		},
		Relay: config.RelayConfig{
			ID:       "relay-ctx-test",
			URL:      fmt.Sprintf("ws://127.0.0.1:%d", port),
			Region:   "test",
			Capacity: 100,
		},
	}

	s := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start(ctx)
	}()

	// Wait for server to be ready
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Cancel context to trigger shutdown (simulating main.go signal handler calling cancel())
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Start returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}

func TestServer_Start_TLS_GetCertificate_NoCert(t *testing.T) {
	// Test GetCertificate callback when no certificate is available
	// Exercises the "no TLS certificate available" return path
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockBackend.Close()

	port := findFreePort(t)
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "127.0.0.1",
			Port:         port,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			TLS: config.TLSConfig{
				Enabled: true,
				// No CertFile/KeyFile → "no TLS certificate available"
			},
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Issuer: "test-issuer",
		},
		Backend: config.BackendConfig{
			URL:               mockBackend.URL,
			InternalAPISecret: "test-internal",
			HeartbeatInterval: 10 * time.Second,
		},
		Session: config.SessionConfig{
			KeepAliveDuration: 5 * time.Second,
			MaxBrowsersPerPod: 10,
		},
		Relay: config.RelayConfig{
			ID:       "relay-tls-nocert",
			URL:      fmt.Sprintf("wss://127.0.0.1:%d", port),
			Region:   "test",
			Capacity: 100,
		},
	}

	s := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start(ctx)
	}()

	// Give server time to start TLS listener
	time.Sleep(200 * time.Millisecond)

	// Connect with TLS to trigger GetCertificate callback
	// The handshake will fail (no cert available), but the callback code is exercised
	tlsConn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 1 * time.Second},
		"tcp",
		fmt.Sprintf("127.0.0.1:%d", port),
		&tls.Config{InsecureSkipVerify: true},
	)
	if err == nil {
		tlsConn.Close()
		// It's OK if the connection fails — the point is to exercise GetCertificate
	}
	// The error is expected (no certificate available)

	cancel()

	select {
	case <-errCh:
		// Either nil or TLS error — both acceptable
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}

func TestServer_Start_TLS_GetCertificate_WithBackendCert(t *testing.T) {
	// Generate a self-signed certificate for testing
	cert, key := generateSelfSignedCert(t)

	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/internal/relays/register" {
			// Return TLS cert in register response
			w.Header().Set("Content-Type", "application/json")
			resp := struct {
				Status    string `json:"status"`
				TLSCert   string `json:"tls_cert"`
				TLSKey    string `json:"tls_key"`
				TLSExpiry string `json:"tls_expiry"`
			}{
				Status:    "ok",
				TLSCert:   cert,
				TLSKey:    key,
				TLSExpiry: "2027-01-01T00:00:00Z",
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mockBackend.Close()

	port := findFreePort(t)
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "127.0.0.1",
			Port:         port,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			TLS: config.TLSConfig{
				Enabled: true,
			},
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Issuer: "test-issuer",
		},
		Backend: config.BackendConfig{
			URL:               mockBackend.URL,
			InternalAPISecret: "test-internal",
			HeartbeatInterval: 10 * time.Second,
		},
		Session: config.SessionConfig{
			KeepAliveDuration: 5 * time.Second,
			MaxBrowsersPerPod: 10,
		},
		Relay: config.RelayConfig{
			ID:       "relay-tls-cert",
			URL:      fmt.Sprintf("wss://127.0.0.1:%d", port),
			Region:   "test",
			Capacity: 100,
		},
	}

	s := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start(ctx)
	}()

	// Wait for TLS server to be ready
	time.Sleep(300 * time.Millisecond)

	// Connect with TLS to trigger GetCertificate → HasTLSCertificate → load from backend
	tlsConn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 2 * time.Second},
		"tcp",
		fmt.Sprintf("127.0.0.1:%d", port),
		&tls.Config{InsecureSkipVerify: true},
	)
	if err != nil {
		// TLS handshake might fail if cert doesn't match hostname, but callback was exercised
		t.Logf("TLS dial error (expected): %v", err)
	} else {
		tlsConn.Close()
	}

	cancel()

	select {
	case <-errCh:
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}

func TestServer_Start_TLS_GetCertificate_WithCertFiles(t *testing.T) {
	// Generate a self-signed certificate and save to files
	certPEM, keyPEM := generateSelfSignedCert(t)

	dir := t.TempDir()
	certFile := dir + "/cert.pem"
	keyFile := dir + "/key.pem"
	_ = os.WriteFile(certFile, []byte(certPEM), 0644)
	_ = os.WriteFile(keyFile, []byte(keyPEM), 0600)

	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockBackend.Close()

	port := findFreePort(t)
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "127.0.0.1",
			Port:         port,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			TLS: config.TLSConfig{
				Enabled:  true,
				CertFile: certFile,
				KeyFile:  keyFile,
			},
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Issuer: "test-issuer",
		},
		Backend: config.BackendConfig{
			URL:               mockBackend.URL,
			InternalAPISecret: "test-internal",
			HeartbeatInterval: 10 * time.Second,
		},
		Session: config.SessionConfig{
			KeepAliveDuration: 5 * time.Second,
			MaxBrowsersPerPod: 10,
		},
		Relay: config.RelayConfig{
			ID:       "relay-tls-files",
			URL:      fmt.Sprintf("wss://127.0.0.1:%d", port),
			Region:   "test",
			Capacity: 100,
		},
	}

	s := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start(ctx)
	}()

	time.Sleep(300 * time.Millisecond)

	// Connect with TLS → GetCertificate → HasTLSCertificate=false → fallback to cert files
	tlsConn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 2 * time.Second},
		"tcp",
		fmt.Sprintf("127.0.0.1:%d", port),
		&tls.Config{InsecureSkipVerify: true},
	)
	if err != nil {
		t.Logf("TLS dial error (may be expected): %v", err)
	} else {
		// TLS handshake succeeded with cert files
		tlsConn.Close()
	}

	cancel()

	select {
	case <-errCh:
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}

func TestServer_Start_TLS_GetCertificate_InvalidBackendCert(t *testing.T) {
	// Backend returns invalid cert data → error branch in GetCertificate
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/internal/relays/register" {
			w.Header().Set("Content-Type", "application/json")
			resp := struct {
				Status    string `json:"status"`
				TLSCert   string `json:"tls_cert"`
				TLSKey    string `json:"tls_key"`
				TLSExpiry string `json:"tls_expiry"`
			}{
				Status:    "ok",
				TLSCert:   "INVALID_CERT_PEM",
				TLSKey:    "INVALID_KEY_PEM",
				TLSExpiry: "2027-01-01T00:00:00Z",
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mockBackend.Close()

	port := findFreePort(t)
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "127.0.0.1",
			Port:         port,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			TLS: config.TLSConfig{
				Enabled: true,
			},
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Issuer: "test-issuer",
		},
		Backend: config.BackendConfig{
			URL:               mockBackend.URL,
			InternalAPISecret: "test-internal",
			HeartbeatInterval: 10 * time.Second,
		},
		Session: config.SessionConfig{
			KeepAliveDuration: 5 * time.Second,
			MaxBrowsersPerPod: 10,
		},
		Relay: config.RelayConfig{
			ID:       "relay-tls-invalid",
			URL:      fmt.Sprintf("wss://127.0.0.1:%d", port),
			Region:   "test",
			Capacity: 100,
		},
	}

	s := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start(ctx)
	}()

	time.Sleep(300 * time.Millisecond)

	// Connect with TLS → GetCertificate → HasTLSCertificate=true → X509KeyPair fails → error logged
	// Then falls through to cert files check (empty) → "no TLS certificate available"
	tlsConn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 1 * time.Second},
		"tcp",
		fmt.Sprintf("127.0.0.1:%d", port),
		&tls.Config{InsecureSkipVerify: true},
	)
	if err == nil {
		tlsConn.Close()
	}

	cancel()

	select {
	case <-errCh:
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}

func TestServer_Start_TLS_GetCertificate_InvalidCertFiles(t *testing.T) {
	// Cert files exist but contain invalid data → LoadX509KeyPair error
	dir := t.TempDir()
	certFile := dir + "/cert.pem"
	keyFile := dir + "/key.pem"
	_ = os.WriteFile(certFile, []byte("INVALID"), 0644)
	_ = os.WriteFile(keyFile, []byte("INVALID"), 0600)

	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockBackend.Close()

	port := findFreePort(t)
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "127.0.0.1",
			Port:         port,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			TLS: config.TLSConfig{
				Enabled:  true,
				CertFile: certFile,
				KeyFile:  keyFile,
			},
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Issuer: "test-issuer",
		},
		Backend: config.BackendConfig{
			URL:               mockBackend.URL,
			InternalAPISecret: "test-internal",
			HeartbeatInterval: 10 * time.Second,
		},
		Session: config.SessionConfig{
			KeepAliveDuration: 5 * time.Second,
			MaxBrowsersPerPod: 10,
		},
		Relay: config.RelayConfig{
			ID:       "relay-tls-bad-files",
			URL:      fmt.Sprintf("wss://127.0.0.1:%d", port),
			Region:   "test",
			Capacity: 100,
		},
	}

	s := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start(ctx)
	}()

	time.Sleep(300 * time.Millisecond)

	// Connect with TLS → GetCertificate → no backend cert → LoadX509KeyPair fails → error returned
	tlsConn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 1 * time.Second},
		"tcp",
		fmt.Sprintf("127.0.0.1:%d", port),
		&tls.Config{InsecureSkipVerify: true},
	)
	if err == nil {
		tlsConn.Close()
	}

	cancel()

	select {
	case <-errCh:
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}

func TestServer_GracefulShutdown_UnregisterFails(t *testing.T) {
	// Test gracefulShutdown when unregister returns error → warning logged
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/internal/relays/unregister" {
			w.WriteHeader(http.StatusInternalServerError) // Unregister fails
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mockBackend.Close()

	port := findFreePort(t)
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "127.0.0.1",
			Port:         port,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Issuer: "test-issuer",
		},
		Backend: config.BackendConfig{
			URL:               mockBackend.URL,
			InternalAPISecret: "test-internal",
			HeartbeatInterval: 10 * time.Second,
		},
		Session: config.SessionConfig{
			KeepAliveDuration: 5 * time.Second,
			MaxBrowsersPerPod: 10,
		},
		Relay: config.RelayConfig{
			ID:       "relay-unreg-fail",
			URL:      fmt.Sprintf("ws://127.0.0.1:%d", port),
			Region:   "test",
			Capacity: 100,
		},
	}

	s := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start(ctx)
	}()

	// Wait for server to be ready
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}

	cancel()

	select {
	case err := <-errCh:
		// Should still complete without error (unregister failure is logged, not returned)
		if err != nil {
			t.Errorf("Start returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}

// generateSelfSignedCert generates a self-signed TLS certificate for testing
func generateSelfSignedCert(t *testing.T) (certPEM, keyPEM string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test"},
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(24 * time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}

	certBuf := &bytes.Buffer{}
	_ = pem.Encode(certBuf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}

	keyBuf := &bytes.Buffer{}
	_ = pem.Encode(keyBuf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return certBuf.String(), keyBuf.String()
}
