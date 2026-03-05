package mcp

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// PodStatusProvider provides Pod status information.
// This interface allows HTTPServer to query Pod status from the Runner.
type PodStatusProvider interface {
	// GetPodStatus returns the agent status (executing/waiting/idle) for a given pod.
	GetPodStatus(podKey string) (agentStatus string, podStatus string, shellPid int, found bool)
}

// LocalTerminalProvider provides direct access to local terminal operations.
// This is used by AutopilotController control process to interact with local Pods
// without going through the Backend API.
type LocalTerminalProvider interface {
	// GetTerminalOutput returns the terminal output for a local pod.
	GetTerminalOutput(podKey string, lines int) (string, error)
	// SendTerminalText sends text to a local pod's terminal.
	SendTerminalText(podKey string, text string) error
	// SendTerminalKey sends special keys to a local pod's terminal.
	SendTerminalKey(podKey string, keys []string) error
}

// HTTPServer provides an MCP server over HTTP for agent collaboration.
// This server exposes collaboration tools to Claude Code via the MCP protocol.
type HTTPServer struct {
	rpcClient        *client.RPCClient
	port             int
	pods             map[string]*PodInfo
	mu               sync.RWMutex
	httpServer       *http.Server
	tools            []*MCPTool
	statusProvider   PodStatusProvider
	terminalProvider LocalTerminalProvider
}

// PodInfo holds information about a registered pod.
type PodInfo struct {
	PodKey       string
	OrgSlug      string
	TicketID     *int
	ProjectID    *int
	AgentType    string
	RegisteredAt time.Time
	Client       tools.CollaborationClient
}

// MCPTool represents a tool exposed via MCP.
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	Handler     MCPToolHandler
}

// MCPToolHandler is a function that handles tool invocations.
type MCPToolHandler func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error)

// NewHTTPServer creates a new MCP HTTP server.
// rpcClient is used for MCP tool calls over the gRPC bidirectional stream.
func NewHTTPServer(rpcClient *client.RPCClient, port int) *HTTPServer {
	log := logger.MCP()
	log.Debug("Creating MCP HTTP server", "port", port)

	server := &HTTPServer{
		rpcClient: rpcClient,
		port:      port,
		pods:      make(map[string]*PodInfo),
	}

	// Register all collaboration tools
	server.registerTools()
	log.Debug("MCP tools registered", "count", len(server.tools))

	return server
}

// Start starts the HTTP server.
func (s *HTTPServer) Start() error {
	mux := http.NewServeMux()

	// MCP endpoint
	mux.HandleFunc("/mcp", s.handleMCP)

	// Health check
	mux.HandleFunc("/health", s.handleHealth)

	// Debug: list pods
	mux.HandleFunc("/pods", s.handlePods)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("127.0.0.1:%d", s.port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log := logger.MCP()
	log.Info("Starting MCP HTTP server", "port", s.port)

	safego.Go("mcp-http-listen", func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Server error", "error", err)
		}
	})

	return nil
}

// Stop stops the HTTP server.
func (s *HTTPServer) Stop() error {
	log := logger.MCP()
	log.Info("Stopping MCP HTTP server")
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := s.httpServer.Shutdown(ctx)
		if err != nil {
			log.Error("Failed to stop MCP HTTP server", "error", err)
		} else {
			log.Info("MCP HTTP server stopped")
		}
		return err
	}
	return nil
}

// RegisterPod registers a pod with the MCP server.
func (s *HTTPServer) RegisterPod(podKey, orgSlug string, ticketID, projectID *int, agentType string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pods[podKey] = &PodInfo{
		PodKey:       podKey,
		OrgSlug:      orgSlug,
		TicketID:     ticketID,
		ProjectID:    projectID,
		AgentType:    agentType,
		RegisteredAt: time.Now(),
		Client:       NewGRPCCollaborationClient(s.rpcClient, podKey),
	}

	logger.MCP().Debug("Registered pod", "pod_key", podKey, "org", orgSlug)
}

// UnregisterPod removes a pod from the MCP server.
func (s *HTTPServer) UnregisterPod(podKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.pods, podKey)
	logger.MCP().Debug("Unregistered pod", "pod_key", podKey)
}

// GetPod returns pod info for a given pod key.
func (s *HTTPServer) GetPod(podKey string) (*PodInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	info, ok := s.pods[podKey]
	return info, ok
}

// SetStatusProvider sets the pod status provider for get_pod_status tool.
func (s *HTTPServer) SetStatusProvider(provider PodStatusProvider) {
	s.statusProvider = provider
}

// SetTerminalProvider sets the local terminal provider for terminal tools.
// This enables direct access to local pods without going through Backend API.
func (s *HTTPServer) SetTerminalProvider(provider LocalTerminalProvider) {
	s.terminalProvider = provider
}

// GetTerminalProvider returns the local terminal provider.
func (s *HTTPServer) GetTerminalProvider() LocalTerminalProvider {
	return s.terminalProvider
}

// PodCount returns the number of registered pods.
func (s *HTTPServer) PodCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.pods)
}

// Port returns the server port.
func (s *HTTPServer) Port() int {
	return s.port
}

// GenerateMCPConfig generates the MCP configuration JSON for Claude Code.
func (s *HTTPServer) GenerateMCPConfig(podKey string) map[string]interface{} {
	return map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"agentsmesh-collaboration": map[string]interface{}{
				"command": "curl",
				"args": []string{
					"-X", "POST",
					"-H", "Content-Type: application/json",
					"-H", fmt.Sprintf("X-Pod-Key: %s", podKey),
					fmt.Sprintf("http://localhost:%d/mcp", s.port),
					"-d", "@-",
				},
			},
		},
	}
}

// registerTools registers all collaboration tools.
func (s *HTTPServer) registerTools() {
	s.tools = []*MCPTool{
		// Terminal tools
		s.createObserveTerminalTool(),
		s.createSendTerminalTextTool(),
		s.createSendTerminalKeyTool(),
		s.createGetPodStatusTool(),

		// Discovery tools
		s.createListAvailablePodsTool(),
		s.createListRunnersTool(),
		s.createListRepositoriesTool(),

		// Binding tools
		s.createBindPodTool(),
		s.createAcceptBindingTool(),
		s.createRejectBindingTool(),
		s.createUnbindPodTool(),
		s.createGetBindingsTool(),
		s.createGetBoundPodsTool(),

		// Channel tools
		s.createSearchChannelsTool(),
		s.createCreateChannelTool(),
		s.createGetChannelTool(),
		s.createSendChannelMessageTool(),
		s.createGetChannelMessagesTool(),
		s.createGetChannelDocumentTool(),
		s.createUpdateChannelDocumentTool(),

		// Ticket tools
		s.createSearchTicketsTool(),
		s.createGetTicketTool(),
		s.createCreateTicketTool(),
		s.createUpdateTicketTool(),
		s.createPostCommentTool(),

		// Pod tools
		s.createCreatePodTool(),
	}
}
