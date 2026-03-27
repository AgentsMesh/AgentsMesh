package mcp

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// MCPRequest represents an MCP JSON-RPC request.
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// MCPResponse represents an MCP JSON-RPC response.
type MCPResponse struct {
	JSONRPC string       `json:"jsonrpc"`
	ID      interface{}  `json:"id"`
	Result  interface{}  `json:"result,omitempty"`
	Error   *MCPRPCError `json:"error,omitempty"`
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

// handleMCP handles MCP JSON-RPC requests.
func (s *HTTPServer) handleMCP(w http.ResponseWriter, r *http.Request) {
	log := logger.MCP()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get pod key from header
	podKey := r.Header.Get("X-Pod-Key")
	if podKey == "" {
		log.Warn("MCP request missing X-Pod-Key header")
		s.sendError(w, nil, -32600, "Missing X-Pod-Key header", nil)
		return
	}

	pod, ok := s.GetPod(podKey)
	if !ok {
		log.Warn("MCP request for unregistered pod", "pod_key", podKey)
		s.sendError(w, nil, -32600, "Pod not registered", nil)
		return
	}

	// Limit request body size to prevent OOM from oversized payloads (1MB)
	const maxMCPRequestSize = 1 * 1024 * 1024
	r.Body = http.MaxBytesReader(w, r.Body, maxMCPRequestSize)

	// Parse request
	var req MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("MCP request parse error", "pod_key", podKey, "error", err)
		s.sendError(w, nil, -32700, "Parse error", err.Error())
		return
	}

	log.Debug("MCP request received", "method", req.Method, "id", req.ID, "pod_key", podKey)

	// Handle MCP notifications (no "id" field, method starts with "notifications/").
	// Per the Streamable HTTP spec, notifications MUST receive 202 Accepted with no body.
	if strings.HasPrefix(req.Method, "notifications/") {
		log.Debug("MCP notification received", "method", req.Method, "pod_key", podKey)
		w.WriteHeader(http.StatusAccepted)
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
		log.Warn("MCP unknown method", "method", req.Method, "pod_key", podKey)
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
