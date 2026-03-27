package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/relay/internal/config"
)

func TestServer_StartAndShutdown_WithTLS(t *testing.T) {
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockBackend.Close()

	port := findFreePort(t)
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "127.0.0.1", Port: port,
			ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second,
			TLS: config.TLSConfig{Enabled: true},
		},
		JWT:     config.JWTConfig{Secret: "test-secret", Issuer: "test-issuer"},
		Backend: config.BackendConfig{URL: mockBackend.URL, InternalAPISecret: "test-internal", HeartbeatInterval: 1 * time.Second},
		Session: config.SessionConfig{KeepAliveDuration: 5 * time.Second, MaxBrowsersPerPod: 10},
		Relay:   config.RelayConfig{ID: "relay-tls-test", URL: fmt.Sprintf("wss://127.0.0.1:%d", port), Region: "test", Capacity: 100},
	}

	s := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- s.Start(ctx) }()

	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case <-errCh:
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}

func TestServer_GracefulShutdown_WithActiveChannels(t *testing.T) {
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
		Server: config.ServerConfig{Host: "127.0.0.1", Port: port, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second},
		JWT:    config.JWTConfig{Secret: "test-secret", Issuer: "test-issuer"},
		Backend: config.BackendConfig{URL: mockBackend.URL, InternalAPISecret: "test-internal", HeartbeatInterval: 1 * time.Second},
		Session: config.SessionConfig{
			KeepAliveDuration: 5 * time.Second, MaxBrowsersPerPod: 10,
			RunnerReconnectTimeout: 200 * time.Millisecond, BrowserReconnectTimeout: 200 * time.Millisecond,
			PendingConnectionTimeout: 500 * time.Millisecond, OutputBufferSize: 1024, OutputBufferCount: 5,
		},
		Relay: config.RelayConfig{ID: "relay-test", URL: fmt.Sprintf("ws://127.0.0.1:%d", port), Region: "test", Capacity: 100},
	}

	s := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- s.Start(ctx) }()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}

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

	s.channelManager.CloseChannel("shutdown-pod")
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

func TestServer_GracefulShutdown_WaitsForChannels(t *testing.T) {
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockBackend.Close()

	port := findFreePort(t)
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "127.0.0.1", Port: port, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second},
		JWT:    config.JWTConfig{Secret: "test-secret", Issuer: "test-issuer"},
		Backend: config.BackendConfig{URL: mockBackend.URL, InternalAPISecret: "test-internal", HeartbeatInterval: 10 * time.Second},
		Session: config.SessionConfig{
			KeepAliveDuration: 5 * time.Second, MaxBrowsersPerPod: 10,
			RunnerReconnectTimeout: 5 * time.Second, BrowserReconnectTimeout: 5 * time.Second,
			PendingConnectionTimeout: 5 * time.Second, OutputBufferSize: 1024, OutputBufferCount: 5,
		},
		Relay: config.RelayConfig{ID: "relay-test", URL: fmt.Sprintf("ws://127.0.0.1:%d", port), Region: "test", Capacity: 100},
	}

	s := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- s.Start(ctx) }()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}

	pubPair := createTestWSPair(t)
	subPair := createTestWSPair(t)
	if err := s.channelManager.HandlePublisherConnect("wait-pod", pubPair.serverConn); err != nil {
		t.Fatalf("HandlePublisherConnect: %v", err)
	}
	if err := s.channelManager.HandleSubscriberConnect("wait-pod", "sub-1", subPair.serverConn); err != nil {
		t.Fatalf("HandleSubscriberConnect: %v", err)
	}

	cancel()

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

func TestServer_GracefulShutdown_UnregisterFails(t *testing.T) {
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/internal/relays/unregister" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mockBackend.Close()

	port := findFreePort(t)
	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "127.0.0.1", Port: port, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second},
		JWT:     config.JWTConfig{Secret: "test-secret", Issuer: "test-issuer"},
		Backend: config.BackendConfig{URL: mockBackend.URL, InternalAPISecret: "test-internal", HeartbeatInterval: 10 * time.Second},
		Session: config.SessionConfig{KeepAliveDuration: 5 * time.Second, MaxBrowsersPerPod: 10},
		Relay:   config.RelayConfig{ID: "relay-unreg-fail", URL: fmt.Sprintf("ws://127.0.0.1:%d", port), Region: "test", Capacity: 100},
	}

	s := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- s.Start(ctx) }()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
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
}

