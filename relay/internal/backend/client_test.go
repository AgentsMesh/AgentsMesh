package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	c := NewClient("http://localhost:8080", "s", "r1", "ws://a", "ws://b", "us", 1000)
	if c == nil || c.baseURL != "http://localhost:8080" || c.IsRegistered() {
		t.Error("client init failed")
	}
}

func TestClient_Register(t *testing.T) {
	for _, tt := range []struct{ name string; status int; wantErr bool }{
		{"ok", http.StatusOK, false}, {"err", http.StatusInternalServerError, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()
			c := NewClient(srv.URL, "s", "r1", "ws://a", "", "us", 1000)
			if err := c.Register(context.Background()); (err != nil) != tt.wantErr {
				t.Error("mismatch")
			}
		})
	}
	c := NewClient("http://127.0.0.1:1", "s", "r1", "ws://a", "", "us", 1000)
	if c.Register(context.Background()) == nil {
		t.Error("should fail")
	}
}

func TestClient_SendHeartbeat(t *testing.T) {
	for _, tt := range []struct{ name string; status int; reg, wantErr bool }{
		{"ok", http.StatusOK, true, false}, {"not_reg", http.StatusOK, false, true},
		{"404", http.StatusNotFound, true, true}, {"500", http.StatusInternalServerError, true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()
			c := NewClient(srv.URL, "s", "r1", "ws://a", "", "us", 1000)
			c.mu.Lock()
			c.registered = tt.reg
			c.mu.Unlock()
			if err := c.SendHeartbeat(context.Background(), 5); (err != nil) != tt.wantErr {
				t.Error("mismatch")
			}
		})
	}
	var req HeartbeatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "s", "r1", "ws://a", "", "us", 1000)
	c.mu.Lock()
	c.registered = true
	c.mu.Unlock()
	_ = c.SendHeartbeat(context.Background(), 5)
	if req.Connections != 5 {
		t.Error("data wrong")
	}
	c2 := NewClient("http://127.0.0.1:1", "s", "r1", "ws://a", "", "us", 1000)
	c2.mu.Lock()
	c2.registered = true
	c2.mu.Unlock()
	if c2.SendHeartbeat(context.Background(), 5) == nil {
		t.Error("should fail")
	}
}

func TestClient_NotifySessionClosed(t *testing.T) {
	for _, tt := range []struct{ name string; status int; wantErr bool }{
		{"ok", http.StatusOK, false}, {"err", http.StatusInternalServerError, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()
			c := NewClient(srv.URL, "s", "r1", "ws://a", "", "us", 1000)
			if err := c.NotifySessionClosed(context.Background(), "p1", "s1"); (err != nil) != tt.wantErr {
				t.Error("mismatch")
			}
		})
	}
	c := NewClient("http://127.0.0.1:1", "s", "r1", "ws://a", "", "us", 1000)
	if c.NotifySessionClosed(context.Background(), "p1", "s1") == nil {
		t.Error("should fail")
	}
}

func TestClient_StartHeartbeat(t *testing.T) {
	var count int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/internal/relays/heartbeat" {
			atomic.AddInt32(&count, 1)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "s", "r1", "ws://a", "", "us", 1000)
	c.mu.Lock()
	c.registered = true
	c.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	go c.StartHeartbeat(ctx, 40*time.Millisecond, func() int { return 1 })
	<-ctx.Done()
	if atomic.LoadInt32(&count) < 1 {
		t.Error("should heartbeat")
	}
}

func TestClient_StartHeartbeat_ReRegister(t *testing.T) {
	var hb, reg int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/internal/relays/heartbeat" {
			if atomic.AddInt32(&hb, 1) == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
		}
		if r.URL.Path == "/api/internal/relays/register" {
			atomic.AddInt32(&reg, 1)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "s", "r1", "ws://a", "", "us", 1000)
	c.mu.Lock()
	c.registered = true
	c.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	go c.StartHeartbeat(ctx, 50*time.Millisecond, func() int { return 1 })
	<-ctx.Done()
	if atomic.LoadInt32(&reg) < 1 {
		t.Error("should re-register")
	}
}

func TestClient_StartHeartbeat_ReRegisterFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "s", "r1", "ws://a", "", "us", 1000)
	c.mu.Lock()
	c.registered = true
	c.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	go c.StartHeartbeat(ctx, 30*time.Millisecond, func() int { return 1 })
	<-ctx.Done()
}

