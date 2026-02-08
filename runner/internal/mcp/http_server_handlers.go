package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

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

	logger.MCP().Debug("Tool call received", "tool", params.Name, "pod_key", pod.PodKey)

	// Find tool
	var tool *MCPTool
	for _, t := range s.tools {
		if t.Name == params.Name {
			tool = t
			break
		}
	}

	if tool == nil {
		logger.MCP().Warn("Tool not found", "tool", params.Name)
		s.sendError(w, req.ID, -32602, "Tool not found", params.Name)
		return
	}

	// Execute tool
	ctx := context.Background()
	result, err := tool.Handler(ctx, pod.Client, params.Arguments)
	if err != nil {
		logger.MCP().Debug("Tool call failed", "tool", params.Name, "error", err)
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
