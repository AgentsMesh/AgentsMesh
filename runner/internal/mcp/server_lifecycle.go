package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/envfilter"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/processmgr"
)

// Server represents an MCP server instance
type Server struct {
	name       string
	command    string
	args       []string
	env        map[string]string
	proc       processmgr.Handle
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

	env := envfilter.FilterEnv(os.Environ())
	for k, v := range s.env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	proc, err := processmgr.Global().Start(ctx, processmgr.Spec{
		Owner:      "mcp:" + s.name,
		Command:    s.command,
		Args:       s.args,
		Env:        env,
		Mode:       processmgr.ModeNormal,
		PipeStdin:  true,
		PipeStdout: true,
	})
	if err != nil {
		s.mu.Unlock()
		log.Error("Failed to start MCP server process", "name", s.name, "command", s.command, "error", err)
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	s.proc = proc
	s.stdin = proc.StdinWriter()
	s.stdout = proc.StdoutReader()
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

// Stop stops the MCP server. processmgr.Handle.Stop owns the SIGTERM →
// SIGKILL escalation and the reapLoop; this function only closes the JSON-RPC
// pipes (which unblocks readResponses) and drains pending request channels.
func (s *Server) Stop() error {
	s.mu.Lock()

	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false

	// Close stdin to signal server to exit.
	if s.stdin != nil {
		_ = s.stdin.Close()
	}
	// Close stdout BEFORE proc.Stop so readResponses' decoder.Decode unblocks
	// and the reapLoop inside processmgr can complete cmd.Wait — otherwise
	// pipe readers hold cmd.Wait open.
	if s.stdout != nil {
		_ = s.stdout.Close()
	}

	proc := s.proc
	s.mu.Unlock()

	if proc != nil {
		stopCtx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
		defer cancel()
		if err := proc.Stop(stopCtx); err != nil {
			logger.MCP().Warn("MCP server stop reported error", "name", s.name, "err", err)
		}
	}

	// Wait for readResponses goroutine to exit before draining pending
	// channels — this prevents send-on-closed-channel.
	s.readerDone.Wait()

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
