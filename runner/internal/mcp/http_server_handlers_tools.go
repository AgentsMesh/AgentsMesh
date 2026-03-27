package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

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
			"agent":         info.Agent,
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
