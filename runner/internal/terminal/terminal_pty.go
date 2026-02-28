package terminal

import (
	"io"
	"time"
)

// ptyProcess abstracts platform-specific PTY operations.
// Unix uses creack/pty + exec.Cmd, Windows uses ConPTY.
type ptyProcess interface {
	io.ReadWriteCloser

	// Resize changes the terminal size (cols=width, rows=height).
	Resize(cols, rows int) error

	// GetSize returns the current terminal size (cols, rows).
	// Returns (0, 0, ErrUnsupported) if not supported on the platform.
	GetSize() (cols, rows int, err error)

	// Pid returns the process ID.
	Pid() int

	// SetReadDeadline sets a deadline for Read operations.
	// After the deadline, Read returns os.ErrDeadlineExceeded.
	SetReadDeadline(t time.Time) error

	// Wait blocks until the process exits and returns the exit code.
	Wait() (exitCode int, err error)

	// Kill forcefully terminates the process.
	Kill() error

	// GracefulStop sends a signal for graceful shutdown.
	// On Unix, this sends SIGTERM. On Windows, this sends Ctrl+C.
	GracefulStop() error
}