func TestRequestStructs(t *testing.T) {
	reg := RegisterRequest{RelayID: "r1", URL: "ws://x", InternalURL: "ws://y", Region: "us", Capacity: 100}
	hb := HeartbeatRequest{RelayID: "r1", Connections: 50, CPUUsage: 25.5, MemoryUsage: 60.0}
	sc := SessionClosedRequest{PodKey: "p1", SessionID: "s1"}
	if reg.RelayID != "r1" || hb.Connections != 50 || sc.PodKey != "p1" {
		t.Error("fields wrong")
	}
	data, _ := json.Marshal(reg)
	var dec RegisterRequest
	_ = json.Unmarshal(data, &dec)
	if dec.RelayID != reg.RelayID {
		t.Error("roundtrip failed")
	}
}

func TestClient_GetRelayURL(t *testing.T) {
	c := NewClient("http://localhost", "s", "r1", "ws://relay.test", "", "us", 1000)
	if got := c.GetRelayURL(); got != "ws://relay.test" {
		t.Errorf("GetRelayURL = %q, want %q", got, "ws://relay.test")
	}
}

func TestClient_TLSGetters(t *testing.T) {
	c := NewClient("http://localhost", "s", "r1", "ws://a", "", "us", 1000)

	// Initially no TLS certificate
	if c.HasTLSCertificate() {
		t.Error("HasTLSCertificate should be false initially")
	}
	cert, key := c.GetTLSCertificate()
	if cert != "" || key != "" {
		t.Error("GetTLSCertificate should return empty strings initially")
	}
	if c.GetTLSExpiry() != "" {
		t.Error("GetTLSExpiry should be empty initially")
	}

	// Set TLS fields
	c.mu.Lock()
	c.tlsCert = "CERT_PEM"
	c.tlsKey = "KEY_PEM"
	c.tlsExpiry = "2026-12-31T00:00:00Z"
	c.mu.Unlock()

	if !c.HasTLSCertificate() {
		t.Error("HasTLSCertificate should be true after setting")
	}
	cert, key = c.GetTLSCertificate()
	if cert != "CERT_PEM" || key != "KEY_PEM" {
		t.Errorf("GetTLSCertificate = (%q, %q), want (CERT_PEM, KEY_PEM)", cert, key)
	}
	if c.GetTLSExpiry() != "2026-12-31T00:00:00Z" {
		t.Errorf("GetTLSExpiry = %q, want 2026-12-31T00:00:00Z", c.GetTLSExpiry())
	}
}

func TestClient_Unregister(t *testing.T) {
	for _, tt := range []struct {
		name    string
		status  int
		wantErr bool
	}{
		{"ok", http.StatusOK, false},
		{"server_error", http.StatusInternalServerError, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()
			c := NewClient(srv.URL, "s", "r1", "ws://a", "", "us", 1000)
			c.mu.Lock()
			c.registered = true
			c.mu.Unlock()
			err := c.Unregister(context.Background(), "shutdown")
			if (err != nil) != tt.wantErr {
				t.Errorf("Unregister error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && c.IsRegistered() {
				t.Error("should be unregistered after success")
			}
		})
	}
	t.Run("network_error", func(t *testing.T) {
		c := NewClient("http://127.0.0.1:1", "s", "r1", "ws://a", "", "us", 1000)
		if c.Unregister(context.Background(), "shutdown") == nil {
			t.Error("should fail on network error")
		}
	})
}

func TestClient_SaveCertificateFiles(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")

	c := NewClient("http://localhost", "s", "r1", "ws://a", "", "us", 1000)
	c.certFile = certPath
	c.keyFile = keyPath

	if err := c.saveCertificateFiles("CERT_DATA", "KEY_DATA"); err != nil {
		t.Fatalf("saveCertificateFiles error: %v", err)
	}

	certData, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read cert file: %v", err)
	}
	if string(certData) != "CERT_DATA" {
		t.Errorf("cert content = %q, want CERT_DATA", certData)
	}

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("read key file: %v", err)
	}
	if string(keyData) != "KEY_DATA" {
		t.Errorf("key content = %q, want KEY_DATA", keyData)
	}

	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat key file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("key file perm = %o, want 0600", perm)
	}
}

