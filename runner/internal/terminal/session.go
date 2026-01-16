// Package terminal provides PTY management and terminal session services.
package terminal

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"

	"github.com/anthropics/agentsmesh/runner/internal/buffer"
)

// Session represents a PTY session.
type Session struct {
	ID           string
	PTY          *os.File
	Cmd          *exec.Cmd
	Size         *pty.Winsize
	CreatedAt    time.Time
	LastActivity time.Time

	// Output buffer for reconnection
	outputBuffer *buffer.Ring

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Channel to signal session closure
	done chan struct{}

	// Connected clients count
	clientCount int

	// Ensure Wait() is called only once to prevent race condition
	waitOnce sync.Once
	waitErr  error
	waitDone chan struct{}
}

// SessionConfig holds configuration for creating a new session.
type SessionConfig struct {
	ID           string
	Command      string
	Args         []string
	Env          []string
	WorkingDir   string
	Cols         uint16
	Rows         uint16
	BufferSize   int               // Size of output ring buffer for reconnection
	ExtraEnv     map[string]string // Extra environment variables (e.g., AI provider credentials)
}

// DefaultSessionConfig returns a SessionConfig with default values.
func DefaultSessionConfig(id string) *SessionConfig {
	// Use SHELL environment variable, fallback to /bin/sh
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	// Use -l flag to make it a login shell that loads .zshrc/.bashrc
	args := []string{"-l"}

	// Start with current environment
	env := os.Environ()

	// Add environment variables to disable interactive prompts
	// These prevent shell frameworks from blocking with update prompts
	env = append(env,
		"DISABLE_AUTO_UPDATE=true",          // oh-my-zsh: disable auto update
		"DISABLE_UPDATE_PROMPT=true",        // oh-my-zsh: disable update prompt
		"ZSH_DISABLE_COMPFIX=true",          // oh-my-zsh: disable compfix warnings
		"PYENV_VIRTUALENV_DISABLE_PROMPT=1", // pyenv: disable prompt
		"VIRTUAL_ENV_DISABLE_PROMPT=1",      // venv: disable prompt
	)

	return &SessionConfig{
		ID:         id,
		Command:    shell,
		Args:       args,
		Env:        env,
		WorkingDir: os.Getenv("HOME"),
		Cols:       80,
		Rows:       24,
		BufferSize: 64 * 1024, // 64KB buffer for scrollback
	}
}

// NewSession creates a new PTY session.
func NewSession(cfg *SessionConfig) (*Session, error) {
	cmd := exec.Command(cfg.Command, cfg.Args...)
	cmd.Env = cfg.Env
	cmd.Dir = cfg.WorkingDir

	// Merge extra environment variables (e.g., AI provider credentials)
	for key, value := range cfg.ExtraEnv {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Note: We don't set Setpgid here as it can conflict with PTY on macOS
	// The PTY library handles process group setup internally

	size := &pty.Winsize{
		Rows: cfg.Rows,
		Cols: cfg.Cols,
	}

	ptmx, err := pty.StartWithSize(cmd, size)
	if err != nil {
		return nil, fmt.Errorf("failed to start pty: %w", err)
	}

	session := &Session{
		ID:           cfg.ID,
		PTY:          ptmx,
		Cmd:          cmd,
		Size:         size,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		outputBuffer: buffer.NewRing(cfg.BufferSize),
		done:         make(chan struct{}),
		waitDone:     make(chan struct{}),
	}

	log.Printf("[terminal] Created new PTY session: pty_session_id=%s, command=%s, pid=%d",
		cfg.ID, cfg.Command, cmd.Process.Pid)

	return session, nil
}

// Resize changes the terminal size.
func (s *Session) Resize(rows, cols uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Size.Rows = rows
	s.Size.Cols = cols

	if err := pty.Setsize(s.PTY, s.Size); err != nil {
		return fmt.Errorf("failed to resize pty: %w", err)
	}

	log.Printf("[terminal] Resized PTY: pty_session_id=%s, rows=%d, cols=%d", s.ID, rows, cols)

	return nil
}

// Read reads from the PTY.
func (s *Session) Read(p []byte) (int, error) {
	n, err := s.PTY.Read(p)
	if n > 0 {
		s.mu.Lock()
		s.LastActivity = time.Now()
		s.outputBuffer.Write(p[:n])
		s.mu.Unlock()
	}
	return n, err
}

// Write writes to the PTY.
func (s *Session) Write(p []byte) (int, error) {
	s.mu.Lock()
	s.LastActivity = time.Now()
	s.mu.Unlock()
	return s.PTY.Write(p)
}

// GetScrollback returns the recent output buffer for reconnection.
func (s *Session) GetScrollback() []byte {
	return s.outputBuffer.Bytes()
}

// GetID returns the session ID.
func (s *Session) GetID() string {
	return s.ID
}

// Pid returns the process ID of the shell.
func (s *Session) Pid() int {
	if s.Cmd == nil || s.Cmd.Process == nil {
		return 0
	}
	return s.Cmd.Process.Pid
}

// GetSize returns the current terminal size.
func (s *Session) GetSize() (rows, cols uint16) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Size.Rows, s.Size.Cols
}

