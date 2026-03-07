package config

import (
	"os"
	"testing"
	"time"
)

func clearEnv() {
	for _, env := range []string{
		"JWT_SECRET", "INTERNAL_API_SECRET", "SERVER_HOST", "SERVER_PORT", "JWT_ISSUER",
		"BACKEND_URL", "HEARTBEAT_INTERVAL", "KEEP_ALIVE_DURATION", "MAX_BROWSERS_PER_POD",
		"RELAY_ID", "RELAY_URL", "RELAY_REGION", "RELAY_CAPACITY",
		"RELAY_SERVER_HOST", "RELAY_SERVER_PORT", "RELAY_JWT_SECRET", "RELAY_JWT_ISSUER",
		"RELAY_BACKEND_URL", "RELAY_INTERNAL_API_SECRET",
		"PRIMARY_DOMAIN", "USE_HTTPS", "TLS_ENABLED", "RELAY_NAME", "RELAY_AUTO_IP",
	} {
		os.Unsetenv(env)
	}
}

func TestServerConfig_Address(t *testing.T) {
	tests := []struct{ host string; port int; expected string }{
		{"0.0.0.0", 8090, "0.0.0.0:8090"}, {"127.0.0.1", 8080, "127.0.0.1:8080"},
		{"localhost", 3000, "localhost:3000"}, {"", 80, ":80"},
	}
	for _, tt := range tests {
		cfg := &ServerConfig{Host: tt.host, Port: tt.port}
		if got := cfg.Address(); got != tt.expected {
			t.Errorf("Address() = %q, want %q", got, tt.expected)
		}
	}
}

func TestLoad_MissingSecrets(t *testing.T) {
	clearEnv()
	defer clearEnv()
	os.Setenv("INTERNAL_API_SECRET", "test")
	if _, err := Load(); err == nil {
		t.Error("expected error for missing JWT_SECRET")
	}
	clearEnv()
	os.Setenv("JWT_SECRET", "test")
	if _, err := Load(); err == nil {
		t.Error("expected error for missing INTERNAL_API_SECRET")
	}
}

func TestLoad_WithRequiredEnvVars(t *testing.T) {
	clearEnv()
	defer clearEnv()
	os.Setenv("JWT_SECRET", "test-jwt")
	os.Setenv("INTERNAL_API_SECRET", "test-internal")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	// Verify defaults
	if cfg.Server.Host != "0.0.0.0" || cfg.Server.Port != 8090 {
		t.Error("Server defaults wrong")
	}
	if cfg.Server.ReadTimeout != 30*time.Second || cfg.Server.WriteTimeout != 30*time.Second {
		t.Error("Server timeout defaults wrong")
	}
	if cfg.JWT.Issuer != "agentsmesh-relay" {
		t.Error("JWT issuer default wrong")
	}
	if cfg.Backend.URL != "http://backend:8080" || cfg.Backend.HeartbeatInterval != 10*time.Second {
		t.Error("Backend defaults wrong")
	}
	if cfg.Session.KeepAliveDuration != 30*time.Second || cfg.Session.MaxBrowsersPerPod != 10 {
		t.Error("Session defaults wrong")
	}
	if cfg.Relay.Capacity != 1000 || cfg.Relay.Region != "default" {
		t.Error("Relay defaults wrong")
	}
}

func TestLoad_EnvironmentOverrides(t *testing.T) {
	clearEnv()
	defer clearEnv()
	os.Setenv("JWT_SECRET", "test-jwt")
	os.Setenv("INTERNAL_API_SECRET", "test-internal")
	os.Setenv("SERVER_HOST", "192.168.1.1")
	os.Setenv("JWT_ISSUER", "custom-issuer")
	os.Setenv("BACKEND_URL", "http://custom:8080")
	os.Setenv("RELAY_ID", "relay-custom")
	os.Setenv("RELAY_URL", "ws://custom:8090")
	os.Setenv("RELAY_REGION", "us-west")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Server.Host != "192.168.1.1" {
		t.Error("Server.Host override failed")
	}
	if cfg.JWT.Issuer != "custom-issuer" {
		t.Error("JWT.Issuer override failed")
	}
	if cfg.Backend.URL != "http://custom:8080" {
		t.Error("Backend.URL override failed")
	}
	if cfg.Relay.ID != "relay-custom" || cfg.Relay.URL != "ws://custom:8090" || cfg.Relay.Region != "us-west" {
		t.Error("Relay overrides failed")
	}
}

func TestLoad_RelayIDDefault(t *testing.T) {
	clearEnv()
	defer clearEnv()
	os.Setenv("JWT_SECRET", "test-jwt")
	os.Setenv("INTERNAL_API_SECRET", "test-internal")
	cfg, _ := Load()
	if cfg.Relay.ID == "" || len(cfg.Relay.ID) < 6 {
		t.Error("Relay.ID should be auto-generated")
	}
}

func TestLoad_RelayURLDefault(t *testing.T) {
	clearEnv()
	defer clearEnv()
	os.Setenv("JWT_SECRET", "test-jwt")
	os.Setenv("INTERNAL_API_SECRET", "test-internal")
	cfg, _ := Load()
	if cfg.Relay.URL != "ws://localhost:8090" {
		t.Errorf("Relay.URL: expected ws://localhost:8090, got %q", cfg.Relay.URL)
	}
}