func TestClient_SaveCertificateFiles_NoPaths(t *testing.T) {
	c := NewClient("http://localhost", "s", "r1", "ws://a", "", "us", 1000)
	// certFile and keyFile are empty by default
	if err := c.saveCertificateFiles("CERT", "KEY"); err != nil {
		t.Errorf("saveCertificateFiles with no paths should return nil, got %v", err)
	}
}

func TestClient_LoadCertificateFiles(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")

	if err := os.WriteFile(certPath, []byte("LOADED_CERT"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, []byte("LOADED_KEY"), 0600); err != nil {
		t.Fatal(err)
	}

	c := NewClient("http://localhost", "s", "r1", "ws://a", "", "us", 1000)
	c.certFile = certPath
	c.keyFile = keyPath

	if err := c.loadCertificateFiles(); err != nil {
		t.Fatalf("loadCertificateFiles error: %v", err)
	}
	if c.tlsCert != "LOADED_CERT" {
		t.Errorf("tlsCert = %q, want LOADED_CERT", c.tlsCert)
	}
	if c.tlsKey != "LOADED_KEY" {
		t.Errorf("tlsKey = %q, want LOADED_KEY", c.tlsKey)
	}
}

func TestClient_LoadCertificateFiles_Errors(t *testing.T) {
	t.Run("paths_not_configured", func(t *testing.T) {
		c := NewClient("http://localhost", "s", "r1", "ws://a", "", "us", 1000)
		if err := c.loadCertificateFiles(); err == nil {
			t.Error("should error when paths not configured")
		}
	})
	t.Run("file_not_exist", func(t *testing.T) {
		c := NewClient("http://localhost", "s", "r1", "ws://a", "", "us", 1000)
		c.certFile = "/nonexistent/cert.pem"
		c.keyFile = "/nonexistent/key.pem"
		if err := c.loadCertificateFiles(); err == nil {
			t.Error("should error when files don't exist")
		}
	})
}

func TestClient_NewClientWithConfig(t *testing.T) {
	t.Run("with_cert_files", func(t *testing.T) {
		dir := t.TempDir()
		certPath := filepath.Join(dir, "cert.pem")
		keyPath := filepath.Join(dir, "key.pem")
		if err := os.WriteFile(certPath, []byte("CFG_CERT"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(keyPath, []byte("CFG_KEY"), 0600); err != nil {
			t.Fatal(err)
		}

		c := NewClientWithConfig(ClientConfig{
			BaseURL:           "http://localhost",
			InternalAPISecret: "s",
			RelayID:           "r1",
			RelayURL:          "ws://a",
			RelayRegion:       "us",
			RelayCapacity:     1000,
			CertFile:          certPath,
			KeyFile:           keyPath,
		})
		if !c.HasTLSCertificate() {
			t.Error("should have TLS certificate after loading from files")
		}
		cert, key := c.GetTLSCertificate()
		if cert != "CFG_CERT" || key != "CFG_KEY" {
			t.Errorf("cert = %q, key = %q, want CFG_CERT/CFG_KEY", cert, key)
		}
	})
	t.Run("without_cert_files", func(t *testing.T) {
		c := NewClientWithConfig(ClientConfig{
			BaseURL:           "http://localhost",
			InternalAPISecret: "s",
			RelayID:           "r1",
			RelayURL:          "ws://a",
			RelayRegion:       "us",
			RelayCapacity:     1000,
		})
		if c.HasTLSCertificate() {
			t.Error("should not have TLS certificate without cert files")
		}
	})
}

func TestClient_Register_WithTLS(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := RegisterResponse{
			Status:    "ok",
			TLSCert:   "REG_CERT_PEM",
			TLSKey:    "REG_KEY_PEM",
			TLSExpiry: "2027-01-01T00:00:00Z",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "s", "r1", "ws://a", "", "us", 1000)
	if err := c.Register(context.Background()); err != nil {
		t.Fatalf("Register error: %v", err)
	}
	if !c.HasTLSCertificate() {
		t.Error("should have TLS certificate after register")
	}
	cert, key := c.GetTLSCertificate()
	if cert != "REG_CERT_PEM" || key != "REG_KEY_PEM" {
		t.Errorf("cert = %q, key = %q, want REG_CERT_PEM/REG_KEY_PEM", cert, key)
	}
	if c.GetTLSExpiry() != "2027-01-01T00:00:00Z" {
		t.Errorf("expiry = %q, want 2027-01-01T00:00:00Z", c.GetTLSExpiry())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// errorReader is an io.Reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (int, error) {
	return 0, fmt.Errorf("simulated read error")
}

func TestClient_Register_DNSCreated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(RegisterResponse{
			Status:     "ok",
			URL:        "wss://us-east-1.relay.example.com",
			DNSCreated: true,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "s", "r1", "", "", "us", 1000)
	if err := c.Register(context.Background()); err != nil {
		t.Fatalf("Register error: %v", err)
	}
	if got := c.GetRelayURL(); got != "wss://us-east-1.relay.example.com" {
		t.Errorf("GetRelayURL() = %q, want %q", got, "wss://us-east-1.relay.example.com")
	}
}

func TestClient_Register_DNSCreated_URLWithoutDNSFlag(t *testing.T) {
	// When URL is returned but DNSCreated is false, relayURL should NOT be updated
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(RegisterResponse{
			Status:     "ok",
			URL:        "wss://ignored.example.com",
			DNSCreated: false,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "s", "r1", "ws://original", "", "us", 1000)
	if err := c.Register(context.Background()); err != nil {
		t.Fatalf("Register error: %v", err)
	}
	if got := c.GetRelayURL(); got != "ws://original" {
		t.Errorf("GetRelayURL() = %q, want %q (should not be updated)", got, "ws://original")
	}
}

func TestClient_Register_WithTLS_SaveFiles(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(RegisterResponse{
			Status:    "ok",
			TLSCert:   "SAVED_CERT",
			TLSKey:    "SAVED_KEY",
			TLSExpiry: "2027-01-01T00:00:00Z",
		})
	}))
	defer srv.Close()

	c := NewClientWithConfig(ClientConfig{
		BaseURL:           srv.URL,
		InternalAPISecret: "s",
		RelayID:           "r1",
		RelayURL:          "ws://a",
		RelayRegion:       "us",
		RelayCapacity:     1000,
		CertFile:          certPath,
		KeyFile:           keyPath,
	})

	if err := c.Register(context.Background()); err != nil {
		t.Fatalf("Register error: %v", err)
	}

	// Verify cert files were written
	certData, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read cert file: %v", err)
	}
	if string(certData) != "SAVED_CERT" {
		t.Errorf("saved cert = %q, want SAVED_CERT", certData)
	}

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("read key file: %v", err)
	}
	if string(keyData) != "SAVED_KEY" {
		t.Errorf("saved key = %q, want SAVED_KEY", keyData)
	}
}

