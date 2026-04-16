package claude

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteStdin_MarshalError(t *testing.T) {
	tr := newTransport(acp.EventCallbacks{}, discardLogger())
	tr.stdin = io.Discard
	err := tr.writeStdin(make(chan int))
	if err == nil {
		t.Error("expected marshal error for channel type")
	}
}

func TestWriteStdin_WriteError(t *testing.T) {
	_, pw := io.Pipe()
	pw.Close()
	tr := newTransport(acp.EventCallbacks{}, discardLogger())
	tr.stdin = pw
	err := tr.writeStdin("hello")
	if err == nil {
		t.Error("expected write error on closed pipe")
	}
}

func TestHandshake_NilStdin(t *testing.T) {
	tr := newTransport(acp.EventCallbacks{}, discardLogger())
	tr.initCh = make(chan struct{})
	close(tr.initCh)
	sid, err := tr.Handshake(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "", sid)
}

func TestHandshake_WriteError(t *testing.T) {
	_, pw := io.Pipe()
	pw.Close()
	tr := newTransport(acp.EventCallbacks{}, discardLogger())
	tr.stdin = pw
	_, err := tr.Handshake(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write control_request initialize")
}

func TestRespondToPermission_WithUpdatedPermissions(t *testing.T) {
	stdinPR, stdinPW := io.Pipe()
	defer stdinPR.Close()
	defer stdinPW.Close()

	tr := newTransport(acp.EventCallbacks{}, discardLogger())
	tr.stdin = stdinPW

	done := make(chan string)
	go func() {
		buf := make([]byte, 8192)
		n, _ := stdinPR.Read(buf)
		done <- string(buf[:n])
	}()

	updatedInput := map[string]any{
		"file_path":          "/tmp/test.go",
		"updatedPermissions": map[string]any{"allow_write": true},
	}
	err := tr.RespondToPermission("req-perm", true, updatedInput)
	require.NoError(t, err)

	select {
	case received := <-done:
		var msg controlResponseMessage
		require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(received)), &msg))
		assert.Equal(t, "allow", msg.Response.Response.Behavior)
		assert.NotNil(t, msg.Response.Response.UpdatedPermissions)

		var perms map[string]any
		require.NoError(t, json.Unmarshal(msg.Response.Response.UpdatedPermissions, &perms))
		assert.Equal(t, true, perms["allow_write"])

		var input map[string]any
		require.NoError(t, json.Unmarshal(msg.Response.Response.UpdatedInput, &input))
		assert.Equal(t, "/tmp/test.go", input["file_path"])
		_, hasPerms := input["updatedPermissions"]
		assert.False(t, hasPerms, "updatedPermissions should be removed from updatedInput")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestRespondToPermission_WithOnlyUpdatedPermissions(t *testing.T) {
	stdinPR, stdinPW := io.Pipe()
	defer stdinPR.Close()
	defer stdinPW.Close()

	tr := newTransport(acp.EventCallbacks{}, discardLogger())
	tr.stdin = stdinPW

	tr.pendingInputsMu.Lock()
	tr.pendingInputs["req-only-perms"] = json.RawMessage(`{"original":"data"}`)
	tr.pendingInputsMu.Unlock()

	done := make(chan string)
	go func() {
		buf := make([]byte, 8192)
		n, _ := stdinPR.Read(buf)
		done <- string(buf[:n])
	}()

	updatedInput := map[string]any{
		"updatedPermissions": map[string]any{"allow_all": true},
	}
	err := tr.RespondToPermission("req-only-perms", true, updatedInput)
	require.NoError(t, err)

	select {
	case received := <-done:
		var msg controlResponseMessage
		require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(received)), &msg))
		var input map[string]any
		require.NoError(t, json.Unmarshal(msg.Response.Response.UpdatedInput, &input))
		assert.Equal(t, "data", input["original"])
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestResolveOutgoingControlResponse_EmptyResponse(t *testing.T) {
	tr := newTransport(acp.EventCallbacks{}, discardLogger())
	msg := &message{Response: nil}
	assert.False(t, tr.resolveOutgoingControlResponse(msg))
}

func TestResolveOutgoingControlResponse_ParseError(t *testing.T) {
	tr := newTransport(acp.EventCallbacks{}, discardLogger())
	msg := &message{Response: json.RawMessage(`not json`)}
	assert.False(t, tr.resolveOutgoingControlResponse(msg))
}

func TestResolveOutgoingControlResponse_EmptyRequestID(t *testing.T) {
	tr := newTransport(acp.EventCallbacks{}, discardLogger())
	msg := &message{
		Response: json.RawMessage(`{"subtype":"success","request_id":"","response":{}}`),
	}
	assert.False(t, tr.resolveOutgoingControlResponse(msg))
}

func TestResolveOutgoingControlResponse_ErrorSubtype(t *testing.T) {
	tr := newTransport(acp.EventCallbacks{}, discardLogger())
	ch := tr.outgoing.track("err-req-1")

	msg := &message{
		Response: json.RawMessage(`{"subtype":"error","request_id":"err-req-1","error":"something went wrong"}`),
	}
	resolved := tr.resolveOutgoingControlResponse(msg)
	assert.True(t, resolved)

	result := <-ch
	assert.Error(t, result.err)
	assert.Contains(t, result.err.Error(), "something went wrong")
}

func TestResolveOutgoingControlResponse_UnknownRequestID(t *testing.T) {
	tr := newTransport(acp.EventCallbacks{}, discardLogger())
	msg := &message{
		Response: json.RawMessage(`{"subtype":"success","request_id":"no-such-id","response":{}}`),
	}
	assert.False(t, tr.resolveOutgoingControlResponse(msg))
}

func TestTransport_RegisterFactory(t *testing.T) {
	tr := acp.NewTransport(TransportType, acp.EventCallbacks{}, discardLogger())
	if tr == nil {
		t.Fatal("NewTransport returned nil for claude-stream")
	}
	_, ok := tr.(*transport)
	if !ok {
		t.Errorf("expected *claude.Transport, got %T", tr)
	}
}

func TestTransport_CommandMapping(t *testing.T) {
	tt := acp.TransportTypeForCommand("claude")
	assert.Equal(t, TransportType, tt)
}
