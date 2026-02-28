package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"

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
	cmd      *exec.Cmd
	pty      *os.File
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

	// Build command
	cmd := exec.Command(opts.Command, opts.Args...)
	cmd.Dir = opts.WorkDir

	// Build environment with proper deduplication.
	// Using a map prevents duplicate keys (e.g., TERM appearing twice)
	// which can confuse some programs.
	envMap := make(map[string]string)
	for _, e := range os.Environ() {
		if idx := strings.Index(e, "="); idx >= 0 {
			envMap[e[:idx]] = e[idx+1:]
		}
	}
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
	cmd.Env = env

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
		cmd:      cmd,
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
	log.Debug("Starting command", "path", t.cmd.Path, "args", t.cmd.Args, "dir", t.cmd.Dir, "cols", t.cols, "rows", t.rows)

	// Start with PTY and initial size
	// Use StartWithSize to set correct terminal dimensions from the beginning
	// This is critical for TUI applications like Claude Code that render based on terminal size
	winSize := &pty.Winsize{
		Rows: uint16(t.rows),
		Cols: uint16(t.cols),
	}
	ptmx, err := pty.StartWithSize(t.cmd, winSize)
	if err != nil {
		return fmt.Errorf("failed to start pty: %w", err)
	}
	t.pty = ptmx

	log.Debug("PTY started", "pid", t.cmd.Process.Pid, "cols", t.cols, "rows", t.rows)

	// Start output reader
	safego.Go("pty-read", t.readOutput)

	// Wait for process exit
	safego.Go("pty-wait", t.waitExit)

	log.Info("Terminal started", "pid", t.cmd.Process.Pid, "cols", t.cols, "rows", t.rows)

	return nil
}

// Stop stops the terminal with graceful shutdown.
// It sends SIGTERM first and waits up to gracefulStopTimeout for the process to exit.
// If the process doesn't exit in time, SIGKILL is sent as a last resort.
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
	proc := t.cmd.Process // nil if not started
	t.mu.Unlock()

	if proc != nil {
		// Graceful shutdown: SIGTERM → wait → SIGKILL
		log.Debug("Sending SIGTERM for graceful shutdown", "pid", proc.Pid)
		if err := proc.Signal(syscall.SIGTERM); err != nil {
			log.Debug("SIGTERM failed (process may have already exited)", "error", err)
		}

		// Wait for process to exit or timeout
		select {
		case <-t.doneCh:
			log.Debug("Process exited gracefully after SIGTERM")
		case <-time.After(gracefulStopTimeout):
			log.Warn("Process did not exit after SIGTERM, sending SIGKILL",
				"pid", proc.Pid, "timeout", gracefulStopTimeout)
			if err := proc.Kill(); err != nil {
				log.Debug("SIGKILL failed (process may have already exited)", "error", err)
			}
			// Wait briefly for waitExit to detect the kill
			select {
			case <-t.doneCh:
			case <-time.After(1 * time.Second):
				log.Warn("Process did not exit after SIGKILL", "pid", proc.Pid)
			}
		}
	}

	// Close PTY (safe to call concurrently via sync.Once)
	t.closePTY()

	log.Info("Terminal stopped")
}

// closePTY closes the PTY file descriptor exactly once.
// Safe to call from multiple goroutines (Stop and waitExit).
func (t *Terminal) closePTY() {
	t.ptyCloseOnce.Do(func() {
		if t.pty != nil {
			t.pty.Close()
		}
	})
}

// PID returns the process ID
func (t *Terminal) PID() int {
	if t.cmd != nil && t.cmd.Process != nil {
		return t.cmd.Process.Pid
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

	if t.closed || t.pty == nil {
		return fmt.Errorf("terminal is not running")
	}

	_, err := t.pty.Write(data)
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
