package mockagent

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

func TestEmitPermissionRequestIncludesMockEditPath(t *testing.T) {
	const path = "/workspace/src/component.css"
	t.Setenv("ACP_MOCK_EDIT_PATH", path)

	var out bytes.Buffer
	if _, err := emitPermissionRequest(acp.NewWriter(&out), 9001, "tc-1", "Edit"); err != nil {
		t.Fatalf("emit permission request: %v", err)
	}

	var msg struct {
		Params struct {
			ToolCall struct {
				RawInput struct {
					FilePath string `json:"file_path"`
				} `json:"rawInput"`
			} `json:"toolCall"`
		} `json:"params"`
	}
	if err := json.Unmarshal(out.Bytes(), &msg); err != nil {
		t.Fatalf("decode permission request: %v", err)
	}
	if got := msg.Params.ToolCall.RawInput.FilePath; got != path {
		t.Errorf("rawInput.file_path = %q, want %q", got, path)
	}
}
