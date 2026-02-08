package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
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

	// Terminal size (set at creation, used when starting PTY)
	rows int
	cols int

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

	// Build environment
	env := os.Environ()

	// Ensure terminal supports colors (critical for CLI tools like claude, ls, etc.)
	env = append(env, "TERM=xterm-256color")
	env = append(env, "COLORTERM=truecolor")

	for k, v := range opts.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
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
	go t.readOutput()

	// Wait for process exit
	go t.waitExit()

	log.Info("Terminal started", "pid", t.cmd.Process.Pid, "cols", t.cols, "rows", t.rows)

	return nil
}

// Stop stops the terminal
func (t *Terminal) Stop() {
	log := logger.Terminal()
	log.Info("Terminal stopping")

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return
	}
	t.closed = true

	// Try graceful shutdown first
	if t.cmd != nil && t.cmd.Process != nil {
		t.cmd.Process.Signal(syscall.SIGTERM)
	}

	// Close PTY
	if t.pty != nil {
		t.pty.Close()
	}

	log.Info("Terminal stopped")
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
