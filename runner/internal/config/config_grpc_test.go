package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/textutil"
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

func TestUpdateGRPCEndpointInFile_PreservesCommentsAndOrder(t *testing.T) {
	original := `# AgentsMesh runner config
# Managed by: agentsmesh-runner register
server_url: https://app.example.com
grpc_endpoint: grpcs://old.example.com:9443
runner_id: abc-123
# Certificate paths
cert_file: /home/user/.agentsmesh/certs/runner.crt
key_file: /home/user/.agentsmesh/certs/runner.key
ca_file: /home/user/.agentsmesh/certs/ca.crt
`
	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tmpFile, []byte(original), 0600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	if err := UpdateGRPCEndpointInFile(tmpFile, "grpcs://new.example.com:9443"); err != nil {
		t.Fatalf("UpdateGRPCEndpointInFile error: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read updated config: %v", err)
	}
	content := string(data)

	// New endpoint must appear.
	if !containsLine(content, "grpc_endpoint: grpcs://new.example.com:9443") {
		t.Errorf("new endpoint not found in file:\n%s", content)
	}
	// Old endpoint must be gone.
	if containsLine(content, "grpc_endpoint: grpcs://old.example.com:9443") {
		t.Errorf("old endpoint still present in file:\n%s", content)
	}
	// Comments must be preserved.
	if !containsLine(content, "# AgentsMesh runner config") {
		t.Errorf("top comment lost:\n%s", content)
	}
	if !containsLine(content, "# Certificate paths") {
		t.Errorf("inline comment lost:\n%s", content)
	}
	// Key order must be preserved (server_url before runner_id before cert_file).
	sIdx := indexOf(content, "server_url:")
	rIdx := indexOf(content, "runner_id:")
	cIdx := indexOf(content, "cert_file:")
	if sIdx >= rIdx || rIdx >= cIdx {
		t.Errorf("key order changed: server_url=%d runner_id=%d cert_file=%d\n%s", sIdx, rIdx, cIdx, content)
	}
}

func TestUpdateGRPCEndpointInFile_AppendsWhenMissing(t *testing.T) {
	original := "server_url: https://app.example.com\nrunner_id: abc-123\n"
	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tmpFile, []byte(original), 0600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	if err := UpdateGRPCEndpointInFile(tmpFile, "grpcs://new.example.com:9443"); err != nil {
		t.Fatalf("UpdateGRPCEndpointInFile error: %v", err)
	}

	data, _ := os.ReadFile(tmpFile)
	content := string(data)
	if !containsLine(content, "grpc_endpoint: grpcs://new.example.com:9443") {
		t.Errorf("appended endpoint not found:\n%s", content)
	}
}

func containsLine(s, substr string) bool {
	for _, line := range textutil.SplitLines(s) {
		if line != "" && line == substr {
			return true
		}
	}
	return false
}

func indexOf(s, sub string) int {
	for i := range s {
		if len(s)-i >= len(sub) && s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestConfigSaveAndLoadGRPCConfig(t *testing.T) {
	tmpHome := t.TempDir()
	// os.UserHomeDir() checks USERPROFILE first on Windows, HOME on Unix.
	t.Setenv("HOME", tmpHome)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmpHome)
	}

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
