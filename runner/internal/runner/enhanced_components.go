package runner

import (
	"time"

	"github.com/thejerf/suture/v4"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/lifecycle"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/mcp"
	"github.com/anthropics/agentsmesh/runner/internal/monitor"
)

// EnhancedComponents encapsulates MCP + Monitor creation and lifecycle management.
// Extracted from Runner to satisfy SRP — these optional components have their own
// initialization and lifecycle concerns.
type EnhancedComponents struct {
	mcpManager   *mcp.Manager
	mcpServer    *mcp.HTTPServer
	agentMonitor *monitor.Monitor
}

// NewEnhancedComponents initializes optional enhanced components based on config.
func NewEnhancedComponents(cfg *config.Config, conn client.Connection) *EnhancedComponents {
	log := logger.Runner()
	log.Debug("Initializing enhanced components")

	c := &EnhancedComponents{}

	// Initialize MCP manager
	c.mcpManager = mcp.NewManager()
	if cfg.MCPConfigPath != "" {
		if err := c.mcpManager.LoadConfig(cfg.MCPConfigPath); err != nil {
			log.Warn("Failed to load MCP config", "error", err)
		} else {
			log.Debug("MCP config loaded", "path", cfg.MCPConfigPath)
		}
	}

	// Initialize RPCClient for MCP over gRPC
	rpcClient := client.NewRPCClient(conn)
	if grpcConn, ok := conn.(*client.GRPCConnection); ok {
		grpcConn.SetRPCClient(rpcClient)
	}

	// Initialize MCP HTTP Server (started by Supervisor in Run())
	c.mcpServer = mcp.NewHTTPServer(rpcClient, cfg.GetMCPPort())

	// Initialize Monitor (started by Supervisor in Run())
	c.agentMonitor = monitor.NewMonitor(5 * time.Second)

	log.Debug("Enhanced components initialized")
	return c
}

// MCPServer returns the MCP server (nil-safe).
func (c *EnhancedComponents) MCPServer() MCPServer {
	if c == nil || c.mcpServer == nil {
		return nil
	}
	return c.mcpServer
}

// AgentMonitor returns the agent monitor (nil-safe).
func (c *EnhancedComponents) AgentMonitor() AgentMonitor {
	if c == nil || c.agentMonitor == nil {
		return nil
	}
	return c.agentMonitor
}

// Services returns suture services for Supervisor registration.
func (c *EnhancedComponents) Services() []suture.Service {
	if c == nil {
		return nil
	}
	var svcs []suture.Service
	if c.mcpServer != nil {
		svcs = append(svcs, &lifecycle.MCPServerService{Server: c.mcpServer})
	}
	if c.agentMonitor != nil {
		svcs = append(svcs, &lifecycle.MonitorService{Monitor: c.agentMonitor})
	}
	return svcs
}

// SetProviders wires status/pod providers into the MCP server.
func (c *EnhancedComponents) SetProviders(status mcp.PodStatusProvider, pod mcp.LocalPodProvider) {
	if c == nil || c.mcpServer == nil {
		return
	}
	c.mcpServer.SetStatusProvider(status)
	c.mcpServer.SetPodProvider(pod)
}
