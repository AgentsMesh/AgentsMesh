package poddaemon

import "io"

// daemonProcess abstracts the platform-specific PTY process inside the daemon.
// This avoids importing the terminal package (which would create circular deps).
type daemonProcess interface {
	io.ReadWriteCloser

	// Resize changes the terminal dimensions.
	Resize(cols, rows int) error

	// Pid returns the child process ID.
	Pid() int

	// Wait blocks until the process exits and returns the exit code.
	Wait() (int, error)

	// GracefulStop sends a graceful shutdown signal (SIGTERM on Unix, Ctrl+C on Windows).
	GracefulStop() error

	// Kill forcefully terminates the process.
	Kill() error
}