func TestClient_Register_WithAutoIP(t *testing.T) {
	var capturedReq RegisterRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedReq)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(RegisterResponse{Status: "ok"})
	}))
	defer srv.Close()

	c := NewClientWithConfig(ClientConfig{
		BaseURL:           srv.URL,
		InternalAPISecret: "s",
		RelayID:           "r1",
		RelayName:         "us-east-1",
		RelayRegion:       "us",
		RelayCapacity:     1000,
		AutoIP:            true,
	})

	// Replace transport: mock IP detection services, proxy backend requests
	origTransport := c.httpClient.Transport
	c.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Host, "ipify") ||
			strings.Contains(r.URL.Host, "ifconfig") ||
			strings.Contains(r.URL.Host, "icanhazip") {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("5.6.7.8")),
			}, nil
		}
		// Proxy to httptest server
		if origTransport != nil {
			return origTransport.RoundTrip(r)
		}
		return http.DefaultTransport.RoundTrip(r)
	})

	if err := c.Register(context.Background()); err != nil {
		t.Fatalf("Register error: %v", err)
	}
	if capturedReq.IP != "5.6.7.8" {
		t.Errorf("IP = %q, want 5.6.7.8", capturedReq.IP)
	}
	if capturedReq.RelayName != "us-east-1" {
		t.Errorf("RelayName = %q, want us-east-1", capturedReq.RelayName)
	}
}

