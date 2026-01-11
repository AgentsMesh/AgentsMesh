// Package mcp implements the Model Context Protocol (MCP) support for Runner.
// MCP allows AI agents to interact with external tools and services through a
// standardized interface.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

// Server represents an MCP server instance
type Server struct {
	name       string
	command    string
	args       []string
	env        map[string]string
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	mu         sync.Mutex
	requestID  int64
	pending    map[int64]chan *Response
	tools      map[string]*Tool
	resources  map[string]*Resource
	running    bool
}

// Config holds MCP server configuration
type Config struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

// Tool represents an MCP tool
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// Resource represents an MCP resource
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// Request represents an MCP request
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents an MCP response
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error represents an MCP error
type Error struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// NewServer creates a new MCP server instance
func NewServer(cfg *Config) *Server {
	return &Server{
		name:      cfg.Name,
		command:   cfg.Command,
		args:      cfg.Args,
		env:       cfg.Env,
		pending:   make(map[int64]chan *Response),
		tools:     make(map[string]*Tool),
		resources: make(map[string]*Resource),
	}
}

// Start starts the MCP server process
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()

	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}

	// Build command
	s.cmd = exec.CommandContext(ctx, s.command, s.args...)

	// Set environment
	s.cmd.Env = os.Environ()
	for k, v := range s.env {
		s.cmd.Env = append(s.cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Set up pipes
	var err error
	s.stdin, err = s.cmd.StdinPipe()
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	s.stdout, err = s.cmd.StdoutPipe()
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Start process
	if err := s.cmd.Start(); err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	s.running = true

	// Start reading responses
	go s.readResponses()

	// Release lock before initialize (which needs to acquire lock for RPC calls)
	s.mu.Unlock()

	// Initialize the server
	if err := s.initialize(ctx); err != nil {
		s.Stop()
		return fmt.Errorf("failed to initialize MCP server: %w", err)
	}

	return nil
}

// Stop stops the MCP server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false

	// Close stdin to signal server to exit
	if s.stdin != nil {
		s.stdin.Close()
	}

	// Kill process if still running
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
		s.cmd.Wait()
	}

	// Cancel all pending requests
	for _, ch := range s.pending {
		close(ch)
	}
	s.pending = make(map[int64]chan *Response)

	return nil
}

// initialize performs MCP initialization handshake
func (s *Server) initialize(ctx context.Context) error {
	params := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"roots": map[string]interface{}{
				"listChanged": true,
			},
		},
		"clientInfo": map[string]interface{}{
			"name":    "AgentsMesh Runner",
			"version": "1.0.0",
		},
	}

	resp, err := s.call(ctx, "initialize", params)
	if err != nil {
		return err
	}

	// Parse server capabilities
	var result struct {
		ProtocolVersion string `json:"protocolVersion"`
		Capabilities    struct {
			Tools struct {
				ListChanged bool `json:"listChanged"`
			} `json:"tools"`
			Resources struct {
				Subscribe   bool `json:"subscribe"`
				ListChanged bool `json:"listChanged"`
			} `json:"resources"`
		} `json:"capabilities"`
		ServerInfo struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
	}

	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("failed to parse initialize response: %w", err)
	}

	// Send initialized notification
	if err := s.notify("notifications/initialized", nil); err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	// List available tools
	if err := s.listTools(ctx); err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	// List available resources
	if err := s.listResources(ctx); err != nil {
		return fmt.Errorf("failed to list resources: %w", err)
	}

	return nil
}

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

