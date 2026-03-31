package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/anthropics/agentsmesh/runner/internal/envfilter"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/process"
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
	readerDone sync.WaitGroup
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
	log := logger.MCP()

	// Pre-check: verify the command exists before acquiring the lock.
	// On Windows, exec.CommandContext may fail with a cryptic error if the
	// binary is not on PATH. LookPath gives a clear "not found" message.
	if _, err := exec.LookPath(s.command); err != nil {
		log.Error("MCP server command not found", "name", s.name, "command", s.command, "error", err)
		return fmt.Errorf("MCP server command not found: %s: %w", s.command, err)
	}

	s.mu.Lock()

	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}

	// Build command
	s.cmd = exec.CommandContext(ctx, s.command, s.args...)

	// Set environment — filter Runner-internal vars to prevent leakage
	s.cmd.Env = envfilter.FilterEnv(os.Environ())
	for k, v := range s.env {
		s.cmd.Env = append(s.cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Set up pipes
	var err error
	s.stdin, err = s.cmd.StdinPipe()
	if err != nil {
		s.mu.Unlock()
		log.Error("Failed to create stdin pipe", "name", s.name, "error", err)
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	s.stdout, err = s.cmd.StdoutPipe()
	if err != nil {
		s.mu.Unlock()
		log.Error("Failed to create stdout pipe", "name", s.name, "error", err)
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Start process
	if err := s.cmd.Start(); err != nil {
		s.mu.Unlock()
		log.Error("Failed to start MCP server process", "name", s.name, "command", s.command, "error", err)
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	s.running = true

	// Start reading responses
	s.readerDone.Add(1)
	go s.readResponses()

	// Release lock before initialize (which needs to acquire lock for RPC calls)
	s.mu.Unlock()

	// Initialize the server
	if err := s.initialize(ctx); err != nil {
		s.Stop()
		log.Error("Failed to initialize MCP server", "name", s.name, "error", err)
		return fmt.Errorf("failed to initialize MCP server: %w", err)
	}

	log.Info("MCP server started and initialized", "name", s.name)
	return nil
}

// Stop stops the MCP server
func (s *Server) Stop() error {
	s.mu.Lock()

	if !s.running {
		s.mu.Unlock()
		return nil
	}

	s.running = false

	// Close stdin to signal server to exit
	if s.stdin != nil {
		s.stdin.Close()
	}

	// Close stdout BEFORE cmd.Wait() to unblock readResponses goroutine.
	// Go's cmd.Wait() waits for all StdoutPipe readers to finish, so if
	// readResponses is still blocking on decoder.Decode, Wait() deadlocks.
	if s.stdout != nil {
		s.stdout.Close()
	}

	// Kill process tree if still running.
	// On Windows, child processes are NOT killed when the parent dies,
	// so we must walk the tree and kill each descendant.
	if s.cmd != nil && s.cmd.Process != nil {
		_ = process.KillProcessTree(s.cmd.Process.Pid)
		s.cmd.Wait()
	}

	// Release the lock and wait for readResponses goroutine to exit
	// before closing pending channels — this prevents send-on-closed-channel.
	s.mu.Unlock()
	s.readerDone.Wait()

	// Now safe to close pending channels — readResponses has exited
	s.mu.Lock()
	for _, ch := range s.pending {
		close(ch)
	}
	s.pending = make(map[int64]chan *Response)
	s.mu.Unlock()

	return nil
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
