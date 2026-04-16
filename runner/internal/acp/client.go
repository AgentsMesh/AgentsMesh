package acp

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"sync"
)

// TransportType constants for ClientConfig.
// TransportTypeACP is the default protocol. Agent-specific transport types
// are defined in their respective packages under internal/agents/<name>/.
const (
	TransportTypeACP = "acp" // JSON-RPC 2.0 (Gemini, OpenCode, default)
)

// ClientConfig configures the ACP client.
type ClientConfig struct {
	Command       string
	Args          []string
	WorkDir       string
	Env           []string
	Logger        *slog.Logger
	Callbacks     EventCallbacks
	TransportType string // registered via acp.RegisterCommandMapping (default: "acp")
}

// ACPClient manages an agent subprocess communicating via a pluggable
// Transport (JSON-RPC 2.0 or Claude stream-json).
type ACPClient struct {
	cfg       ClientConfig
	cmd       *exec.Cmd
	transport Transport

	// State management
	state   string
	stateMu sync.RWMutex

	// Session tracking
	sessionID string
	sessionMu sync.RWMutex

	// Message history for snapshots
	messages    []ContentChunk
	messagesMu  sync.RWMutex
	maxMessages int

	// Tool call history for snapshots (keyed by tool_call_id)
	toolCalls   map[string]*ToolCallSnapshot
	toolCallsMu sync.RWMutex

	// Current plan for snapshots
	plan   []PlanStep
	planMu sync.RWMutex

	// Pending permission requests for snapshots
	pendingPerms   []PermissionRequest
	pendingPermsMu sync.RWMutex

	// Lifecycle
	ctx          context.Context
	cancel       context.CancelFunc
	done         chan struct{}
	waitExitDone chan struct{} // closed when waitExit() completes
	stopOnce     sync.Once

	logger *slog.Logger
}

// NewClient creates an unstarted ACP client with the given configuration.
func NewClient(cfg ClientConfig) *ACPClient {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.TransportType == "" {
		cfg.TransportType = TransportTypeACP
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &ACPClient{
		cfg:          cfg,
		state:        StateUninitialized,
		messages:     make([]ContentChunk, 0, 256),
		maxMessages:  1000,
		toolCalls:    make(map[string]*ToolCallSnapshot),
		ctx:          ctx,
		cancel:       cancel,
		done:         make(chan struct{}),
		waitExitDone: make(chan struct{}),
		logger:       cfg.Logger.With("component", "acp-client"),
	}
}

// State returns the current client state.
func (c *ACPClient) State() string {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.state
}

func (c *ACPClient) setState(state string) {
	c.stateMu.Lock()
	old := c.state
	c.state = state
	c.stateMu.Unlock()

	if old != state && c.cfg.Callbacks.OnStateChange != nil {
		c.cfg.Callbacks.OnStateChange(state)
	}
}

// SessionID returns the current session ID.
func (c *ACPClient) SessionID() string {
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	return c.sessionID
}

// Start launches the subprocess and performs the transport-specific handshake.
func (c *ACPClient) Start() error {
	c.setState(StateInitializing)

	c.cmd = exec.CommandContext(c.ctx, c.cfg.Command, c.cfg.Args...)
	c.cmd.Dir = c.cfg.WorkDir
	c.cmd.Env = c.cfg.Env

	stdin, err := c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := c.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	// Build wrapped callbacks that keep internal state in sync.
	wrappedCallbacks := c.cfg.Callbacks
	originalOnContent := wrappedCallbacks.OnContentChunk
	wrappedCallbacks.OnContentChunk = func(sessionID string, chunk ContentChunk) {
		c.addMessage(chunk)
		if originalOnContent != nil {
			originalOnContent(sessionID, chunk)
		}
	}
	originalOnToolCallUpdate := wrappedCallbacks.OnToolCallUpdate
	wrappedCallbacks.OnToolCallUpdate = func(sessionID string, update ToolCallUpdate) {
		c.upsertToolCall(update)
		if originalOnToolCallUpdate != nil {
			originalOnToolCallUpdate(sessionID, update)
		}
	}
	originalOnToolCallResult := wrappedCallbacks.OnToolCallResult
	wrappedCallbacks.OnToolCallResult = func(sessionID string, result ToolCallResult) {
		c.applyToolCallResult(result)
		if originalOnToolCallResult != nil {
			originalOnToolCallResult(sessionID, result)
		}
	}
	originalOnPlanUpdate := wrappedCallbacks.OnPlanUpdate
	wrappedCallbacks.OnPlanUpdate = func(sessionID string, update PlanUpdate) {
		c.setPlan(update.Steps)
		if originalOnPlanUpdate != nil {
			originalOnPlanUpdate(sessionID, update)
		}
	}
	// setState() already fires c.cfg.Callbacks.OnStateChange, so the wrapped
	// callback must NOT call originalOnStateChange again (would double-fire).
	wrappedCallbacks.OnStateChange = func(newState string) {
		c.setState(newState)
	}

	// Create transport via registry (subpackages register themselves via init())
	c.transport = NewTransport(c.cfg.TransportType, wrappedCallbacks, c.logger)

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("start process: %w", err)
	}

	// Wire I/O before starting goroutines
	if err := c.transport.Initialize(c.ctx, stdin, stdout, stderr); err != nil {
		c.Stop()
		return fmt.Errorf("transport initialize: %w", err)
	}

	go c.readStderr(stderr)
	go c.transport.ReadLoop(c.ctx)
	go c.waitExit()

	// Protocol handshake (must come after ReadLoop is running)
	sessionID, err := c.transport.Handshake(c.ctx)
	if err != nil {
		c.Stop()
		return fmt.Errorf("initialize: %w", err)
	}

	// Claude transport auto-discovers session ID during init
	if sessionID != "" {
		c.sessionMu.Lock()
		c.sessionID = sessionID
		c.sessionMu.Unlock()
	}

	c.setState(StateIdle)
	return nil
}
