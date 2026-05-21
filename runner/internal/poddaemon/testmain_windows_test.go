//go:build windows

package poddaemon

import (
	"context"
	"os"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/processmgr"
)

// TestMain wires the Windows unit-test process so startDaemon (which now
// flows through processmgr.Global().Start with ModeDaemon) finds an
// initialized manager. The integration suite has its own TestMain in
// helpers_integration_test.go; this file covers the unit suite, which the
// Windows CI job actually runs (TestStartDaemonWindows is Windows-only).
//
// Also re-enters as the __processmgr_launcher__ subprocess so ModeDaemon's
// double-fork can hand the daemon off cleanly.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == processmgr.LauncherSubcommand {
		processmgr.RunLauncher()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	processmgr.Init(ctx, processmgr.Options{})
	os.Exit(m.Run())
}
