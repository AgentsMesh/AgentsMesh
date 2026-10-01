package runner

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
	"github.com/anthropics/agentsmesh/runner/internal/agents/mockagent"
)

const (
	mockEditEnv    = "AGENTSMESH_ACP_EDIT_MOCK"
	mockEditTarget = "ACP_MOCK_EDIT_PATH"
	mockEditToolID = "tc-mock-edit-perm-1"
)

// claudeACPArgs mirrors the args the shipped Claude Code AgentFile resolves for
// MODE acp; extra flags stand in for a non-default permission_mode.
func claudeACPArgs(extra ...string) []string {
	base := []string{"-p", "--verbose", "--input-format", "stream-json", "--output-format", "stream-json"}
	return append(base, extra...)
}

func editRequest(t *testing.T, path string) acp.PermissionRequest {
	t.Helper()
	args, err := json.Marshal(map[string]any{"file_path": path, "old_string": "a", "new_string": "b"})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	return acp.PermissionRequest{ToolName: "Edit", ArgumentsJSON: string(args)}
}

// Pre-fix, no ACP session had a workspace-edit policy, so every Edit waited for
// a browser and surfaced as denied in headless pods (issue #241).
func TestACPWorkspaceEditPolicy(t *testing.T) {
	ws := t.TempDir()
	inside := editRequest(t, filepath.Join(ws, "src", "component.css"))
	if !newACPWorkspaceEditPolicy(ws, claudeACPArgs()).shouldAutoApprove(inside) {
		t.Error("CLI-default ACP session must allow workspace edits")
	}
	if !newACPWorkspaceEditPolicy(ws, claudeACPArgs("--permission-mode", "acceptEdits")).shouldAutoApprove(inside) {
		t.Error("acceptEdits must allow workspace edits")
	}

	// ACP rewrites "plan" to --permission-mode default; that explicit default
	// must stay read-only, as must dontAsk and every non-workspace target.
	for _, tc := range []struct {
		name string
		args []string
		req  acp.PermissionRequest
	}{
		{"plan rewritten to default", claudeACPArgs("--permission-mode", "default"), inside},
		{"dontAsk", claudeACPArgs("--permission-mode", "dontAsk"), inside},
		{"outside workspace", claudeACPArgs(), editRequest(t, filepath.Join(filepath.Dir(ws), "sibling.txt"))},
		{"parent escape", claudeACPArgs(), editRequest(t, filepath.Join(ws, "..", "escaped.txt"))},
		{"relative path", claudeACPArgs(), editRequest(t, "relative.txt")},
		{"non-edit tool", claudeACPArgs(), acp.PermissionRequest{ToolName: "Bash", ArgumentsJSON: `{"command":"rm -rf /"}`}},
		{"unparseable args", claudeACPArgs(), acp.PermissionRequest{ToolName: "Edit", ArgumentsJSON: "not-json"}},
		{"generic acp agent", []string{"acp"}, inside},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if newACPWorkspaceEditPolicy(ws, tc.args).shouldAutoApprove(tc.req) {
				t.Error("request must stay human-gated")
			}
		})
	}
}

// TestACPMockEditAgent re-runs the repository's e2e mock ACP agent as a
// subprocess so the permission path runs over a real stdin/stdout pipe.
func TestACPMockEditAgent(t *testing.T) {
	if os.Getenv(mockEditEnv) != "1" {
		t.Skip("mock ACP edit agent entry point")
	}
	if code := mockagent.RunACP("permission_request_edit", slog.Default()); code != 0 {
		t.Fatalf("mock ACP agent exited with code %d", code)
	}
}

// TestACPPodAutoApprovesWorkspaceEdit drives the real permission path: the mock
// agent asks to Edit a workspace file, and the production handler
// (handleACPPermissionRequest, wired by wireAndStartACPPod) must answer it
// itself so the Edit completes with no browser attached.
func TestACPPodAutoApprovesWorkspaceEdit(t *testing.T) {
	ws := t.TempDir()
	target := filepath.Join(ws, "mock-target.txt")
	policy := newACPWorkspaceEditPolicy(ws, claudeACPArgs())
	var forwarded []acp.PermissionRequest
	pod := &Pod{PodKey: "mock-edit-pod"}
	var client *acp.ACPClient
	client = acp.NewClient(acp.ClientConfig{
		Command: os.Args[0],
		Args:    []string{"-test.run=^TestACPMockEditAgent$"},
		Env:     append(os.Environ(), mockEditEnv+"=1", mockEditTarget+"="+target),
		WorkDir: ws,
		Logger:  slog.Default(),
		Callbacks: acp.EventCallbacks{
			OnPermissionRequest: func(req acp.PermissionRequest) {
				if !policy.shouldAutoApprove(req) {
					forwarded = append(forwarded, req)
				}
				handleACPPermissionRequest(client, pod, policy, req)
			},
		},
	})
	if err := client.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer client.Stop()
	if err := client.NewSession(nil); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if err := client.SendPrompt("fix the padding"); err != nil {
		t.Fatalf("SendPrompt: %v", err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		for _, tc := range client.GetSessionSnapshot().ToolCalls {
			// A denied/timed-out permission also normalises the tool call to
			// "completed", so assert on the recorded outcome instead.
			if tc.ToolCallID != mockEditToolID || tc.Success == nil || !*tc.Success {
				continue
			}
			if len(forwarded) != 0 {
				t.Errorf("workspace edit was forwarded for human approval: %+v", forwarded)
			}
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("workspace edit never succeeded; forwarded=%+v tool calls=%+v",
		forwarded, client.GetSessionSnapshot().ToolCalls)
}
