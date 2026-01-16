package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// HTTPServer provides an MCP server over HTTP for agent collaboration.
// This server exposes collaboration tools to Claude Code via the MCP protocol.
type HTTPServer struct {
	backendURL string
	port       int
	pods       map[string]*PodInfo
	mu         sync.RWMutex
	httpServer *http.Server
	tools      []*MCPTool
}

// PodInfo holds information about a registered pod.
type PodInfo struct {
	PodKey       string
	OrgSlug      string
	TicketID     *int
	ProjectID    *int
	AgentType    string
	RegisteredAt time.Time
	Client       *BackendClient
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

// MCPRequest represents an MCP JSON-RPC request.
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// MCPResponse represents an MCP JSON-RPC response.
type MCPResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *MCPRPCError  `json:"error,omitempty"`
}

// MCPRPCError represents an MCP JSON-RPC error.
type MCPRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCPToolResult represents the result of a tool call.
type MCPToolResult struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

// MCPContent represents content in a tool result.
type MCPContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// NewHTTPServer creates a new MCP HTTP server.
func NewHTTPServer(backendURL string, port int) *HTTPServer {
	server := &HTTPServer{
		backendURL: backendURL,
		port:       port,
		pods:       make(map[string]*PodInfo),
	}

	// Register all collaboration tools
	server.registerTools()

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
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Printf("[mcp_http_server] Starting MCP HTTP server on port %d", s.port)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[mcp_http_server] Server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the HTTP server.
func (s *HTTPServer) Stop() error {
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
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
		Client:       NewBackendClient(s.backendURL, orgSlug, podKey),
	}

	log.Printf("[mcp_http_server] Registered pod: %s (org: %s)", podKey, orgSlug)
}

// UnregisterPod removes a pod from the MCP server.
func (s *HTTPServer) UnregisterPod(podKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.pods, podKey)
	log.Printf("[mcp_http_server] Unregistered pod: %s", podKey)
}

// GetPod returns pod info for a given pod key.
func (s *HTTPServer) GetPod(podKey string) (*PodInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	info, ok := s.pods[podKey]
	return info, ok
}

// handleMCP handles MCP JSON-RPC requests.
func (s *HTTPServer) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get pod key from header
	podKey := r.Header.Get("X-Pod-Key")
	if podKey == "" {
		s.sendError(w, nil, -32600, "Missing X-Pod-Key header", nil)
		return
	}

	pod, ok := s.GetPod(podKey)
	if !ok {
		s.sendError(w, nil, -32600, "Pod not registered", nil)
		return
	}

	// Parse request
	var req MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, nil, -32700, "Parse error", err.Error())
		return
	}

	// Route request
	switch req.Method {
	case "initialize":
		s.handleInitialize(w, &req)
	case "tools/list":
		s.handleToolsList(w, &req)
	case "tools/call":
		s.handleToolsCall(w, &req, pod)
	default:
		s.sendError(w, req.ID, -32601, "Method not found", nil)
	}
}

// handleInitialize handles the MCP initialize request.
func (s *HTTPServer) handleInitialize(w http.ResponseWriter, req *MCPRequest) {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": false,
			},
		},
		"serverInfo": map[string]interface{}{
			"name":    "AgentsMesh Collaboration Server",
			"version": "1.0.0",
		},
	}

	s.sendResult(w, req.ID, result)
}

// handleToolsList handles the tools/list request.
func (s *HTTPServer) handleToolsList(w http.ResponseWriter, req *MCPRequest) {
	toolsList := make([]map[string]interface{}, 0, len(s.tools))
	for _, tool := range s.tools {
		toolsList = append(toolsList, map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		})
	}

	s.sendResult(w, req.ID, map[string]interface{}{
		"tools": toolsList,
	})
}

// handleToolsCall handles the tools/call request.
func (s *HTTPServer) handleToolsCall(w http.ResponseWriter, req *MCPRequest, pod *PodInfo) {
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(w, req.ID, -32602, "Invalid params", err.Error())
		return
	}

	// Find tool
	var tool *MCPTool
	for _, t := range s.tools {
		if t.Name == params.Name {
			tool = t
			break
		}
	}

	if tool == nil {
		s.sendError(w, req.ID, -32602, "Tool not found", params.Name)
		return
	}

	// Execute tool
	ctx := context.Background()
	result, err := tool.Handler(ctx, pod.Client, params.Arguments)
	if err != nil {
		s.sendResult(w, req.ID, MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: err.Error()}},
			IsError: true,
		})
		return
	}

	// Format result
	var text string
	switch v := result.(type) {
	case string:
		text = v
	default:
		data, _ := json.MarshalIndent(result, "", "  ")
		text = string(data)
	}

	s.sendResult(w, req.ID, MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: text}},
	})
}

// handleHealth handles health check requests.
func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"pods":   s.PodCount(),
	})
}

// handlePods lists registered pods (debug endpoint).
func (s *HTTPServer) handlePods(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pods := make([]map[string]interface{}, 0, len(s.pods))
	for _, info := range s.pods {
		pods = append(pods, map[string]interface{}{
			"pod_key":       info.PodKey,
			"ticket_id":     info.TicketID,
			"project_id":    info.ProjectID,
			"agent_type":    info.AgentType,
			"registered_at": info.RegisteredAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pods": pods,
	})
}

// sendResult sends a successful JSON-RPC response.
func (s *HTTPServer) sendResult(w http.ResponseWriter, id interface{}, result interface{}) {
	resp := MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// sendError sends an error JSON-RPC response.
func (s *HTTPServer) sendError(w http.ResponseWriter, id interface{}, code int, message string, data interface{}) {
	resp := MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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

		// Pod tools
		s.createCreatePodTool(),
	}
}

// Helper to extract string from args
func getStringArg(args map[string]interface{}, key string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return ""
}

// Helper to extract int from args
func getIntArg(args map[string]interface{}, key string) int {
	switch v := args[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}

// Helper to extract bool from args
func getBoolArg(args map[string]interface{}, key string) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return false
}

// Helper to extract int pointer from args
func getIntPtrArg(args map[string]interface{}, key string) *int {
	switch v := args[key].(type) {
	case float64:
		i := int(v)
		return &i
	case int:
		return &v
	}
	return nil
}

// Helper to extract int64 pointer from args
func getInt64PtrArg(args map[string]interface{}, key string) *int64 {
	switch v := args[key].(type) {
	case float64:
		i := int64(v)
		return &i
	case int:
		i := int64(v)
		return &i
	case int64:
		return &v
	}
	return nil
}

// Helper to extract string slice from args
func getStringSliceArg(args map[string]interface{}, key string) []string {
	if v, ok := args[key].([]interface{}); ok {
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}
