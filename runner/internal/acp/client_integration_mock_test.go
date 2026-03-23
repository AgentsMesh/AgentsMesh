package acp

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"testing"
	"time"
)

// Mock agent modes (selected via ACP_MOCK_MODE env var):
//   - "" (default): standard mock agent
//   - "send_request": after initialize, agent sends a JSON-RPC request to the client
//   - "error_init": responds to initialize with an error
//   - "slow_init": delays 5 seconds before responding to initialize
//   - "bad_json_init": responds with unparseable result to session/new
const (
	mockModeDefault    = ""
	mockModeSendReq    = "send_request"
	mockModeErrorInit  = "error_init"
	mockModeSlowInit   = "slow_init"
	mockModeBadJSONNew = "bad_json_new"
)

// TestMain implements the mock ACP agent subprocess.
// When invoked with ACP_MOCK_AGENT=1, this test binary acts as a minimal
// ACP-compatible agent that speaks JSON-RPC over stdin/stdout.
func TestMain(m *testing.M) {
	if os.Getenv("ACP_MOCK_AGENT") == "1" {
		runMockAgent()
		return
	}
	os.Exit(m.Run())
}

// runMockAgent is a minimal ACP agent that handles:
//   - initialize -> responds with capabilities
//   - session/new -> responds with session_id
//   - session/prompt -> sends notifications then responds
//   - permission/response (notification) -> no response needed
//   - session/cancel (notification) -> no response needed
//
// The behavior is influenced by the ACP_MOCK_MODE env var.
func runMockAgent() {
	mode := os.Getenv("ACP_MOCK_MODE")
	reader := NewReader(os.Stdin, slog.Default())
	writer := NewWriter(os.Stdout)

	for {
		msg, err := reader.ReadMessage()
		if err != nil {
			return // EOF or error -> exit
		}

		switch {
		case msg.IsRequest():
			id, _ := msg.GetID()
			switch msg.Method {
			case "initialize":
				switch mode {
				case mockModeErrorInit:
					writer.WriteResponse(id, nil, &JSONRPCError{
						Code:    ErrCodeInternal,
						Message: "mock init error",
					})
				case mockModeSlowInit:
					time.Sleep(5 * time.Second)
					writer.WriteResponse(id, map[string]any{
						"protocol_version": "2025-01-01",
					}, nil)
				default:
					writer.WriteResponse(id, map[string]any{
						"protocol_version": "2025-01-01",
						"capabilities":    map[string]any{"permissions": true},
					}, nil)
					// In send_request mode, send a request to the client after init
					if mode == mockModeSendReq {
						writer.WriteRequest("agent/custom_method", map[string]any{
							"info": "agent-initiated request",
						})
					}
				}

			case "session/new":
				if mode == mockModeBadJSONNew {
					// Write a raw response with un-parseable result
					raw := fmt.Sprintf(
						`{"jsonrpc":"2.0","id":%d,"result":"not_an_object"}`, id)
					os.Stdout.Write([]byte(raw + "\n"))
				} else {
					writer.WriteResponse(id, map[string]any{
						"session_id": "mock-session-001",
					}, nil)
				}

			case "session/prompt":
				// Send a content notification before responding
				writer.WriteNotification("session/update", map[string]any{
					"session_id": "mock-session-001",
					"type":       "content",
					"data":       map[string]any{"role": "assistant", "text": "Hello from mock agent"},
				})
				// Send session complete as a separate notification method
				writer.WriteNotification("session/complete", map[string]any{
					"session_id": "mock-session-001",
				})
				// Respond to the RPC
				writer.WriteResponse(id, map[string]any{"status": "completed"}, nil)

			default:
				writer.WriteResponse(id, nil, &JSONRPCError{
					Code:    ErrCodeMethodNotFound,
					Message: fmt.Sprintf("unknown method: %s", msg.Method),
				})
			}

		case msg.IsNotification():
			// Notifications don't require a response
		}
	}
}

// mockAgentCmd returns an exec.Cmd that runs this test binary as a mock ACP agent.
func mockAgentCmd() string {
	return os.Args[0]
}

// mockAgentArgs returns args that trigger the mock agent mode.
func mockAgentArgs() []string {
	return []string{"-test.run=TestMain"}
}

// mockAgentEnv returns environment variables for the mock agent (default mode).
func mockAgentEnv() []string {
	return append(os.Environ(), "ACP_MOCK_AGENT=1")
}

// mockAgentEnvWithMode returns environment variables for the mock agent with
// a specific mode override.
func mockAgentEnvWithMode(mode string) []string {
	return append(os.Environ(), "ACP_MOCK_AGENT=1", "ACP_MOCK_MODE="+mode)
}

// startMockClient creates and starts a mock ACP client for testing.
func startMockClient(t *testing.T) *ACPClient {
	t.Helper()
	client := NewClient(ClientConfig{
		Command: mockAgentCmd(),
		Args:    mockAgentArgs(),
		Env:     mockAgentEnv(),
		Logger:  slog.Default(),
	})
	if err := client.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	return client
}

// Ensure mock agent JSON is well-formed by parsing a sample exchange.
func TestMockAgent_SmokeTest(t *testing.T) {
	cmd := exec.Command(mockAgentCmd(), mockAgentArgs()...)
	cmd.Env = mockAgentEnv()

	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock: %v", err)
	}
	defer cmd.Process.Kill()

	writer := NewWriter(stdin)
	reader := NewReader(stdout, slog.Default())

	// Send initialize
	id, _ := writer.WriteRequest("initialize", map[string]any{
		"protocol_version": "2025-01-01",
		"client_info":      map[string]any{"name": "test", "version": "1.0"},
	})

	msg, err := reader.ReadMessage()
	if err != nil {
		t.Fatalf("read init response: %v", err)
	}
	if !msg.IsResponse() {
		t.Fatalf("expected response, got method=%s", msg.Method)
	}
	respID, _ := msg.GetID()
	if respID != id {
		t.Errorf("response ID mismatch: %d != %d", respID, id)
	}

	var result map[string]any
	json.Unmarshal(msg.Result, &result)
	if result["protocol_version"] != "2025-01-01" {
		t.Errorf("unexpected protocol version: %v", result["protocol_version"])
	}
}
