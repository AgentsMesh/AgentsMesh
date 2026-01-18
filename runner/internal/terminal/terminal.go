package terminal

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// Options for creating a new terminal (legacy API)
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

// Terminal represents a PTY terminal session (legacy API)
// For new code, prefer using Session and Manager
type Terminal struct {
	cmd      *exec.Cmd
	pty      *os.File
	mu       sync.Mutex
	closed   bool
	onOutput func([]byte)
	onExit   func(int)
}

// New creates a new terminal instance (legacy API)
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

	return &Terminal{
		cmd:      cmd,
		onOutput: opts.OnOutput,
		onExit:   opts.OnExit,
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
	log.Debug("Starting command", "path", t.cmd.Path, "args", t.cmd.Args, "dir", t.cmd.Dir)

	// Start with PTY
	ptmx, err := pty.Start(t.cmd)
	if err != nil {
		return fmt.Errorf("failed to start pty: %w", err)
	}
	t.pty = ptmx

	log.Debug("PTY started", "pid", t.cmd.Process.Pid)

	// Start output reader
	go t.readOutput()

	// Wait for process exit
	go t.waitExit()

	return nil
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

// Resize resizes the terminal
func (t *Terminal) Resize(rows, cols int) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed || t.pty == nil {
		return fmt.Errorf("terminal is not running")
	}

	return pty.Setsize(t.pty, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
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

// Stop stops the terminal
func (t *Terminal) Stop() {
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

// readOutput reads output from the PTY and sends to handler
func (t *Terminal) readOutput() {
	log := logger.Terminal()
	buf := make([]byte, 4096)
	readCount := 0
	for {
		n, err := t.pty.Read(buf)
		if err != nil {
			if err != io.EOF {
				// Only log if not a normal close
				t.mu.Lock()
				closed := t.closed
				t.mu.Unlock()
				if !closed {
					log.Error("PTY read error", "error", err, "read_count", readCount)
				}
			} else {
				log.Debug("PTY EOF received", "read_count", readCount)
			}
			break
		}

		readCount++
		if n > 0 {
			// Debug: Log first few reads
			if readCount <= 5 {
				preview := string(buf[:min(n, 100)])
				log.Debug("PTY read", "read_num", readCount, "bytes", n, "preview", preview)
			}

			// Make a copy of the data
			data := make([]byte, n)
			copy(data, buf[:n])

			// Get handler with lock to prevent race condition
			t.mu.Lock()
			handler := t.onOutput
			t.mu.Unlock()

			if handler != nil {
				handler(data)
			} else {
				log.Warn("No output handler set", "read_num", readCount)
			}
		}
	}
}

// waitExit waits for the process to exit
func (t *Terminal) waitExit() {
	exitCode := 0
	if err := t.cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	t.mu.Lock()
	t.closed = true
	t.mu.Unlock()

	if t.pty != nil {
		t.pty.Close()
	}

	// Get handler with lock to prevent race condition
	t.mu.Lock()
	handler := t.onExit
	t.mu.Unlock()

	if handler != nil {
		handler(exitCode)
	}
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
