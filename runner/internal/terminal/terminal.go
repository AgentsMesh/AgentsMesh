package terminal

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/envfilter"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
)

const (
	// gracefulStopTimeout is the maximum time to wait for the process to exit
	// after sending SIGTERM before escalating to SIGKILL.
	gracefulStopTimeout = 5 * time.Second
)

// PTYFactory is a function that creates a PtyProcess.
// When set in Options, it replaces the default platform-specific startPTY.
// This enables dependency injection for Pod Daemon mode.
type PTYFactory func(command string, args []string, workDir string, env []string, cols, rows int) (PtyProcess, error)

// Options for creating a new terminal.
type Options struct {
	Command  string
	Args     []string
	WorkDir  string
	Env      map[string]string
	Rows     int
	Cols     int
	Label    string // Identifier for log correlation (e.g., pod_key)
	OnOutput func([]byte)
	OnExit   func(int)

	// PTYFactory overrides the default platform PTY creation.
	// Used by Pod Daemon to inject daemonPTY instead of direct PTY.
	PTYFactory PTYFactory
}

// Terminal represents a PTY terminal session.
type Terminal struct {
	// Command configuration (set in New, consumed in Start)
	command string
	args    []string
	workDir string
	env     []string
	label   string // Identifier for log correlation (e.g., pod_key)

	// PTY process handle (set in Start)
	proc ptyProcess

	// Custom PTY factory (nil = use default platform startPTY)
	ptyFactory PTYFactory

	mu          sync.Mutex
	lifecycleMu sync.Mutex
	closed      bool
	stopping    bool
	onOutput    func([]byte)
	onExit      func(int)
	// readOrder is injected before Start and remains immutable while running.
	// It orders each PTY Read + onOutput callback against external resize/teardown.
	readOrder sync.Locker

	// onPTYError is called when readOutput encounters a fatal I/O error
	// (not timeout, not EOF, not normal close). This allows the runner to
	// send an error message to the frontend before the process is killed.
	onPTYError func(error)

	// Terminal size (set at creation, used when starting PTY)
	rows int
	cols int

	// Lifecycle synchronization
	doneCh         chan struct{} // Closed when process exits (signaled by waitExit)
	ptyCloseOnce   sync.Once     // Ensures PTY file descriptor is closed exactly once
	readDoneCh     chan struct{} // Closed after readOutput and its last callback exit
	readDoneOnce   sync.Once
	readActiveCh   chan struct{} // Closed when the reader starts its first PTY read
	readActiveOnce sync.Once
	readProgressCh chan struct{} // Edge-triggered notification after a dispatched read
	readProgress   uint64
	readStarted    bool
	processExited  bool

	// Backpressure control (ttyd-style flow control)
	// When paused, readOutput() blocks to prevent unbounded memory growth
	readPaused  bool          // Whether PTY reading is paused
	readPauseMu sync.RWMutex  // Protects readPaused flag
	resumeCh    chan struct{} // Signal to resume reading
}

// New creates a new terminal instance.
func New(opts Options) (*Terminal, error) {
	if opts.Command == "" {
		return nil, fmt.Errorf("command is required")
	}

	// Build environment with proper deduplication.
	// Using a map prevents duplicate keys (e.g., TERM appearing twice)
	// which can confuse some programs.
	// Filter Runner-internal vars to prevent leakage to child processes.
	envMap := make(map[string]string)
	for _, e := range envfilter.FilterEnv(os.Environ()) {
		if idx := strings.Index(e, "="); idx >= 0 {
			envMap[e[:idx]] = e[idx+1:]
		}
	}
	// Remove CLAUDECODE to prevent nested session detection when running
	// Claude Code inside a pod - the runner intentionally spawns claude sessions.
	delete(envMap, "CLAUDECODE")
	// Ensure terminal supports colors (critical for CLI tools like claude, ls, etc.)
	envMap["TERM"] = "xterm-256color"
	envMap["COLORTERM"] = "truecolor"
	// Apply user-specified env vars (highest priority)
	for k, v := range opts.Env {
		envMap[k] = v
	}
	env := make([]string, 0, len(envMap))
	for k, v := range envMap {
		env = append(env, k+"="+v)
	}

	// Default terminal size if not specified
	rows := opts.Rows
	cols := opts.Cols
	if rows <= 0 {
		rows = 24
	}
	if cols <= 0 {
		cols = 80
	}

	logger.Terminal().Debug("Terminal instance created",
		"command", opts.Command,
		"work_dir", opts.WorkDir,
		"cols", cols,
		"rows", rows)

	return &Terminal{
		command:        opts.Command,
		args:           opts.Args,
		workDir:        opts.WorkDir,
		env:            env,
		label:          opts.Label,
		ptyFactory:     opts.PTYFactory,
		onOutput:       opts.OnOutput,
		onExit:         opts.OnExit,
		rows:           rows,
		cols:           cols,
		doneCh:         make(chan struct{}),
		readDoneCh:     make(chan struct{}),
		readActiveCh:   make(chan struct{}),
		readProgressCh: make(chan struct{}, 1),
		resumeCh:       make(chan struct{}, 1), // Buffered to avoid blocking
	}, nil
}

// Start starts the terminal process
func (t *Terminal) Start() error {
	t.lifecycleMu.Lock()
	defer t.lifecycleMu.Unlock()
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed || t.stopping {
		return fmt.Errorf("terminal is closed")
	}
	if t.proc != nil || t.readStarted {
		return fmt.Errorf("terminal is already started")
	}

	log := logger.Terminal()
	log.Debug("Starting command", "command", t.command, "args", t.args, "dir", t.workDir, "cols", t.cols, "rows", t.rows)

	// Start with PTY and initial size (custom factory or platform-specific default)
	var proc ptyProcess
	var err error
	if t.ptyFactory != nil {
		proc, err = t.ptyFactory(t.command, t.args, t.workDir, t.env, t.cols, t.rows)
	} else {
		proc, err = startPTY(t.command, t.args, t.workDir, t.env, t.cols, t.rows)
	}
	if err != nil {
		return fmt.Errorf("failed to start pty: %w", err)
	}
	t.proc = proc
	t.readStarted = true

	log.Debug("PTY started", "pid", t.proc.Pid(), "cols", t.cols, "rows", t.rows)

	// Start output reader
	safego.Go("pty-read", t.readOutput)

	// Wait for process exit
	safego.Go("pty-wait", t.waitExit)

	log.Info("Terminal started", "pid", t.proc.Pid(), "cols", t.cols, "rows", t.rows)

	return nil
}
