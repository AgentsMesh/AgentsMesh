package config

import (
	"os"
	"path/filepath"
	"testing"
)

// Tests for config defaults and basic loading

func TestConfigDefaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ServerURL != "https://agentsmesh.ai" {
		t.Errorf("ServerURL: got %v, want https://agentsmesh.ai", cfg.ServerURL)
	}

	if cfg.MaxConcurrentPods != 5 {
		t.Errorf("MaxConcurrentPods: got %v, want 5", cfg.MaxConcurrentPods)
	}

	expectedRoot := DefaultWorkspaceRoot()
	if cfg.WorkspaceRoot != expectedRoot {
		t.Errorf("WorkspaceRoot: got %v, want %v", cfg.WorkspaceRoot, expectedRoot)
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
