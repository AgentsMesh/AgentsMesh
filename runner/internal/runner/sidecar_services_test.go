package runner

import (
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
)

func TestSidecarServicesMCPServer(t *testing.T) {
	cfg := &config.Config{WorkspaceRoot: t.TempDir()}
	mockConn := client.NewMockConnection()

	c := NewSidecarServices(cfg, mockConn)

	if c.MCPServer() == nil {
		t.Error("MCPServer should not be nil")
	}
}

func TestSidecarServicesAgentMonitor(t *testing.T) {
	cfg := &config.Config{WorkspaceRoot: t.TempDir()}
	mockConn := client.NewMockConnection()

	c := NewSidecarServices(cfg, mockConn)

	if c.AgentMonitor() == nil {
		t.Error("AgentMonitor should not be nil")
	}
}

func TestSidecarServicesNilSafe(t *testing.T) {
	var c *SidecarServices

	if c.MCPServer() != nil {
		t.Error("nil SidecarServices.MCPServer() should return nil")
	}
	if c.AgentMonitor() != nil {
		t.Error("nil SidecarServices.AgentMonitor() should return nil")
	}
	if svcs := c.Services(); len(svcs) != 0 {
		t.Errorf("nil SidecarServices.Services() should return empty, got %d", len(svcs))
	}
	// Should not panic
	c.SetProviders(nil, nil)
}

func TestSidecarServicesServices(t *testing.T) {
	cfg := &config.Config{WorkspaceRoot: t.TempDir()}
	mockConn := client.NewMockConnection()

	c := NewSidecarServices(cfg, mockConn)

	svcs := c.Services()
	// Should have 2 services: MCPServerService + MonitorService
	if len(svcs) != 2 {
		t.Errorf("Services() returned %d services, want 2", len(svcs))
	}
}

func TestSidecarServicesWithMCPConfig(t *testing.T) {
	cfg := &config.Config{
		WorkspaceRoot: t.TempDir(),
		MCPConfigPath: "/nonexistent/mcp.json", // should warn but not fail
	}
	mockConn := client.NewMockConnection()

	c := NewSidecarServices(cfg, mockConn)
	if c.MCPServer() == nil {
		t.Error("MCPServer should still be initialized even with bad config path")
	}
}
