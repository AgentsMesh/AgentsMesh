package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDefaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ServerURL != "https://api.agentsmesh.ai" {
		t.Errorf("ServerURL: got %v, want https://api.agentsmesh.ai", cfg.ServerURL)
	}

	if cfg.MaxConcurrentPods != 5 {
		t.Errorf("MaxConcurrentPods: got %v, want 5", cfg.MaxConcurrentPods)
	}

	if cfg.WorkspaceRoot != "/workspace" {
		t.Errorf("WorkspaceRoot: got %v, want /workspace", cfg.WorkspaceRoot)
	}

	if cfg.HealthCheckPort != 9090 {
		t.Errorf("HealthCheckPort: got %v, want 9090", cfg.HealthCheckPort)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel: got %v, want info", cfg.LogLevel)
	}

	if cfg.DefaultAgent != "claude-code" {
		t.Errorf("DefaultAgent: got %v, want claude-code", cfg.DefaultAgent)
	}
}

func TestConfigNodeIDGeneration(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// NodeID should be generated from hostname
	if cfg.NodeID == "" {
		t.Error("NodeID should not be empty")
	}
}

func TestConfigValidateMissingServerURL(t *testing.T) {
	cfg := &Config{
		ServerURL: "",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for missing server_url")
	}
}

func TestConfigValidateMissingGRPCConfig(t *testing.T) {
	cfg := &Config{
		ServerURL: "https://localhost",
		// No gRPC config
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for missing gRPC config")
	}
}

func TestConfigValidateWithGRPCConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test certificate files
	certFile := filepath.Join(tmpDir, "runner.crt")
	keyFile := filepath.Join(tmpDir, "runner.key")
	caFile := filepath.Join(tmpDir, "ca.crt")

	os.WriteFile(certFile, []byte("cert"), 0600)
	os.WriteFile(keyFile, []byte("key"), 0600)
	os.WriteFile(caFile, []byte("ca"), 0644)

	cfg := &Config{
		ServerURL:         "https://localhost",
		GRPCEndpoint:      "localhost:9443",
		CertFile:          certFile,
		KeyFile:           keyFile,
		CAFile:            caFile,
		MaxConcurrentPods: 5,
		WorkspaceRoot:     tmpDir,
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigValidateInvalidMaxPods(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test certificate files
	certFile := filepath.Join(tmpDir, "runner.crt")
	keyFile := filepath.Join(tmpDir, "runner.key")
	caFile := filepath.Join(tmpDir, "ca.crt")

	os.WriteFile(certFile, []byte("cert"), 0600)
	os.WriteFile(keyFile, []byte("key"), 0600)
	os.WriteFile(caFile, []byte("ca"), 0644)

	cfg := &Config{
		ServerURL:         "https://localhost",
		GRPCEndpoint:      "localhost:9443",
		CertFile:          certFile,
		KeyFile:           keyFile,
		CAFile:            caFile,
		MaxConcurrentPods: 0,
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for invalid max_concurrent_pods")
	}
}

func TestConfigValidateCreatesWorkspaceDir(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, "new_workspace")

	// Create test certificate files
	certFile := filepath.Join(tmpDir, "runner.crt")
	keyFile := filepath.Join(tmpDir, "runner.key")
	caFile := filepath.Join(tmpDir, "ca.crt")

	os.WriteFile(certFile, []byte("cert"), 0600)
	os.WriteFile(keyFile, []byte("key"), 0600)
	os.WriteFile(caFile, []byte("ca"), 0644)

	cfg := &Config{
		ServerURL:         "https://localhost",
		GRPCEndpoint:      "localhost:9443",
		CertFile:          certFile,
		KeyFile:           keyFile,
		CAFile:            caFile,
		MaxConcurrentPods: 5,
		WorkspaceRoot:     workspaceDir,
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Directory should be created
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		t.Error("workspace directory was not created")
	}
}

func TestConfigSaveAndLoadOrgSlug(t *testing.T) {
	// Create temp home directory
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	cfg := &Config{}

	// Save org slug
	err := cfg.SaveOrgSlug("test-org")
	if err != nil {
		t.Fatalf("SaveOrgSlug error: %v", err)
	}

	if cfg.OrgSlug != "test-org" {
		t.Errorf("OrgSlug after save: got %v, want test-org", cfg.OrgSlug)
	}

	// Clear and reload
	cfg.OrgSlug = ""
	err = cfg.LoadOrgSlug()
	if err != nil {
		t.Fatalf("LoadOrgSlug error: %v", err)
	}

	if cfg.OrgSlug != "test-org" {
		t.Errorf("OrgSlug after load: got %v, want test-org", cfg.OrgSlug)
	}
}

func TestConfigLoadOrgSlugNotExists(t *testing.T) {
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	cfg := &Config{}

	// Should not error when file doesn't exist
	err := cfg.LoadOrgSlug()
	if err != nil {
		t.Errorf("LoadOrgSlug should not error when file missing: %v", err)
	}
}

