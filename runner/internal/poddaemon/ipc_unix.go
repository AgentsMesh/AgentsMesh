//go:build !windows

package poddaemon

import (
	"net"
	"os"
	"path/filepath"
)

// Listen creates a Unix domain socket listener.
// Removes any stale socket file left by a previously crashed daemon.
func Listen(path string) (net.Listener, error) {
	// Clean up stale socket from a previous crash (ignore errors).
	_ = os.Remove(path)
	return net.Listen("unix", path)
}

// Dial connects to a Unix domain socket.
func Dial(path string) (net.Conn, error) {
	return net.Dial("unix", path)
}

// IPCPath returns the IPC socket path.
// socketDir is a guaranteed-short directory provided by config.
func IPCPath(socketDir, name string) string {
	return filepath.Join(socketDir, name+".sock")
}

// EnsureSocketDir creates the socket directory if needed.
func EnsureSocketDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}
