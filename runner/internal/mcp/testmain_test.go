package mcp

import (
	"context"
	"os"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/processmgr"
)

// TestMain initializes processmgr.Global() for every test in this package.
// Without it, server_lifecycle.go's call to processmgr.Global().Start would
// hit the uninitializedManager and fail every Start.
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
