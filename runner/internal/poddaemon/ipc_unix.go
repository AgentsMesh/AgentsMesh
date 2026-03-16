//go:build !windows

package poddaemon

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"os"
	"path/filepath"
	"runtime"
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

// maxSocketPath is the max Unix socket path length.
// macOS: 104 bytes (sun_path in struct sockaddr_un), Linux: 108 bytes.
func maxSocketPath() int {
	if runtime.GOOS == "darwin" {
		return 104
	}
	return 108
}

// IPCPath returns the IPC socket path for the given directory and name.
// If the resulting path exceeds the Unix socket path length limit (104 on
// macOS, 108 on Linux), it falls back to a deterministic short path under
// os.TempDir() using a SHA-256 hash of the original path.
func IPCPath(dir, name string) string {
	candidate := filepath.Join(dir, name+".sock")
	if len(candidate) < maxSocketPath() {
		return candidate
	}
	// Hash the original path to produce a deterministic short name.
	h := sha256.Sum256([]byte(candidate))
	short := "am-" + hex.EncodeToString(h[:8]) + ".sock" // am-<16hex>.sock = 24 chars
	return filepath.Join(os.TempDir(), short)
}
