package backend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClient_TLSGetters(t *testing.T) {
	c := NewClient("http://localhost", "s", "r1", "ws://a", "us", 1000)

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

func TestClient_SaveCertificateFiles(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")

	c := NewClient("http://localhost", "s", "r1", "ws://a", "us", 1000)
	c.certFile = certPath
	c.keyFile = keyPath

	if err := c.saveCertificateFiles("CERT_DATA", "KEY_DATA"); err != nil {
		t.Fatalf("saveCertificateFiles error: %v", err)
	}

	certData, _ := os.ReadFile(certPath)
	if string(certData) != "CERT_DATA" {
		t.Errorf("cert content = %q, want CERT_DATA", certData)
	}

	keyData, _ := os.ReadFile(keyPath)
	if string(keyData) != "KEY_DATA" {
		t.Errorf("key content = %q, want KEY_DATA", keyData)
	}

	info, _ := os.Stat(keyPath)
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("key file perm = %o, want 0600", perm)
	}
}

func TestClient_SaveCertificateFiles_NoPaths(t *testing.T) {
	c := NewClient("http://localhost", "s", "r1", "ws://a", "us", 1000)
	if err := c.saveCertificateFiles("CERT", "KEY"); err != nil {
		t.Errorf("saveCertificateFiles with no paths should return nil, got %v", err)
	}
}

func TestClient_SaveCertificateFiles_KeyWriteError(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "nonexistent_dir", "key.pem")

	c := NewClient("http://localhost", "s", "r1", "ws://a", "us", 1000)
	c.certFile = certPath
	c.keyFile = keyPath

	err := c.saveCertificateFiles("CERT", "KEY")
	if err == nil {
		t.Error("expected error when key file directory doesn't exist")
	}
}

func TestClient_SaveCertificateFiles_CertWriteError(t *testing.T) {
	c := NewClient("http://localhost", "s", "r1", "ws://a", "us", 1000)
	c.certFile = "/nonexistent_dir/cert.pem"
	c.keyFile = "/tmp/test_key.pem"

	err := c.saveCertificateFiles("CERT", "KEY")
	if err == nil {
		t.Error("expected error when cert file directory doesn't exist")
	}
	if !strings.Contains(err.Error(), "failed to write certificate file") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClient_LoadCertificateFiles(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")

	_ = os.WriteFile(certPath, []byte("LOADED_CERT"), 0644)
	_ = os.WriteFile(keyPath, []byte("LOADED_KEY"), 0600)

	c := NewClient("http://localhost", "s", "r1", "ws://a", "us", 1000)
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
		c := NewClient("http://localhost", "s", "r1", "ws://a", "us", 1000)
		if err := c.loadCertificateFiles(); err == nil {
			t.Error("should error when paths not configured")
		}
	})
	t.Run("file_not_exist", func(t *testing.T) {
		c := NewClient("http://localhost", "s", "r1", "ws://a", "us", 1000)
		c.certFile = "/nonexistent/cert.pem"
		c.keyFile = "/nonexistent/key.pem"
		if err := c.loadCertificateFiles(); err == nil {
			t.Error("should error when files don't exist")
		}
	})
}

func TestClient_LoadCertificateFiles_KeyReadError(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "nonexistent_key.pem")

	_ = os.WriteFile(certPath, []byte("CERT"), 0644)

	c := NewClient("http://localhost", "s", "r1", "ws://a", "us", 1000)
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

func TestClient_NewClientWithConfig(t *testing.T) {
	t.Run("with_cert_files", func(t *testing.T) {
		dir := t.TempDir()
		certPath := filepath.Join(dir, "cert.pem")
		keyPath := filepath.Join(dir, "key.pem")
		_ = os.WriteFile(certPath, []byte("CFG_CERT"), 0644)
		_ = os.WriteFile(keyPath, []byte("CFG_KEY"), 0600)

		c := NewClientWithConfig(ClientConfig{
			BaseURL: "http://localhost", InternalAPISecret: "s",
			RelayID: "r1", RelayURL: "ws://a", RelayRegion: "us", RelayCapacity: 1000,
			CertFile: certPath, KeyFile: keyPath,
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
			BaseURL: "http://localhost", InternalAPISecret: "s",
			RelayID: "r1", RelayURL: "ws://a", RelayRegion: "us", RelayCapacity: 1000,
		})
		if c.HasTLSCertificate() {
			t.Error("should not have TLS certificate without cert files")
		}
	})
}

