package config

import (
	"os"
	"testing"
)

// Tests for gRPC configuration and persistence

func TestConfigUsesGRPC(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		expected bool
	}{
		{
			name: "all gRPC fields set",
			cfg: Config{
				GRPCEndpoint: "localhost:9443",
				CertFile:     "/path/cert",
				KeyFile:      "/path/key",
				CAFile:       "/path/ca",
			},
			expected: true,
		},
		{
			name: "missing endpoint",
			cfg: Config{
				CertFile: "/path/cert",
				KeyFile:  "/path/key",
				CAFile:   "/path/ca",
			},
			expected: false,
		},
		{
			name: "missing cert",
			cfg: Config{
				GRPCEndpoint: "localhost:9443",
				KeyFile:      "/path/key",
				CAFile:       "/path/ca",
			},
			expected: false,
		},
		{
			name:     "empty config",
			cfg:      Config{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.UsesGRPC(); got != tt.expected {
				t.Errorf("UsesGRPC() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfigSaveAndLoadGRPCConfig(t *testing.T) {
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	cfg := &Config{}

	// Save certificates
	err := cfg.SaveCertificates([]byte("cert-pem"), []byte("key-pem"), []byte("ca-pem"))
	if err != nil {
		t.Fatalf("SaveCertificates error: %v", err)
	}

	// Clear cert paths and reload via LoadGRPCConfig (auto-detects cert files)
	cfg2 := &Config{}
	err = cfg2.LoadGRPCConfig()
	if err != nil {
		t.Fatalf("LoadGRPCConfig error: %v", err)
	}

	if cfg2.CertFile == "" {
		t.Error("CertFile should be set after load")
	}
	if cfg2.KeyFile == "" {
		t.Error("KeyFile should be set after load")
	}
	if cfg2.CAFile == "" {
		t.Error("CAFile should be set after load")
	}
}
