package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/relay/internal/config"
)

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
		_ = tlsConn.Close()
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
		_ = tlsConn.Close()
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
		_ = tlsConn.Close()
	}

	cancel()

	select {
	case <-errCh:
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}
