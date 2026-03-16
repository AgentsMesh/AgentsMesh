//go:build windows

package poddaemon

import (
	"net"

	"github.com/Microsoft/go-winio"
)

// Listen creates a Windows named pipe listener.
func Listen(path string) (net.Listener, error) {
	return winio.ListenPipe(path, nil)
}

// Dial connects to a Windows named pipe.
func Dial(path string) (net.Conn, error) {
	return winio.DialPipe(path, nil)
}

// IPCPath returns the named pipe path for the given name.
// The dir parameter is ignored on Windows.
func IPCPath(_, name string) string {
	return `\\.\pipe\agentsmesh-` + name
}