func TestServer_New_OnAllSubscribersGoneCallback(t *testing.T) {
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
		Server: config.ServerConfig{Host: "127.0.0.1", Port: 8090, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second},
		JWT:    config.JWTConfig{Secret: "test-secret", Issuer: "test-issuer"},
		Backend: config.BackendConfig{URL: mockBackend.URL, InternalAPISecret: "test-internal", HeartbeatInterval: 10 * time.Second},
		Session: config.SessionConfig{
			KeepAliveDuration: 50 * time.Millisecond, MaxBrowsersPerPod: 10,
			RunnerReconnectTimeout: 200 * time.Millisecond, BrowserReconnectTimeout: 200 * time.Millisecond,
			PendingConnectionTimeout: 500 * time.Millisecond, OutputBufferSize: 1024, OutputBufferCount: 5,
		},
		Relay: config.RelayConfig{ID: "relay-1", URL: "ws://localhost:8090", Region: "test", Capacity: 100},
	}

	s := New(cfg)

	pubUpgrader := createTestWSPair(t)
	subUpgrader := createTestWSPair(t)

	if err := s.channelManager.HandlePublisherConnect("test-pod", pubUpgrader.serverConn); err != nil {
		t.Fatalf("HandlePublisherConnect: %v", err)
	}
	if err := s.channelManager.HandleSubscriberConnect("test-pod", "sub-1", subUpgrader.serverConn); err != nil {
		t.Fatalf("HandleSubscriberConnect: %v", err)
	}

	if s.channelManager.GetChannel("test-pod") == nil {
		t.Fatal("expected channel to exist")
	}

	_ = subUpgrader.clientConn.Close()

	select {
	case podKey := <-notifyCalled:
		if podKey != "test-pod" {
			t.Errorf("NotifySessionClosed podKey: got %q, want %q", podKey, "test-pod")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("onAllSubscribersGone callback was not triggered within timeout")
	}
}

func TestServer_New_OnAllSubscribersGone_NotifyFails(t *testing.T) {
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/internal/relays/session-closed" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mockBackend.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Host: "127.0.0.1", Port: 8090, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second},
		JWT:    config.JWTConfig{Secret: "test-secret", Issuer: "test-issuer"},
		Backend: config.BackendConfig{URL: mockBackend.URL, InternalAPISecret: "test-internal", HeartbeatInterval: 10 * time.Second},
		Session: config.SessionConfig{
			KeepAliveDuration: 50 * time.Millisecond, MaxBrowsersPerPod: 10,
			RunnerReconnectTimeout: 200 * time.Millisecond, BrowserReconnectTimeout: 200 * time.Millisecond,
			PendingConnectionTimeout: 500 * time.Millisecond, OutputBufferSize: 1024, OutputBufferCount: 5,
		},
		Relay: config.RelayConfig{ID: "relay-1", URL: "ws://localhost:8090", Region: "test", Capacity: 100},
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

	_ = subPair.clientConn.Close()
	time.Sleep(500 * time.Millisecond)
}

func TestServer_Start_ContextCancellation(t *testing.T) {
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockBackend.Close()

	port := findFreePort(t)
	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "127.0.0.1", Port: port, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second},
		JWT:     config.JWTConfig{Secret: "test-secret", Issuer: "test-issuer"},
		Backend: config.BackendConfig{URL: mockBackend.URL, InternalAPISecret: "test-internal", HeartbeatInterval: 10 * time.Second},
		Session: config.SessionConfig{KeepAliveDuration: 5 * time.Second, MaxBrowsersPerPod: 10},
		Relay:   config.RelayConfig{ID: "relay-ctx-test", URL: fmt.Sprintf("ws://127.0.0.1:%d", port), Region: "test", Capacity: 100},
	}

	s := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- s.Start(ctx) }()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
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
}
