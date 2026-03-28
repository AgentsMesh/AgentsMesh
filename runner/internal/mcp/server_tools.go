package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// listTools retrieves available tools from the server
func (s *Server) listTools(ctx context.Context) error {
	resp, err := s.call(ctx, "tools/list", nil)
	if err != nil {
		return err
	}

	var result struct {
		Tools []Tool `json:"tools"`
	}

	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("failed to parse tools list: %w", err)
	}

	s.mu.Lock()
	for _, tool := range result.Tools {
		t := tool
		s.tools[tool.Name] = &t
	}
	s.mu.Unlock()

	return nil
}

// GetTools returns available tools
func (s *Server) GetTools() []*Tool {
	s.mu.Lock()
	defer s.mu.Unlock()

	tools := make([]*Tool, 0, len(s.tools))
	for _, t := range s.tools {
		tools = append(tools, t)
	}
	return tools
}

// CallTool calls an MCP tool
func (s *Server) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (json.RawMessage, error) {
	log := logger.MCP()
	params := map[string]interface{}{
		"name":      name,
		"arguments": arguments,
	}

	resp, err := s.call(ctx, "tools/call", params)
	if err != nil {
		log.Error("MCP tool call RPC failed", "server", s.name, "tool", name, "error", err)
		return nil, err
	}

	if resp.Error != nil {
		log.Error("MCP tool call returned error", "server", s.name, "tool", name, "error", resp.Error.Message)
		return nil, fmt.Errorf("tool call failed: %s", resp.Error.Message)
	}

	var result struct {
		Content []struct {
			Type string          `json:"type"`
			Text string          `json:"text,omitempty"`
			Data json.RawMessage `json:"data,omitempty"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}

	if err := json.Unmarshal(resp.Result, &result); err != nil {
		log.Error("Failed to parse MCP tool result", "server", s.name, "tool", name, "error", err)
		return nil, fmt.Errorf("failed to parse tool result: %w", err)
	}

	if result.IsError {
		if len(result.Content) > 0 && result.Content[0].Text != "" {
			log.Warn("MCP tool returned error content", "server", s.name, "tool", name, "error", result.Content[0].Text)
			return nil, fmt.Errorf("tool error: %s", result.Content[0].Text)
		}
		log.Warn("MCP tool returned error without details", "server", s.name, "tool", name)
		return nil, fmt.Errorf("tool returned error")
	}

	return resp.Result, nil
}
