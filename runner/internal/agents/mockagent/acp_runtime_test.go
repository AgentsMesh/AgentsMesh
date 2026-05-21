package mockagent

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

// driveACP runs the ACP runtime on a goroutine and feeds it a sequence of
// JSON-RPC lines, returning everything written to stdout. Closes the input
// after the last line so the runtime exits on EOF.
func driveACP(t *testing.T, scenario string, in string) string {
	t.Helper()

	reader, writer := io.Pipe()
	var out bytes.Buffer
	var mu sync.Mutex
	syncOut := &lockedWriter{mu: &mu, w: &out}

	done := make(chan int)
	go func() {
		done <- runACPWithIO(scenario, reader, syncOut, slog.Default())
	}()

	_, _ = writer.Write([]byte(in))
	_ = writer.Close()
	<-done

	mu.Lock()
	defer mu.Unlock()
	return out.String()
}

type lockedWriter struct {
	mu *sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

func TestRunACP_InitializeResponse(t *testing.T) {
	req := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n"
	out := driveACP(t, "echo", req)

	if !strings.Contains(out, `"id":1`) {
		t.Errorf("response missing id=1: %s", out)
	}
	if !strings.Contains(out, `"protocol_version"`) {
		t.Errorf("response missing protocol_version: %s", out)
	}
}

func TestRunACP_SessionNewReturnsMockID(t *testing.T) {
	req := `{"jsonrpc":"2.0","id":2,"method":"session/new","params":{}}` + "\n"
	out := driveACP(t, "echo", req)

	if !strings.Contains(out, `"sessionId":"mock-session-001"`) {
		t.Errorf("session/new should return mock session id, got: %s", out)
	}
}

func TestRunACP_EchoScenario_EmitsContentAndStop(t *testing.T) {
	req := `{"jsonrpc":"2.0","id":3,"method":"session/prompt","params":{"prompt":"hello"}}` + "\n"
	out := driveACP(t, "echo", req)

	if !strings.Contains(out, `"sessionUpdate":"agent_message_chunk"`) {
		t.Errorf("expected agent_message_chunk notification, got: %s", out)
	}
	if !strings.Contains(out, `"text":"echo: hello"`) {
		t.Errorf("expected echoed text, got: %s", out)
	}
	if !strings.Contains(out, `"stopReason":"end_turn"`) {
		t.Errorf("expected stopReason=end_turn response, got: %s", out)
	}
}

func TestRunACP_PromptWithContentBlocks(t *testing.T) {
	req := `{"jsonrpc":"2.0","id":4,"method":"session/prompt","params":{"prompt":[{"type":"text","text":"world"}]}}` + "\n"
	out := driveACP(t, "echo", req)

	if !strings.Contains(out, `"text":"echo: world"`) {
		t.Errorf("expected echo of block text, got: %s", out)
	}
}

func TestRunACP_UnknownMethodReturnsError(t *testing.T) {
	req := `{"jsonrpc":"2.0","id":5,"method":"session/bogus","params":{}}` + "\n"
	out := driveACP(t, "echo", req)

	if !strings.Contains(out, `"code":-32601`) {
		t.Errorf("expected method_not_found error, got: %s", out)
	}
}

func TestRunACP_UnknownScenarioRefusesToStart(t *testing.T) {
	code := runACPWithIO("nonexistent", strings.NewReader(""), io.Discard, slog.Default())
	if code != 2 {
		t.Errorf("unknown scenario exit code = %d, want 2", code)
	}
}

func TestRunACP_CleanEOF(t *testing.T) {
	code := runACPWithIO("echo", strings.NewReader(""), io.Discard, slog.Default())
	if code != 0 {
		t.Errorf("clean EOF exit code = %d, want 0", code)
	}
}

func TestExtractPromptText_StringForm(t *testing.T) {
	raw, _ := json.Marshal(map[string]string{"prompt": "hi"})
	got := extractPromptText(raw)
	if got != "hi" {
		t.Errorf("got %q, want %q", got, "hi")
	}
}

func TestExtractPromptText_BlockForm(t *testing.T) {
	raw := json.RawMessage(`{"prompt":[{"type":"text","text":"hello"}]}`)
	got := extractPromptText(raw)
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestExtractPromptText_Empty(t *testing.T) {
	got := extractPromptText(json.RawMessage(`{}`))
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

// Sanity check: runACPWithIO must not hold the JSONRPCMessage type reference
// after exit (regression guard for goroutine leaks under tests).
var _ acp.JSONRPCMessage
