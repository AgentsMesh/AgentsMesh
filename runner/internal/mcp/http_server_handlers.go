package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
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

// handleToolsCall handles the tools/call request.
func (s *HTTPServer) handleToolsCall(w http.ResponseWriter, req *MCPRequest, pod *PodInfo) {
	log := logger.MCP()

	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		log.Warn("Tool call invalid params", "pod_key", pod.PodKey, "error", err)
		s.sendError(w, req.ID, -32602, "Invalid params", err.Error())
		return
	}

	log.Info("Tool call received", "tool", params.Name, "pod_key", pod.PodKey)
	log.Debug("Tool call arguments", "tool", params.Name, "pod_key", pod.PodKey, "args", params.Arguments)

	// Find tool
	var tool *MCPTool
	for _, t := range s.tools {
		if t.Name == params.Name {
			tool = t
			break
		}
	}

	if tool == nil {
		log.Warn("Tool not found", "tool", params.Name, "pod_key", pod.PodKey)
		s.sendError(w, req.ID, -32602, "Tool not found", params.Name)
		return
	}

	// Execute tool
	ctx := context.Background()
	start := time.Now()
	result, err := tool.Handler(ctx, pod.Client, params.Arguments)
	elapsed := time.Since(start)

	if err != nil {
		log.Error("Tool call failed",
			"tool", params.Name,
			"pod_key", pod.PodKey,
			"error", err,
			"duration", elapsed,
		)
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
	case tools.TextFormatter:
		text = v.FormatText()
	default:
		data, _ := json.MarshalIndent(result, "", "  ")
		text = string(data)
	}

	log.Info("Tool call succeeded",
		"tool", params.Name,
		"pod_key", pod.PodKey,
		"duration", elapsed,
		"result_len", len(text),
	)

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
