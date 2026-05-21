package mockagent

import (
	"strings"
	"testing"
)

func TestControlRequest_SetThinkingLevel_Acked(t *testing.T) {
	out := driveACP(t, "config_change_plan",
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`+"\n"+
			`{"jsonrpc":"2.0","id":2,"method":"session/new","params":{}}`+"\n"+
			`{"jsonrpc":"2.0","id":3,"method":"session/control_request","params":{"sessionId":"mock-session-001","subtype":"set_thinking_level","params":{"level":"high"}}}`+"\n",
	)
	if !strings.Contains(out, `"id":3,"result":{"ok":true}`) {
		t.Errorf("expected ok response for set_thinking_level, got: %s", out)
	}
}

func TestControlRequest_Interrupt_Acked(t *testing.T) {
	out := driveACP(t, "config_change_plan",
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`+"\n"+
			`{"jsonrpc":"2.0","id":2,"method":"session/new","params":{}}`+"\n"+
			`{"jsonrpc":"2.0","id":3,"method":"session/control_request","params":{"sessionId":"mock-session-001","subtype":"interrupt"}}`+"\n",
	)
	if !strings.Contains(out, `"id":3,"result":{"ok":true}`) {
		t.Errorf("expected ok response for interrupt, got: %s", out)
	}
}

func TestControlRequest_GetContextUsage_ReturnsSyntheticUsage(t *testing.T) {
	out := driveACP(t, "config_change_plan",
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`+"\n"+
			`{"jsonrpc":"2.0","id":2,"method":"session/new","params":{}}`+"\n"+
			`{"jsonrpc":"2.0","id":3,"method":"session/control_request","params":{"sessionId":"mock-session-001","subtype":"get_context_usage"}}`+"\n",
	)
	for _, want := range []string{`"input_tokens":1234`, `"output_tokens":567`, `"total_tokens":1801`} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %s in response, got: %s", want, out)
		}
	}
}

func TestControlRequest_UnknownSubtype_ReturnsMethodNotFound(t *testing.T) {
	out := driveACP(t, "config_change_plan",
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`+"\n"+
			`{"jsonrpc":"2.0","id":2,"method":"session/new","params":{}}`+"\n"+
			`{"jsonrpc":"2.0","id":3,"method":"session/control_request","params":{"sessionId":"mock-session-001","subtype":"bogus_subtype"}}`+"\n",
	)
	if !strings.Contains(out, `"code":-32601`) {
		t.Errorf("expected method_not_found error, got: %s", out)
	}
}
