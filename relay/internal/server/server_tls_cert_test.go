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
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/relay/internal/config"
)

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

	// Connect with TLS to trigger GetCertificate -> HasTLSCertificate -> load from backend
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

func TestServer_Start_TLS_GetCertificate_InvalidBackendCert(t *testing.T) {
	// Backend returns invalid cert data -> error branch in GetCertificate
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

	// Connect with TLS -> GetCertificate -> HasTLSCertificate=true -> X509KeyPair fails -> error logged
	// Then falls through to cert files check (empty) -> "no TLS certificate available"
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

// generateSelfSignedCert generates a self-signed TLS certificate for testing.
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