func TestLoad_WithRelayPrefix(t *testing.T) {
	clearEnv()
	defer clearEnv()
	os.Setenv("JWT_SECRET", "test-jwt")
	os.Setenv("INTERNAL_API_SECRET", "test-internal")
	os.Setenv("RELAY_SERVER_HOST", "10.0.0.1")
	os.Setenv("RELAY_JWT_ISSUER", "prefixed-issuer")
	cfg, _ := Load()
	if cfg.Server.Host != "10.0.0.1" || cfg.JWT.Issuer != "prefixed-issuer" {
		t.Error("RELAY_ prefixed env vars not working")
	}
}

func TestLoad_PrimaryDomain_WS(t *testing.T) {
	clearEnv()
	defer clearEnv()
	os.Setenv("JWT_SECRET", "test-jwt")
	os.Setenv("INTERNAL_API_SECRET", "test-internal")
	os.Setenv("PRIMARY_DOMAIN", "example.com:10000")
	// RELAY_URL not set, USE_HTTPS not set → ws://
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	want := "ws://example.com:10000/relay"
	if cfg.Relay.URL != want {
		t.Errorf("Relay.URL: got %q, want %q", cfg.Relay.URL, want)
	}
}

func TestLoad_PrimaryDomain_WSS(t *testing.T) {
	clearEnv()
	defer clearEnv()
	os.Setenv("JWT_SECRET", "test-jwt")
	os.Setenv("INTERNAL_API_SECRET", "test-internal")
	os.Setenv("PRIMARY_DOMAIN", "agentsmesh.ai")
	os.Setenv("USE_HTTPS", "true")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	want := "wss://agentsmesh.ai/relay"
	if cfg.Relay.URL != want {
		t.Errorf("Relay.URL: got %q, want %q", cfg.Relay.URL, want)
	}
}

func TestLoad_TLSEnabled_FallbackURL(t *testing.T) {
	clearEnv()
	defer clearEnv()
	os.Setenv("JWT_SECRET", "test-jwt")
	os.Setenv("INTERNAL_API_SECRET", "test-internal")
	os.Setenv("TLS_ENABLED", "true")
	// No PRIMARY_DOMAIN, no RELAY_URL → fallback to wss://localhost:port
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	want := "wss://localhost:8090"
	if cfg.Relay.URL != want {
		t.Errorf("Relay.URL: got %q, want %q", cfg.Relay.URL, want)
	}
	if !cfg.Server.TLS.Enabled {
		t.Error("TLS.Enabled should be true")
	}
}

func TestLoad_SessionConfigDefaults(t *testing.T) {
	clearEnv()
	defer clearEnv()
	os.Setenv("JWT_SECRET", "test-jwt")
	os.Setenv("INTERNAL_API_SECRET", "test-internal")
	cfg, _ := Load()
	if cfg.Session.RunnerReconnectTimeout != 30*time.Second {
		t.Errorf("RunnerReconnectTimeout: got %v, want 30s", cfg.Session.RunnerReconnectTimeout)
	}
	if cfg.Session.BrowserReconnectTimeout != 30*time.Second {
		t.Errorf("BrowserReconnectTimeout: got %v, want 30s", cfg.Session.BrowserReconnectTimeout)
	}
	if cfg.Session.PendingConnectionTimeout != 60*time.Second {
		t.Errorf("PendingConnectionTimeout: got %v, want 60s", cfg.Session.PendingConnectionTimeout)
	}
	if cfg.Session.OutputBufferSize != 256*1024 {
		t.Errorf("OutputBufferSize: got %d, want %d", cfg.Session.OutputBufferSize, 256*1024)
	}
	if cfg.Session.OutputBufferCount != 200 {
		t.Errorf("OutputBufferCount: got %d, want 200", cfg.Session.OutputBufferCount)
	}
}

func TestLoad_TLSConfig(t *testing.T) {
	clearEnv()
	defer clearEnv()
	os.Setenv("JWT_SECRET", "test-jwt")
	os.Setenv("INTERNAL_API_SECRET", "test-internal")
	os.Setenv("TLS_ENABLED", "true")
	os.Setenv("TLS_CERT_FILE", "/etc/relay/cert.pem")
	os.Setenv("TLS_KEY_FILE", "/etc/relay/key.pem")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if !cfg.Server.TLS.Enabled {
		t.Error("TLS.Enabled should be true")
	}
	if cfg.Server.TLS.CertFile != "/etc/relay/cert.pem" {
		t.Errorf("TLS.CertFile: got %q", cfg.Server.TLS.CertFile)
	}
	if cfg.Server.TLS.KeyFile != "/etc/relay/key.pem" {
		t.Errorf("TLS.KeyFile: got %q", cfg.Server.TLS.KeyFile)
	}
}

func TestLoad_RelayNameAndAutoIP(t *testing.T) {
	clearEnv()
	defer clearEnv()
	os.Setenv("JWT_SECRET", "test-jwt")
	os.Setenv("INTERNAL_API_SECRET", "test-internal")
	os.Setenv("RELAY_NAME", "us-east-1")
	os.Setenv("RELAY_AUTO_IP", "true")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Relay.Name != "us-east-1" {
		t.Errorf("Relay.Name: got %q, want us-east-1", cfg.Relay.Name)
	}
	if !cfg.Relay.AutoIP {
		t.Error("Relay.AutoIP should be true")
	}
}
