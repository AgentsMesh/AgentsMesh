package mockagent

import (
	"bytes"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestScenario_Streaming3_EmitsThreeChunks(t *testing.T) {
	out := driveACP(t, "streaming_3",
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`+"\n"+
			`{"jsonrpc":"2.0","id":2,"method":"session/new","params":{}}`+"\n"+
			`{"jsonrpc":"2.0","id":3,"method":"session/prompt","params":{"prompt":"hi"}}`+"\n",
	)
	matches := strings.Count(out, `"sessionUpdate":"agent_message_chunk"`)
	if matches != 3 {
		t.Errorf("expected 3 agent_message_chunk notifications, got %d. output: %s", matches, out)
	}
	if !strings.Contains(out, `"text":"streaming: "`) ||
		!strings.Contains(out, `"text":"hi "`) ||
		!strings.Contains(out, `"text":"(done)"`) {
		t.Errorf("missing one or more streaming chunks. output: %s", out)
	}
}

func TestScenario_ThinkingThenAnswer_EmitsBoth(t *testing.T) {
	out := driveACP(t, "thinking_then_answer",
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`+"\n"+
			`{"jsonrpc":"2.0","id":2,"method":"session/new","params":{}}`+"\n"+
			`{"jsonrpc":"2.0","id":3,"method":"session/prompt","params":{"prompt":"x"}}`+"\n",
	)
	if !strings.Contains(out, `"sessionUpdate":"agent_thought_chunk"`) {
		t.Errorf("expected thinking notification, got: %s", out)
	}
	if !strings.Contains(out, `"sessionUpdate":"agent_message_chunk"`) {
		t.Errorf("expected answer notification, got: %s", out)
	}
}

func TestScenario_ToolCallEdit_EmitsCallAndCompleteResult(t *testing.T) {
	out := driveACP(t, "tool_call_edit",
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`+"\n"+
			`{"jsonrpc":"2.0","id":2,"method":"session/new","params":{}}`+"\n"+
			`{"jsonrpc":"2.0","id":3,"method":"session/prompt","params":{"prompt":"hi"}}`+"\n",
	)
	if !strings.Contains(out, `"sessionUpdate":"tool_call"`) {
		t.Errorf("expected tool_call notification, got: %s", out)
	}
	if !strings.Contains(out, `"sessionUpdate":"tool_call_update"`) ||
		!strings.Contains(out, `"status":"completed"`) {
		t.Errorf("expected completed tool_call_update, got: %s", out)
	}
}

func TestScenario_PermissionRequestEdit_ApproveCompletesTool(t *testing.T) {
	out := drivePermissionScenario(t, "allow_once", 1)
	if !strings.Contains(out, `"sessionUpdate":"tool_call_update"`) ||
		!strings.Contains(out, `"status":"completed"`) {
		t.Errorf("expected completed result on approve, got: %s", out)
	}
}

func TestScenario_PermissionRequestEdit_DenyFailsTool(t *testing.T) {
	out := drivePermissionScenario(t, "reject_once", 1)
	if !strings.Contains(out, `"status":"failed"`) ||
		!strings.Contains(out, `"errorMessage":"denied by user"`) {
		t.Errorf("expected failed result with denial message, got: %s", out)
	}
}

// drivePermissionScenario runs permission_request_edit with a scripted
// response. holdMs is the post-input grace period before EOF so the
// WaitGroup-drain has time to publish the tool_call_update.
func drivePermissionScenario(t *testing.T, optionID string, holdSec int) string {
	t.Helper()

	reader, writer := io.Pipe()
	var out bytes.Buffer
	var mu sync.Mutex
	syncOut := &lockedWriter{mu: &mu, w: &out}

	done := make(chan int)
	go func() {
		done <- runACPWithIO("permission_request_edit", reader, syncOut, slog.Default())
	}()

	handshake := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"session/new","params":{}}` + "\n" +
		`{"jsonrpc":"2.0","id":3,"method":"session/prompt","params":{"prompt":"hi"}}` + "\n"
	_, _ = writer.Write([]byte(handshake))

	// Wait for mock to actually emit the permission request before we
	// respond — otherwise our response races the registration and the
	// channel-based awaitResponse misses it.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		emitted := strings.Contains(out.String(), `"method":"session/request_permission"`)
		mu.Unlock()
		if emitted { break }
		time.Sleep(10 * time.Millisecond)
	}

	response := `{"jsonrpc":"2.0","id":9001,"result":{"outcome":{"outcome":"selected","optionId":"` + optionID + `"}}}` + "\n"
	_, _ = writer.Write([]byte(response))

	time.Sleep(time.Duration(holdSec) * time.Second)
	_ = writer.Close()
	<-done

	mu.Lock()
	defer mu.Unlock()
	return out.String()
}
