package acp

import (
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"
)

// Mock agent modes (selected via ACP_MOCK_MODE env var).
const (
	mockModeDefault     = ""
	mockModeSendReq     = "send_request"
	mockModeErrorInit   = "error_init"
	mockModeSlowInit    = "slow_init"
	mockModeBadJSONNew  = "bad_json_new"
	mockModeSendPerm    = "send_permission"
	mockModeValidate    = "validate"
	mockModeMultiUpdate = "multi_update"
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

// runMockAgent dispatches JSON-RPC messages based on ACP_MOCK_MODE.
func runMockAgent() {
	mode := os.Getenv("ACP_MOCK_MODE")
	reader := NewReader(os.Stdin, slog.Default())
	writer := NewWriter(os.Stdout)

	for {
		msg, err := reader.ReadMessage()
		if err != nil {
			return
		}
		switch {
		case msg.IsRequest():
			id, _ := msg.GetID()
			handleMockRequest(mode, id, msg, reader, writer)
		case msg.IsNotification():
			// Notifications don't require a response
		}
	}
}

// handleMockRequest routes a JSON-RPC request to the appropriate handler.
func handleMockRequest(mode string, id int64, msg *JSONRPCMessage, reader *Reader, writer *Writer) {
	switch msg.Method {
	case "initialize":
		handleMockInitialize(mode, id, msg.Params, writer)
	case "session/new":
		handleMockSessionNew(mode, id, msg.Params, writer)
	case "session/prompt":
		handleMockSessionPrompt(mode, id, msg.Params, reader, writer)
	default:
		writer.WriteResponse(id, nil, &JSONRPCError{
			Code:    ErrCodeMethodNotFound,
			Message: fmt.Sprintf("unknown method: %s", msg.Method),
		})
	}
}

// handleMockInitialize handles the initialize request for all modes.
func handleMockInitialize(mode string, id int64, params []byte, writer *Writer) {
	switch mode {
	case mockModeErrorInit:
		writer.WriteResponse(id, nil, &JSONRPCError{
			Code: ErrCodeInternal, Message: "mock init error",
		})
	case mockModeSlowInit:
		time.Sleep(5 * time.Second)
		writer.WriteResponse(id, map[string]any{"protocol_version": "2025-01-01"}, nil)
	case mockModeValidate:
		mockHandleValidateInit(id, params, writer)
	default:
		writer.WriteResponse(id, map[string]any{
			"protocol_version": "2025-01-01",
			"capabilities":     map[string]any{"permissions": true},
		}, nil)
		if mode == mockModeSendReq {
			writer.WriteRequest("agent/custom_method", map[string]any{
				"info": "agent-initiated request",
			})
		}
	}
}

// handleMockSessionNew handles the session/new request for all modes.
func handleMockSessionNew(mode string, id int64, params []byte, writer *Writer) {
	switch mode {
	case mockModeBadJSONNew:
		raw := fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"result":"not_an_object"}`, id)
		os.Stdout.Write([]byte(raw + "\n"))
	case mockModeValidate:
		mockHandleValidateSessionNew(id, params, writer)
	default:
		writer.WriteResponse(id, map[string]any{"sessionId": "mock-session-001"}, nil)
	}
}

// handleMockSessionPrompt handles the session/prompt request for all modes.
func handleMockSessionPrompt(mode string, id int64, params []byte, reader *Reader, writer *Writer) {
	switch mode {
	case mockModeSendPerm:
		mockHandlePermissionPrompt(id, reader, writer)
	case mockModeValidate:
		mockHandleValidatePrompt(id, params, writer)
	case mockModeMultiUpdate:
		mockHandleMultiUpdatePrompt(id, writer)
	default:
		mockSendDefaultContent(id, writer)
	}
}

// mockSendDefaultContent sends a single agent_message_chunk and responds.
func mockSendDefaultContent(id int64, writer *Writer) {
	writer.WriteNotification("session/update", map[string]any{
		"sessionId": "mock-session-001",
		"update": map[string]any{
			"sessionUpdate": "agent_message_chunk",
			"content":       map[string]any{"type": "text", "text": "Hello from mock agent"},
		},
	})
	writer.WriteResponse(id, map[string]any{"stopReason": "end_turn"}, nil)
}

// mockAgentCmd returns the test binary path for use as a mock agent.
func mockAgentCmd() string { return os.Args[0] }

// mockAgentArgs returns args that trigger the mock agent mode.
func mockAgentArgs() []string { return []string{"-test.run=TestMain"} }

// mockAgentEnv returns environment variables for the mock agent (default mode).
func mockAgentEnv() []string {
	return append(os.Environ(), "ACP_MOCK_AGENT=1")
}

// mockAgentEnvWithMode returns env vars for a specific mock agent mode.
func mockAgentEnvWithMode(mode string) []string {
	return append(os.Environ(), "ACP_MOCK_AGENT=1", "ACP_MOCK_MODE="+mode)
}

// startMockClient creates and starts a mock ACP client for testing.
func startMockClient(t *testing.T) *ACPClient {
	t.Helper()
	return startMockClientWithMode(t, mockModeDefault, EventCallbacks{})
}

// startMockClientWithMode creates and starts a mock ACP client with a specific mode.
func startMockClientWithMode(t *testing.T, mode string, callbacks EventCallbacks) *ACPClient {
	t.Helper()
	env := mockAgentEnv()
	if mode != "" {
		env = mockAgentEnvWithMode(mode)
	}
	client := NewClient(ClientConfig{
		Command:   mockAgentCmd(),
		Args:      mockAgentArgs(),
		Env:       env,
		WorkDir:   t.TempDir(),
		Logger:    slog.Default(),
		Callbacks: callbacks,
	})
	if err := client.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	return client
}
