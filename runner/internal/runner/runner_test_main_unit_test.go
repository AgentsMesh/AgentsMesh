//go:build !integration

package runner

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
	"github.com/anthropics/agentsmesh/runner/internal/poddaemon"
	"github.com/anthropics/agentsmesh/runner/internal/processmgr"
)

const acpWireTestAgentEnv = "_AGENTSMESH_ACP_WIRE_TEST_AGENT"

// TestMain mirrors the production runner's two re-exec entry points. Unit
// tests can therefore exercise PodDaemonManager through its real process and
// IPC boundary rather than replacing lifecycle behavior with a mock.
func TestMain(m *testing.M) {
	if os.Getenv(acpWireTestAgentEnv) == "1" {
		runACPWireTestAgent()
		return
	}
	if len(os.Args) > 1 && os.Args[1] == processmgr.LauncherSubcommand {
		processmgr.RunLauncher()
		return
	}
	if configPath := os.Getenv("_AGENTSMESH_POD_DAEMON"); configPath != "" {
		poddaemon.RunDaemon(configPath)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	processmgr.Init(ctx, processmgr.Options{})
	code := m.Run()
	cancel()
	os.Exit(code)
}

func runACPWireTestAgent() {
	reader := acp.NewReader(os.Stdin, slog.Default())
	writer := acp.NewWriter(os.Stdout)
	for {
		msg, err := reader.ReadMessage()
		if err != nil {
			return
		}
		if !msg.IsRequest() {
			continue
		}
		id, _ := msg.GetID()
		switch msg.Method {
		case "initialize":
			_ = writer.WriteResponse(id, map[string]any{
				"protocol_version": "2025-01-01",
				"capabilities":     map[string]any{},
			}, nil)
		case "session/new":
			_ = writer.WriteResponse(id, map[string]any{"sessionId": "wire-test-session"}, nil)
		default:
			_ = writer.WriteResponse(id, nil, &acp.JSONRPCError{Code: -32601, Message: "unknown method"})
		}
	}
}