// listResources retrieves available resources from the server
func (s *Server) listResources(ctx context.Context) error {
	resp, err := s.call(ctx, "resources/list", nil)
	if err != nil {
		return err
	}

	var result struct {
		Resources []Resource `json:"resources"`
	}

	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("failed to parse resources list: %w", err)
	}

	s.mu.Lock()
	for _, res := range result.Resources {
		r := res
		s.resources[res.URI] = &r
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

// GetResources returns available resources
func (s *Server) GetResources() []*Resource {
	s.mu.Lock()
	defer s.mu.Unlock()

	resources := make([]*Resource, 0, len(s.resources))
	for _, r := range s.resources {
		resources = append(resources, r)
	}
	return resources
}

// CallTool calls an MCP tool
func (s *Server) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (json.RawMessage, error) {
	params := map[string]interface{}{
		"name":      name,
		"arguments": arguments,
	}

	resp, err := s.call(ctx, "tools/call", params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
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
		return nil, fmt.Errorf("failed to parse tool result: %w", err)
	}

	if result.IsError {
		if len(result.Content) > 0 && result.Content[0].Text != "" {
			return nil, fmt.Errorf("tool error: %s", result.Content[0].Text)
		}
		return nil, fmt.Errorf("tool returned error")
	}

	return resp.Result, nil
}

// ReadResource reads an MCP resource
func (s *Server) ReadResource(ctx context.Context, uri string) ([]byte, string, error) {
	params := map[string]interface{}{
		"uri": uri,
	}

	resp, err := s.call(ctx, "resources/read", params)
	if err != nil {
		return nil, "", err
	}

	if resp.Error != nil {
		return nil, "", fmt.Errorf("resource read failed: %s", resp.Error.Message)
	}

	var result struct {
		Contents []struct {
			URI      string `json:"uri"`
			MimeType string `json:"mimeType,omitempty"`
			Text     string `json:"text,omitempty"`
			Blob     string `json:"blob,omitempty"`
		} `json:"contents"`
	}

	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, "", fmt.Errorf("failed to parse resource: %w", err)
	}

	if len(result.Contents) == 0 {
		return nil, "", fmt.Errorf("no content returned")
	}

	content := result.Contents[0]
	if content.Text != "" {
		return []byte(content.Text), content.MimeType, nil
	}

	// Handle blob (base64 encoded)
	// In a real implementation, you'd decode the base64
	return []byte(content.Blob), content.MimeType, nil
}

// call makes an RPC call and waits for response
func (s *Server) call(ctx context.Context, method string, params interface{}) (*Response, error) {
	s.mu.Lock()
	s.requestID++
	id := s.requestID
	respChan := make(chan *Response, 1)
	s.pending[id] = respChan
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
	}()

	// Build request
	var paramsJSON json.RawMessage
	if params != nil {
		var err error
		paramsJSON, err = json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
	}

	req := Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  paramsJSON,
	}

	// Send request
	if err := s.send(&req); err != nil {
		return nil, err
	}

	// Wait for response
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp, ok := <-respChan:
		if !ok {
			return nil, fmt.Errorf("server closed")
		}
		return resp, nil
	}
}

// notify sends a notification (no response expected)
func (s *Server) notify(method string, params interface{}) error {
	var paramsJSON json.RawMessage
	if params != nil {
		var err error
		paramsJSON, err = json.Marshal(params)
		if err != nil {
			return fmt.Errorf("failed to marshal params: %w", err)
		}
	}

	req := Request{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsJSON,
	}

	return s.send(&req)
}

// send writes a request to the server
func (s *Server) send(req *Request) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("server not running")
	}

	if s.stdin == nil {
		return fmt.Errorf("stdin not available")
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// MCP uses newline-delimited JSON
	data = append(data, '\n')

	if _, err := s.stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write request: %w", err)
	}

	return nil
}

// readResponses reads responses from the server
func (s *Server) readResponses() {
	decoder := json.NewDecoder(s.stdout)

	for {
		var resp Response
		if err := decoder.Decode(&resp); err != nil {
			if err == io.EOF {
				return
			}
			continue
		}

		// Route response to waiting caller
		s.mu.Lock()
		if ch, ok := s.pending[resp.ID]; ok {
			ch <- &resp
		}
		s.mu.Unlock()
	}
}

// IsRunning returns whether the server is running
func (s *Server) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// Name returns the server name
func (s *Server) Name() string {
	return s.name
}
