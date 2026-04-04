//go:build integration

package runner

import (
	"log/slog"
	"os"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

// TestMain acts as a mock ACP agent when ACP_MOCK_AGENT=1,
// otherwise it runs the normal test suite.
// This mirrors the pattern from internal/acp/client_integration_mock_test.go.
func TestMain(m *testing.M) {
	if os.Getenv("ACP_MOCK_AGENT") == "1" {
		runMockACPAgent()
		return
	}
	os.Exit(m.Run())
}

// runMockACPAgent speaks JSON-RPC 2.0 over stdin/stdout.
// It handles initialize, session/new, and session/prompt.
func runMockACPAgent() {
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
			writer.WriteResponse(id, map[string]any{
				"protocol_version": "2025-01-01",
				"capabilities":     map[string]any{"permissions": true},
			}, nil)
		case "session/new":
			writer.WriteResponse(id, map[string]any{
				"sessionId": "mock-session-001",
			}, nil)
		case "session/prompt":
			writer.WriteNotification("session/update", map[string]any{
				"sessionId": "mock-session-001",
				"update": map[string]any{
					"sessionUpdate": "agent_message_chunk",
					"content":       map[string]any{"type": "text", "text": "Hello from runner mock"},
				},
			})
			writer.WriteResponse(id, map[string]any{"stopReason": "end_turn"}, nil)
		default:
			writer.WriteResponse(id, nil, &acp.JSONRPCError{
				Code: -32601, Message: "unknown method",
			})
		}
	}
}

// mockAgentCmd returns the test binary path for use as a mock agent.
func mockAgentCmd() string { return os.Args[0] }

// mockAgentArgs returns args that trigger the mock agent mode.
func mockAgentArgs() []string { return []string{"-test.run=TestMain"} }

// mockAgentEnv returns environment variables for the mock agent.
func mockAgentEnv() []string {
	return append(os.Environ(), "ACP_MOCK_AGENT=1")
}
