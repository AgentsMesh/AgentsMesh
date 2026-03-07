package terminal

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"

	"github.com/anthropics/agentsmesh/runner/internal/envfilter"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
)

const (
	// gracefulStopTimeout is the maximum time to wait for the process to exit
	// after sending SIGTERM before escalating to SIGKILL.
	gracefulStopTimeout = 5 * time.Second
)

// Options for creating a new terminal.
type Options struct {
	Command  string
	Args     []string
	WorkDir  string
	Env      map[string]string
	Rows     int
	Cols     int
	OnOutput func([]byte)
	OnExit   func(int)
}

// Terminal represents a PTY terminal session.
type Terminal struct {
	// Command configuration (set in New, consumed in Start)
	command string
	args    []string
	workDir string
	env     []string

	// PTY process handle (set in Start)
	proc ptyProcess

	mu       sync.Mutex
	closed   bool
	onOutput func([]byte)
	onExit   func(int)

	// onPTYError is called when readOutput encounters a fatal I/O error
	// (not timeout, not EOF, not normal close). This allows the runner to
	// send an error message to the frontend before the process is killed.
	onPTYError func(error)

	// Terminal size (set at creation, used when starting PTY)
	rows int
	cols int

	// Lifecycle synchronization
	doneCh       chan struct{} // Closed when process exits (signaled by waitExit)
	ptyCloseOnce sync.Once    // Ensures PTY file descriptor is closed exactly once

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
		command:  opts.Command,
		args:     opts.Args,
		workDir:  opts.WorkDir,
		env:      env,
		onOutput: opts.OnOutput,
		onExit:   opts.OnExit,
		rows:     rows,
		cols:     cols,
		doneCh:   make(chan struct{}),
		resumeCh: make(chan struct{}, 1), // Buffered to avoid blocking
	}, nil
}

// Start starts the terminal process
func (t *Terminal) Start() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return fmt.Errorf("terminal is closed")
	}

	log := logger.Terminal()
	log.Debug("Starting command", "command", t.command, "args", t.args, "dir", t.workDir, "cols", t.cols, "rows", t.rows)

	// Start with PTY and initial size (platform-specific)
	proc, err := startPTY(t.command, t.args, t.workDir, t.env, t.cols, t.rows)
	if err != nil {
		return fmt.Errorf("failed to start pty: %w", err)
	}
	t.proc = proc

	log.Debug("PTY started", "pid", t.proc.Pid(), "cols", t.cols, "rows", t.rows)

	// Start output reader
	safego.Go("pty-read", t.readOutput)

	// Wait for process exit
	safego.Go("pty-wait", t.waitExit)

	log.Info("Terminal started", "pid", t.proc.Pid(), "cols", t.cols, "rows", t.rows)

	return nil
}

// Stop stops the terminal with graceful shutdown.
// It sends a graceful stop signal first and waits up to gracefulStopTimeout
// for the process to exit. If the process doesn't exit in time, it is killed.
// This ensures AI agents (Claude Code, Aider, etc.) have time to perform cleanup
// operations like saving state and releasing git locks.
func (t *Terminal) Stop() {
	log := logger.Terminal()
	log.Info("Terminal stopping")

	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return
	}
	t.closed = true
	proc := t.proc
	t.mu.Unlock()

	if proc != nil {
		// Graceful shutdown: signal → wait → kill
		pid := proc.Pid()
		log.Debug("Sending graceful stop signal", "pid", pid)
		if err := proc.GracefulStop(); err != nil {
			log.Debug("Graceful stop failed (process may have already exited)", "error", err)
		}

		// Wait for process to exit or timeout
		select {
		case <-t.doneCh:
			log.Debug("Process exited gracefully")
		case <-time.After(gracefulStopTimeout):
			log.Warn("Process did not exit after graceful stop, killing",
				"pid", pid, "timeout", gracefulStopTimeout)
			if err := proc.Kill(); err != nil {
				log.Debug("Kill failed (process may have already exited)", "error", err)
			}
			// Wait briefly for waitExit to detect the kill
			select {
			case <-t.doneCh:
			case <-time.After(1 * time.Second):
				log.Warn("Process did not exit after kill", "pid", pid)
			}
		}
	}

	// Close PTY (safe to call concurrently via sync.Once)
	t.closePTY()

	log.Info("Terminal stopped")
}

// closePTY closes the PTY exactly once.
// Safe to call from multiple goroutines (Stop and waitExit).
func (t *Terminal) closePTY() {
	t.ptyCloseOnce.Do(func() {
		if t.proc != nil {
			t.proc.Close()
		}
	})
}

// PID returns the process ID
func (t *Terminal) PID() int {
	if t.proc != nil {
		return t.proc.Pid()
	}
	return 0
}

// IsClosed returns whether the terminal is closed.
func (t *Terminal) IsClosed() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.closed
}

// SetOutputHandler sets the output handler callback.
// Must be called before Start().
func (t *Terminal) SetOutputHandler(handler func([]byte)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onOutput = handler
}

// SetExitHandler sets the exit handler callback.
// Must be called before Start().
func (t *Terminal) SetExitHandler(handler func(int)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onExit = handler
}

// SetPTYErrorHandler sets the callback for fatal PTY read errors.
// When set, this is called when readOutput encounters a non-recoverable I/O error,
// giving the caller a chance to notify the frontend before the process is killed.
func (t *Terminal) SetPTYErrorHandler(handler func(error)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onPTYError = handler
}

// Write writes data to the terminal
func (t *Terminal) Write(data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed || t.proc == nil {
		return fmt.Errorf("terminal is not running")
	}

	_, err := t.proc.Write(data)
	return err
}

// IsRaw checks if terminal is in raw mode
func IsRaw(fd int) bool {
	return term.IsTerminal(fd)
}

// MakeRaw puts terminal in raw mode
func MakeRaw(fd int) (*term.State, error) {
	return term.MakeRaw(fd)
}

// Restore restores terminal state
func Restore(fd int, state *term.State) error {
	return term.Restore(fd, state)
}
