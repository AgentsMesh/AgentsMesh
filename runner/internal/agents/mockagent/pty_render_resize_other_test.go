//go:build !(aix || darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd || solaris)

package mockagent

import "testing"

func TestWatchTerminalResizeLifecycle(t *testing.T) {
	signals, stop := watchTerminalResize()
	if signals != nil {
		t.Fatal("non-Unix resize watcher returned a signal channel")
	}
	if stop == nil {
		t.Fatal("non-Unix resize watcher returned nil stop function")
	}
	stop()
}
