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
		AuthToken: "test-token",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for missing server_url")
	}
}

func TestConfigValidateMissingAuth(t *testing.T) {
	cfg := &Config{
		ServerURL:         "http://localhost",
		AuthToken:         "",
		RegistrationToken: "",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for missing auth token")
	}
}

func TestConfigValidateWithAuthToken(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		ServerURL:             "http://localhost",
		AuthToken:             "test-token",
		MaxConcurrentPods: 5,
		WorkspaceRoot:         tmpDir,
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigValidateWithRegistrationToken(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		ServerURL:             "http://localhost",
		RegistrationToken:     "reg-token",
		MaxConcurrentPods: 5,
		WorkspaceRoot:         tmpDir,
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigValidateInvalidMaxPods(t *testing.T) {
	cfg := &Config{
		ServerURL:             "http://localhost",
		AuthToken:             "test-token",
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

	cfg := &Config{
		ServerURL:             "http://localhost",
		AuthToken:             "test-token",
		MaxConcurrentPods: 5,
		WorkspaceRoot:         workspaceDir,
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

func TestConfigSaveAndLoadAuthToken(t *testing.T) {
	// Create temp home directory
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	cfg := &Config{}

	// Save token
	err := cfg.SaveAuthToken("saved-token")
	if err != nil {
		t.Fatalf("SaveAuthToken error: %v", err)
	}

	if cfg.AuthToken != "saved-token" {
		t.Errorf("AuthToken after save: got %v, want saved-token", cfg.AuthToken)
	}

	// Clear and reload
	cfg.AuthToken = ""
	err = cfg.LoadAuthToken()
	if err != nil {
		t.Fatalf("LoadAuthToken error: %v", err)
	}

	if cfg.AuthToken != "saved-token" {
		t.Errorf("AuthToken after load: got %v, want saved-token", cfg.AuthToken)
	}
}

func TestConfigLoadAuthTokenNotExists(t *testing.T) {
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	cfg := &Config{}

	// Should not error when file doesn't exist
	err := cfg.LoadAuthToken()
	if err != nil {
		t.Errorf("LoadAuthToken should not error when file missing: %v", err)
	}
}

func TestConfigLoadAuthTokenSkipsIfSet(t *testing.T) {
	cfg := &Config{
		AuthToken: "existing-token",
	}

	err := cfg.LoadAuthToken()
	if err != nil {
		t.Fatalf("LoadAuthToken error: %v", err)
	}

	if cfg.AuthToken != "existing-token" {
		t.Errorf("AuthToken should remain: got %v, want existing-token", cfg.AuthToken)
	}
}

func TestConfigFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "runner.yaml")

	content := `
server_url: http://test.example.com
node_id: test-node
auth_token: test-token
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

	if cfg.ServerURL != "http://test.example.com" {
		t.Errorf("ServerURL: got %v, want http://test.example.com", cfg.ServerURL)
	}

	if cfg.NodeID != "test-node" {
		t.Errorf("NodeID: got %v, want test-node", cfg.NodeID)
	}

	if cfg.AuthToken != "test-token" {
		t.Errorf("AuthToken: got %v, want test-token", cfg.AuthToken)
	}

	if cfg.MaxConcurrentPods != 10 {
		t.Errorf("MaxConcurrentPods: got %v, want 10", cfg.MaxConcurrentPods)
	}
}

func TestConfigFromEnvironment(t *testing.T) {
	// Set environment variables
	os.Setenv("AGENTSMESH_SERVER_URL", "http://env.example.com")
	defer func() {
		os.Unsetenv("AGENTSMESH_SERVER_URL")
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.ServerURL != "http://env.example.com" {
		t.Errorf("ServerURL from env: got %v, want http://env.example.com", cfg.ServerURL)
	}
}

func TestConfigWorkspaceRootExpansion(t *testing.T) {
	os.Setenv("TEST_WORKSPACE", "/custom/workspace")
	defer os.Unsetenv("TEST_WORKSPACE")

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "runner.yaml")

	content := `
server_url: http://test.example.com
auth_token: test-token
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
		ServerURL:             "http://localhost",
		NodeID:                "test-node",
		Description:           "Test runner",
		AuthToken:             "token",
		RegistrationToken:     "reg-token",
		MaxConcurrentPods: 5,
		WorkspaceRoot:         "/workspace",
		GitConfigPath:         "/git/config",
		RepositoryPath:        "/repo",
		WorktreesDir:          "/worktrees",
		BaseBranch:            "main",
		MCPConfigPath:         "/mcp/config",
		DefaultAgent:          "claude-code",
		DefaultShell:          "/bin/bash",
		AgentEnvVars:          map[string]string{"KEY": "VALUE"},
		HealthCheckPort:       9090,
		LogLevel:              "debug",
		LogFile:               "/var/log/runner.log",
	}

	if cfg.ServerURL != "http://localhost" {
		t.Errorf("ServerURL: got %v, want http://localhost", cfg.ServerURL)
	}

	if cfg.NodeID != "test-node" {
		t.Errorf("NodeID: got %v, want test-node", cfg.NodeID)
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