func TestClient_SendHeartbeat_WithTLSResponse(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(HeartbeatResponse{
			Status:    "ok",
			TLSCert:   "HB_CERT",
			TLSKey:    "HB_KEY",
			TLSExpiry: "2027-06-01T00:00:00Z",
		})
	}))
	defer srv.Close()

	c := NewClientWithConfig(ClientConfig{
		BaseURL:           srv.URL,
		InternalAPISecret: "s",
		RelayID:           "r1",
		RelayURL:          "ws://a",
		RelayRegion:       "us",
		RelayCapacity:     1000,
		CertFile:          certPath,
		KeyFile:           keyPath,
	})
	c.mu.Lock()
	c.registered = true
	c.mu.Unlock()

	if err := c.SendHeartbeat(context.Background(), 5); err != nil {
		t.Fatalf("SendHeartbeat error: %v", err)
	}

	if !c.HasTLSCertificate() {
		t.Error("should have TLS cert after heartbeat")
	}
	cert, key := c.GetTLSCertificate()
	if cert != "HB_CERT" || key != "HB_KEY" {
		t.Errorf("cert=%q key=%q, want HB_CERT/HB_KEY", cert, key)
	}

	// Verify files were saved
	certData, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read cert file: %v", err)
	}
	if string(certData) != "HB_CERT" {
		t.Errorf("saved cert = %q, want HB_CERT", certData)
	}
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("read key file: %v", err)
	}
	if string(keyData) != "HB_KEY" {
		t.Errorf("saved key = %q, want HB_KEY", keyData)
	}
}

func TestClient_SendHeartbeat_NeedCert(t *testing.T) {
	var captured HeartbeatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "s", "r1", "ws://a", "", "us", 1000)
	c.mu.Lock()
	c.registered = true
	c.mu.Unlock()

	_ = c.SendHeartbeat(context.Background(), 1)
	if !captured.NeedCert {
		t.Error("NeedCert should be true when no TLS cert is set")
	}
}

func TestClient_SendHeartbeat_NeedCert_False(t *testing.T) {
	var captured HeartbeatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "s", "r1", "ws://a", "", "us", 1000)
	c.mu.Lock()
	c.registered = true
	c.tlsCert = "EXISTING_CERT"
	c.tlsKey = "EXISTING_KEY"
	c.mu.Unlock()

	_ = c.SendHeartbeat(context.Background(), 1)
	if captured.NeedCert {
		t.Error("NeedCert should be false when TLS cert is already set")
	}
}

func TestClient_SaveCertificateFiles_KeyWriteError(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "nonexistent_dir", "key.pem")

	c := NewClient("http://localhost", "s", "r1", "ws://a", "", "us", 1000)
	c.certFile = certPath
	c.keyFile = keyPath

	err := c.saveCertificateFiles("CERT", "KEY")
	if err == nil {
		t.Error("expected error when key file directory doesn't exist")
	}
}

func TestClient_Register_WithAutoIP_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(RegisterResponse{Status: "ok"})
	}))
	defer srv.Close()

	c := NewClientWithConfig(ClientConfig{
		BaseURL:           srv.URL,
		InternalAPISecret: "s",
		RelayID:           "r1",
		RelayName:         "us-east-1",
		RelayRegion:       "us",
		RelayCapacity:     1000,
		AutoIP:            true,
	})

	// All IP detection services fail → warning is logged but register succeeds
	c.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Host, "ipify") ||
			strings.Contains(r.URL.Host, "ifconfig") ||
			strings.Contains(r.URL.Host, "icanhazip") {
			return nil, fmt.Errorf("connection refused")
		}
		return http.DefaultTransport.RoundTrip(r)
	})

	if err := c.Register(context.Background()); err != nil {
		t.Fatalf("Register should succeed even when IP detection fails: %v", err)
	}
	if c.relayIP != "" {
		t.Errorf("relayIP should be empty when detection fails, got %q", c.relayIP)
	}
}

func TestClient_Register_WithTLS_SaveFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(RegisterResponse{
			Status:    "ok",
			TLSCert:   "CERT",
			TLSKey:    "KEY",
			TLSExpiry: "2027-01-01T00:00:00Z",
		})
	}))
	defer srv.Close()

	c := NewClientWithConfig(ClientConfig{
		BaseURL:           srv.URL,
		InternalAPISecret: "s",
		RelayID:           "r1",
		RelayURL:          "ws://a",
		RelayRegion:       "us",
		RelayCapacity:     1000,
		CertFile:          "/nonexistent_dir/cert.pem",
		KeyFile:           "/nonexistent_dir/key.pem",
	})

	// Register succeeds but save cert files logs warning
	if err := c.Register(context.Background()); err != nil {
		t.Fatalf("Register error: %v", err)
	}
	// Cert is still stored in memory
	if !c.HasTLSCertificate() {
		t.Error("should have TLS cert in memory even when file save fails")
	}
}