func TestClient_Register_WithTLS(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := RegisterResponse{Status: "ok", TLSCert: "REG_CERT_PEM", TLSKey: "REG_KEY_PEM", TLSExpiry: "2027-01-01T00:00:00Z"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "s", "r1", "ws://a", "us", 1000)
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
}

func TestClient_Register_WithTLS_SaveFiles(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(RegisterResponse{Status: "ok", TLSCert: "SAVED_CERT", TLSKey: "SAVED_KEY", TLSExpiry: "2027-01-01T00:00:00Z"})
	}))
	defer srv.Close()

	c := NewClientWithConfig(ClientConfig{
		BaseURL: srv.URL, InternalAPISecret: "s",
		RelayID: "r1", RelayURL: "ws://a", RelayRegion: "us", RelayCapacity: 1000,
		CertFile: certPath, KeyFile: keyPath,
	})

	if err := c.Register(context.Background()); err != nil {
		t.Fatalf("Register error: %v", err)
	}

	certData, _ := os.ReadFile(certPath)
	if string(certData) != "SAVED_CERT" {
		t.Errorf("saved cert = %q, want SAVED_CERT", certData)
	}
	keyData, _ := os.ReadFile(keyPath)
	if string(keyData) != "SAVED_KEY" {
		t.Errorf("saved key = %q, want SAVED_KEY", keyData)
	}
}

func TestClient_Register_WithTLS_SaveFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(RegisterResponse{Status: "ok", TLSCert: "CERT", TLSKey: "KEY", TLSExpiry: "2027-01-01T00:00:00Z"})
	}))
	defer srv.Close()

	c := NewClientWithConfig(ClientConfig{
		BaseURL: srv.URL, InternalAPISecret: "s",
		RelayID: "r1", RelayURL: "ws://a", RelayRegion: "us", RelayCapacity: 1000,
		CertFile: "/nonexistent_dir/cert.pem", KeyFile: "/nonexistent_dir/key.pem",
	})

	if err := c.Register(context.Background()); err != nil {
		t.Fatalf("Register error: %v", err)
	}
	if !c.HasTLSCertificate() {
		t.Error("should have TLS cert in memory even when file save fails")
	}
}

func TestClient_SendHeartbeat_WithTLSResponse(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(HeartbeatResponse{Status: "ok", TLSCert: "HB_CERT", TLSKey: "HB_KEY", TLSExpiry: "2027-06-01T00:00:00Z"})
	}))
	defer srv.Close()

	c := NewClientWithConfig(ClientConfig{
		BaseURL: srv.URL, InternalAPISecret: "s",
		RelayID: "r1", RelayURL: "ws://a", RelayRegion: "us", RelayCapacity: 1000,
		CertFile: certPath, KeyFile: keyPath,
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

	certData, _ := os.ReadFile(certPath)
	if string(certData) != "HB_CERT" {
		t.Errorf("saved cert = %q, want HB_CERT", certData)
	}
	keyData, _ := os.ReadFile(keyPath)
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

	c := NewClient(srv.URL, "s", "r1", "ws://a", "us", 1000)
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

	c := NewClient(srv.URL, "s", "r1", "ws://a", "us", 1000)
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

func TestClient_SendHeartbeat_WithTLS_SaveFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(HeartbeatResponse{Status: "ok", TLSCert: "HB_CERT", TLSKey: "HB_KEY", TLSExpiry: "2027-06-01T00:00:00Z"})
	}))
	defer srv.Close()

	c := NewClientWithConfig(ClientConfig{
		BaseURL: srv.URL, InternalAPISecret: "s",
		RelayID: "r1", RelayURL: "ws://a", RelayRegion: "us", RelayCapacity: 1000,
		CertFile: "/nonexistent_dir/cert.pem", KeyFile: "/nonexistent_dir/key.pem",
	})
	c.mu.Lock()
	c.registered = true
	c.mu.Unlock()

	if err := c.SendHeartbeat(context.Background(), 5); err != nil {
		t.Fatalf("SendHeartbeat error: %v", err)
	}
	if !c.HasTLSCertificate() {
		t.Error("should have TLS cert in memory even when file save fails")
	}
}
