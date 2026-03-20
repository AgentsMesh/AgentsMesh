package runner

import (
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
)

func TestEnhancedComponentsMCPServer(t *testing.T) {
	cfg := &config.Config{WorkspaceRoot: t.TempDir()}
	mockConn := client.NewMockConnection()

	c := NewEnhancedComponents(cfg, mockConn)

	if c.MCPServer() == nil {
		t.Error("MCPServer should not be nil")
	}
}

func TestEnhancedComponentsAgentMonitor(t *testing.T) {
	cfg := &config.Config{WorkspaceRoot: t.TempDir()}
	mockConn := client.NewMockConnection()

	c := NewEnhancedComponents(cfg, mockConn)

	if c.AgentMonitor() == nil {
		t.Error("AgentMonitor should not be nil")
	}
}

func TestEnhancedComponentsNilSafe(t *testing.T) {
	var c *EnhancedComponents

	if c.MCPServer() != nil {
		t.Error("nil EnhancedComponents.MCPServer() should return nil")
	}
	if c.AgentMonitor() != nil {
		t.Error("nil EnhancedComponents.AgentMonitor() should return nil")
	}
	if svcs := c.Services(); len(svcs) != 0 {
		t.Errorf("nil EnhancedComponents.Services() should return empty, got %d", len(svcs))
	}
	// Should not panic
	c.SetProviders(nil, nil)
}

func TestEnhancedComponentsServices(t *testing.T) {
	cfg := &config.Config{WorkspaceRoot: t.TempDir()}
	mockConn := client.NewMockConnection()

	c := NewEnhancedComponents(cfg, mockConn)

	svcs := c.Services()
	// Should have 2 services: MCPServerService + MonitorService
	if len(svcs) != 2 {
		t.Errorf("Services() returned %d services, want 2", len(svcs))
	}
}

func TestEnhancedComponentsWithMCPConfig(t *testing.T) {
	cfg := &config.Config{
		WorkspaceRoot: t.TempDir(),
		MCPConfigPath: "/nonexistent/mcp.json", // should warn but not fail
	}
	mockConn := client.NewMockConnection()

	c := NewEnhancedComponents(cfg, mockConn)
	if c.MCPServer() == nil {
		t.Error("MCPServer should still be initialized even with bad config path")
	}
}
