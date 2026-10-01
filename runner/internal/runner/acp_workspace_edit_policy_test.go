package runner

import (
	"encoding/json"
	"errors"
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
	// Build the handler the same way wireAndStartACPPod does, so this test
	// covers the production callback factory rather than a local stand-in.
	handler := acpPermissionHandler(&client, pod, policy)
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
				handler(req)
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

// A launcher may render the same flag as one token (--input-format=stream-json)
// instead of two. Both spellings describe an ACP session and must be classified
// identically, otherwise the workspace edit policy silently stops applying.
func TestACPWorkspaceEditPolicySingleTokenInputFormat(t *testing.T) {
	ws := t.TempDir()
	args := []string{"-p", "--input-format=stream-json"}
	if !newACPWorkspaceEditPolicy(ws, args).shouldAutoApprove(editRequest(t, filepath.Join(ws, "a.txt"))) {
		t.Error("--input-format=stream-json must classify as an ACP session")
	}
}

// fakeACPResponder stands in for *acp.ACPClient so the permission routing can be
// exercised without spawning an agent: it records the response, reports whether
// the request was held for human approval, and can be told to fail the write.
type fakeACPResponder struct {
	err       error
	responded []string
	pending   []acp.PermissionRequest
	done      chan struct{}
}

func (f *fakeACPResponder) RespondToPermission(requestID string, _ bool, _ map[string]any) error {
	f.responded = append(f.responded, requestID)
	if f.done != nil {
		f.done <- struct{}{}
	}
	return f.err
}

func (f *fakeACPResponder) AddPendingPermission(req acp.PermissionRequest) {
	f.pending = append(f.pending, req)
}

func (f *fakeACPResponder) waitForResponse(t *testing.T) {
	t.Helper()
	select {
	case <-f.done:
	case <-time.After(5 * time.Second):
		t.Fatal("workspace edit was never answered")
	}
}

func TestHandleACPPermissionRequestAutoApprovesInWorkspace(t *testing.T) {
	ws := t.TempDir()
	responder := &fakeACPResponder{done: make(chan struct{}, 1)}
	req := editRequest(t, filepath.Join(ws, "src", "component.css"))
	req.RequestID = "req-inside"

	handleACPPermissionRequest(responder, &Pod{PodKey: "pod-1"}, newACPWorkspaceEditPolicy(ws, claudeACPArgs()), req)
	responder.waitForResponse(t)

	if len(responder.responded) != 1 || responder.responded[0] != "req-inside" {
		t.Errorf("expected one inline approval for req-inside, got %v", responder.responded)
	}
	if len(responder.pending) != 0 {
		t.Errorf("auto-approved edit must not stay pending for a human: %+v", responder.pending)
	}
}

// A failed inline answer must not silently fall through to the browser flow: the
// request would then never resolve, which is the behaviour issue #241 reports.
func TestHandleACPPermissionRequestDropsFailedInlineAnswer(t *testing.T) {
	ws := t.TempDir()
	responder := &fakeACPResponder{err: errors.New("transport closed"), done: make(chan struct{}, 1)}
	req := editRequest(t, filepath.Join(ws, "src", "component.css"))
	req.RequestID = "req-failed"

	handleACPPermissionRequest(responder, &Pod{PodKey: "pod-2"}, newACPWorkspaceEditPolicy(ws, claudeACPArgs()), req)
	responder.waitForResponse(t)

	if len(responder.responded) != 1 {
		t.Errorf("expected one attempted inline approval, got %v", responder.responded)
	}
	if len(responder.pending) != 0 {
		t.Errorf("failed inline approval must not be re-queued for a human: %+v", responder.pending)
	}
}

func TestHandleACPPermissionRequestKeepsHumanApprovalOutsideWorkspace(t *testing.T) {
	ws := t.TempDir()
	responder := &fakeACPResponder{}
	req := editRequest(t, filepath.Join(filepath.Dir(ws), "sibling.txt"))
	req.RequestID = "req-outside"

	handleACPPermissionRequest(responder, &Pod{PodKey: "pod-3"}, newACPWorkspaceEditPolicy(ws, claudeACPArgs()), req)

	if len(responder.responded) != 0 {
		t.Errorf("out-of-workspace edit must never be auto-approved: %v", responder.responded)
	}
	if len(responder.pending) != 1 || responder.pending[0].RequestID != "req-outside" {
		t.Errorf("out-of-workspace edit must stay pending for a human: %+v", responder.pending)
	}
}