// IsRunning checks if the session process is still running.
func (s *Session) IsRunning() bool {
	if s.Cmd == nil || s.Cmd.Process == nil {
		return false
	}

	// Check if process exists
	err := s.Cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// AddClient increments the client count.
func (s *Session) AddClient() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clientCount++
	return s.clientCount
}

// RemoveClient decrements the client count.
func (s *Session) RemoveClient() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.clientCount > 0 {
		s.clientCount--
	}
	return s.clientCount
}

// ClientCount returns the current client count.
func (s *Session) ClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clientCount
}

// Done returns a channel that's closed when the session ends.
func (s *Session) Done() <-chan struct{} {
	return s.done
}

// WaitProcess waits for the process to exit.
// This method is safe to call multiple times from different goroutines.
// It ensures Wait() is only called once on the underlying exec.Cmd.
func (s *Session) WaitProcess() error {
	s.waitOnce.Do(func() {
		if s.Cmd != nil && s.Cmd.Process != nil {
			s.waitErr = s.Cmd.Wait()
		}
		close(s.waitDone)
	})
	<-s.waitDone
	return s.waitErr
}

// WaitDone returns a channel that's closed when the process has exited.
func (s *Session) WaitDone() <-chan struct{} {
	return s.waitDone
}

// Close terminates the session.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.done:
		// Already closed
		return nil
	default:
		close(s.done)
	}

	var errs []error

	// Close PTY first to signal EOF to process
	if s.PTY != nil {
		if err := s.PTY.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	// Kill the process
	if s.Cmd != nil && s.Cmd.Process != nil {
		// Try graceful termination first
		if err := s.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
			// Process may have already exited
			log.Printf("[terminal] SIGTERM failed for session %s, process may have exited", s.ID)
		}

		// Wait for process to exit with timeout using the thread-safe WaitProcess method
		select {
		case <-s.waitDone:
			// Process already exited (WaitProcess was called by monitorSession)
		case <-time.After(2 * time.Second):
			// Timeout, force kill
			log.Printf("[terminal] Process did not exit gracefully for session %s, force killing", s.ID)
			s.Cmd.Process.Kill()
			// Wait for WaitProcess to complete (will be called by monitorSession or this triggers it)
			s.WaitProcess()
		}
	}

	log.Printf("[terminal] Closed PTY session: pty_session_id=%s", s.ID)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing session: %v", errs)
	}
	return nil
}

// SessionInfoData holds session information for external use.
type SessionInfoData struct {
	ID           string    `json:"id"`
	Pid          int       `json:"pid"`
	Rows         uint16    `json:"rows"`
	Cols         uint16    `json:"cols"`
	CreatedAt    time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity"`
	IsRunning    bool      `json:"is_running"`
	ClientCount  int       `json:"client_count"`
}

// Info returns session information.
func (s *Session) Info() SessionInfoData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return SessionInfoData{
		ID:           s.ID,
		Pid:          s.Pid(),
		Rows:         s.Size.Rows,
		Cols:         s.Size.Cols,
		CreatedAt:    s.CreatedAt,
		LastActivity: s.LastActivity,
		IsRunning:    s.IsRunning(),
		ClientCount:  s.clientCount,
	}
}