func TestClient_SendHeartbeat_WithTLS_SaveFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(HeartbeatResponse{
			Status:    "ok",
			TLSCert:   "HB_CERT",
			TLSKey:    "HB_KEY",
			TLSExpiry: "2027-06-01T00:00:00Z",
		})
	}))
	defer srv.Close()

	c := NewClientWithConfig(ClientConfig{
		BaseURL:           srv.URL,
		InternalAPISecret: "s",
		RelayID:           "r1",
		RelayURL:          "ws://a",
		RelayRegion:       "us",
		RelayCapacity:     1000,
		CertFile:          "/nonexistent_dir/cert.pem",
		KeyFile:           "/nonexistent_dir/key.pem",
	})
	c.mu.Lock()
	c.registered = true
	c.mu.Unlock()

	// Heartbeat succeeds, save cert files logs warning (doesn't fail the call)
	if err := c.SendHeartbeat(context.Background(), 5); err != nil {
		t.Fatalf("SendHeartbeat error: %v", err)
	}
	// Cert is still stored in memory
	if !c.HasTLSCertificate() {
		t.Error("should have TLS cert in memory even when file save fails")
	}
}

func TestClient_SaveCertificateFiles_CertWriteError(t *testing.T) {
	c := NewClient("http://localhost", "s", "r1", "ws://a", "", "us", 1000)
	c.certFile = "/nonexistent_dir/cert.pem" // directory doesn't exist
	c.keyFile = "/tmp/test_key.pem"

	err := c.saveCertificateFiles("CERT", "KEY")
	if err == nil {
		t.Error("expected error when cert file directory doesn't exist")
	}
	if !strings.Contains(err.Error(), "failed to write certificate file") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClient_LoadCertificateFiles_KeyReadError(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "nonexistent_key.pem")

	// Create cert file but NOT key file
	if err := os.WriteFile(certPath, []byte("CERT"), 0644); err != nil {
		t.Fatal(err)
	}

	c := NewClient("http://localhost", "s", "r1", "ws://a", "", "us", 1000)
	c.certFile = certPath
	c.keyFile = keyPath

	err := c.loadCertificateFiles()
	if err == nil {
		t.Error("expected error when key file doesn't exist")
	}
	if !strings.Contains(err.Error(), "failed to read key file") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClient_DetectPublicIP(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c := NewClient("http://localhost", "s", "r1", "ws://a", "", "us", 1000)
		c.httpClient = &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("1.2.3.4\n")),
				}, nil
			}),
		}
		ip, err := c.detectPublicIP(context.Background())
		if err != nil {
			t.Fatalf("detectPublicIP error: %v", err)
		}
		if ip != "1.2.3.4" {
			t.Errorf("ip = %q, want 1.2.3.4", ip)
		}
	})
	t.Run("all_fail", func(t *testing.T) {
		c := NewClient("http://localhost", "s", "r1", "ws://a", "", "us", 1000)
		c.httpClient = &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return nil, fmt.Errorf("connection refused")
			}),
		}
		_, err := c.detectPublicIP(context.Background())
		if err == nil {
			t.Error("should error when all services fail")
		}
	})
	t.Run("html_response_skipped", func(t *testing.T) {
		c := NewClient("http://localhost", "s", "r1", "ws://a", "", "us", 1000)
		callCount := 0
		c.httpClient = &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				callCount++
				// All services return HTML
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("<html>blocked</html>")),
				}, nil
			}),
		}
		_, err := c.detectPublicIP(context.Background())
		if err == nil {
			t.Error("should error when all responses contain HTML")
		}
	})
	t.Run("body_read_error", func(t *testing.T) {
		c := NewClient("http://localhost", "s", "r1", "ws://a", "", "us", 1000)
		c.httpClient = &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(&errorReader{}),
				}, nil
			}),
		}
		_, err := c.detectPublicIP(context.Background())
		if err == nil {
			t.Error("should error when body read fails")
		}
	})
}
