package acp

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/processmgr"
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
	proc      processmgr.Handle
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

	// Thinking history for snapshots (accumulator parallel to message history,
	// so late subscribers see prior thinking blocks, not just future incremental
	// events).
	thinkings   []ThinkingUpdate
	thinkingsMu sync.RWMutex
	maxThinkings int

	// Log history for snapshots.
	logs    []LogEntry
	logsMu  sync.RWMutex
	maxLogs int

	// Current configuration (permission_mode, model) for snapshots and broadcast.
	// Writes go through applyConfiguration (callback-wrap) or SeedConfiguration (init).
	configuration Configuration
	configMu      sync.RWMutex

	// Pending permission requests for snapshots
	pendingPerms   []PermissionRequest
	pendingPermsMu sync.RWMutex

	// Lifecycle
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
	stopOnce sync.Once

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
		thinkings:    make([]ThinkingUpdate, 0, 64),
		maxThinkings: 200,
		logs:         make([]LogEntry, 0, 64),
		maxLogs:      200,
		toolCalls:    make(map[string]*ToolCallSnapshot),
		ctx:          ctx,
		cancel:       cancel,
		done:         make(chan struct{}),
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

	proc, err := processmgr.Global().Start(c.ctx, processmgr.Spec{
		Owner:       "acp:" + c.cfg.Command,
		Command:     c.cfg.Command,
		Args:        c.cfg.Args,
		Dir:         c.cfg.WorkDir,
		Env:         c.cfg.Env,
		Mode:        processmgr.ModeNormal,
		PipeStdin:   true,
		PipeStdout:  true,
		PipeStderr:  true,
		// 1 s preserves the prior client.Stop() escalation budget (5 s SIGTERM
		// → 2 s SIGKILL wait fit within the 7 s overall timeout); the previous
		// inline Stop used the same 5+2 split, but tests assume a 3 s budget,
		// so we tighten the SIGTERM window here.
		StopTimeout: 1 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("start process: %w", err)
	}
	c.proc = proc
	stdin := proc.StdinWriter()
	stdout := proc.StdoutReader()
	stderr := proc.StderrReader()

	// Build wrapped callbacks that keep internal state in sync.
	wrappedCallbacks := c.wrapCallbacks()

	// Create transport via registry (subpackages register themselves via init())
	c.transport = NewTransport(c.cfg.TransportType, wrappedCallbacks, c.logger)

	// Wire I/O before starting goroutines
	if err := c.transport.Initialize(c.ctx, stdin, stdout, stderr); err != nil {
		c.Stop()
		return fmt.Errorf("transport initialize: %w", err)
	}

	go c.readStderr(stderr)
	go c.transport.ReadLoop(c.ctx)
	go c.watchExit()

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
