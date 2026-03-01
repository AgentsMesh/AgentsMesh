//go:build linux

package lifecycle

import (
	"net"
	"os"

	"github.com/AgentsMesh/AgentsMesh/runner/internal/logger"
)

// notifySystemHealthy sends a WATCHDOG=1 notification to systemd via sd_notify.
// Pure Go implementation, no CGO required.
// If NOTIFY_SOCKET is not set (i.e., not running under systemd with WatchdogSec), this is a no-op.
func notifySystemHealthy() {
	addr := os.Getenv("NOTIFY_SOCKET")
	if addr == "" {
		return // Not running under systemd watchdog
	}

	conn, err := net.Dial("unixgram", addr)
	if err != nil {
		logger.Runner().Warn("Failed to connect to sd_notify socket", "error", err)
		return
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("WATCHDOG=1")); err != nil {
		logger.Runner().Warn("Failed to send sd_notify WATCHDOG=1", "error", err)
	}
}
