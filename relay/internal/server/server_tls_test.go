package server

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
		_ = tlsConn.Close()
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
		_ = tlsConn.Close()
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
