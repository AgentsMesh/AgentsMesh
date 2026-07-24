//go:build aix || darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd || solaris

package mockagent

import (
	"os"
	"os/signal"
	"syscall"
)

func watchTerminalResize() (<-chan os.Signal, func()) {
	resizeSignals := make(chan os.Signal, 8)
	signal.Notify(resizeSignals, syscall.SIGWINCH)
	return resizeSignals, func() { signal.Stop(resizeSignals) }
}