func TestConfigLoadOrgSlugSkipsIfSet(t *testing.T) {
	cfg := &Config{
		OrgSlug: "existing-org",
	}

	err := cfg.LoadOrgSlug()
	if err != nil {
		t.Fatalf("LoadOrgSlug error: %v", err)
	}

	if cfg.OrgSlug != "existing-org" {
		t.Errorf("OrgSlug should remain: got %v, want existing-org", cfg.OrgSlug)
	}
}

func TestConfigFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "runner.yaml")

	content := `
server_url: https://test.example.com
node_id: test-node
grpc_endpoint: localhost:9443
max_concurrent_pods: 10
workspace_root: /tmp/test
`
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.ServerURL != "https://test.example.com" {
		t.Errorf("ServerURL: got %v, want https://test.example.com", cfg.ServerURL)
	}

	if cfg.NodeID != "test-node" {
		t.Errorf("NodeID: got %v, want test-node", cfg.NodeID)
	}

	if cfg.GRPCEndpoint != "localhost:9443" {
		t.Errorf("GRPCEndpoint: got %v, want localhost:9443", cfg.GRPCEndpoint)
	}

	if cfg.MaxConcurrentPods != 10 {
		t.Errorf("MaxConcurrentPods: got %v, want 10", cfg.MaxConcurrentPods)
	}
}

func TestConfigFromEnvironment(t *testing.T) {
	// Set environment variables
	os.Setenv("AGENTSMESH_SERVER_URL", "https://env.example.com")
	defer func() {
		os.Unsetenv("AGENTSMESH_SERVER_URL")
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.ServerURL != "https://env.example.com" {
		t.Errorf("ServerURL from env: got %v, want https://env.example.com", cfg.ServerURL)
	}
}

func TestConfigWorkspaceRootExpansion(t *testing.T) {
	os.Setenv("TEST_WORKSPACE", "/custom/workspace")
	defer os.Unsetenv("TEST_WORKSPACE")

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "runner.yaml")

	content := `
server_url: https://test.example.com
workspace_root: $TEST_WORKSPACE
`
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.WorkspaceRoot != "/custom/workspace" {
		t.Errorf("WorkspaceRoot: got %v, want /custom/workspace", cfg.WorkspaceRoot)
	}
}

func TestConfigStruct(t *testing.T) {
	cfg := Config{
		ServerURL:         "https://localhost",
		NodeID:            "test-node",
		Description:       "Test runner",
		GRPCEndpoint:      "localhost:9443",
		CertFile:          "/path/to/cert",
		KeyFile:           "/path/to/key",
		CAFile:            "/path/to/ca",
		OrgSlug:           "test-org",
		MaxConcurrentPods: 5,
		WorkspaceRoot:     "/workspace",
		GitConfigPath:     "/git/config",
		RepositoryPath:    "/repo",
		WorktreesDir:      "/worktrees",
		BaseBranch:        "main",
		MCPConfigPath:     "/mcp/config",
		DefaultAgent:      "claude-code",
		DefaultShell:      "/bin/bash",
		AgentEnvVars:      map[string]string{"KEY": "VALUE"},
		HealthCheckPort:   9090,
		LogLevel:          "debug",
		LogFile:           "/var/log/runner.log",
	}

	if cfg.ServerURL != "https://localhost" {
		t.Errorf("ServerURL: got %v, want https://localhost", cfg.ServerURL)
	}

	if cfg.NodeID != "test-node" {
		t.Errorf("NodeID: got %v, want test-node", cfg.NodeID)
	}

	if cfg.GRPCEndpoint != "localhost:9443" {
		t.Errorf("GRPCEndpoint: got %v, want localhost:9443", cfg.GRPCEndpoint)
	}

	if cfg.AgentEnvVars["KEY"] != "VALUE" {
		t.Errorf("AgentEnvVars: got %v, want VALUE", cfg.AgentEnvVars["KEY"])
	}
}

func TestConfigInvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid.yaml")

	content := `
server_url: [invalid yaml
`
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := Load(configFile)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

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

	// Save gRPC endpoint
	err := cfg.SaveGRPCEndpoint("grpc.example.com:9443")
	if err != nil {
		t.Fatalf("SaveGRPCEndpoint error: %v", err)
	}

	// Save certificates
	err = cfg.SaveCertificates([]byte("cert-pem"), []byte("key-pem"), []byte("ca-pem"))
	if err != nil {
		t.Fatalf("SaveCertificates error: %v", err)
	}

	// Clear and reload
	cfg2 := &Config{}
	err = cfg2.LoadGRPCConfig()
	if err != nil {
		t.Fatalf("LoadGRPCConfig error: %v", err)
	}

	if cfg2.GRPCEndpoint != "grpc.example.com:9443" {
		t.Errorf("GRPCEndpoint after load: got %v, want grpc.example.com:9443", cfg2.GRPCEndpoint)
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

func TestConfigValidateGRPCMissingFiles(t *testing.T) {
	cfg := &Config{
		ServerURL:    "https://localhost",
		GRPCEndpoint: "localhost:9443",
		CertFile:     "/nonexistent/cert",
		KeyFile:      "/nonexistent/key",
		CAFile:       "/nonexistent/ca",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for missing certificate files")
	}
}
