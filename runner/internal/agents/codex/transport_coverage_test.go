package codex

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var timeZero = time.Time{}

func TestHandshake_ErrorResponse(t *testing.T) {
	f := newFixture()
	defer f.Close()

	go func() {
		r := bufio.NewReader(f.StdinPR)
		line, _ := r.ReadBytes('\n')
		var req map[string]any
		json.Unmarshal(line, &req)
		id := int64(req["id"].(float64))
		writeResponse(f.PW, id, nil, &acp.JSONRPCError{Code: -1, Message: "fail"})
	}()

	_, err := f.transport.Handshake(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "initialize error")
}

func TestNewSession_ErrorResponse(t *testing.T) {
	f := newFixture()
	defer f.Close()

	go func() {
		r := bufio.NewReader(f.StdinPR)
		line, _ := r.ReadBytes('\n')
		var req map[string]any
		json.Unmarshal(line, &req)
		id := int64(req["id"].(float64))
		writeResponse(f.PW, id, nil, &acp.JSONRPCError{Code: -2, Message: "no thread"})
	}()

	_, err := f.transport.NewSession("", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "thread/start error")
}

func TestNewSession_InvalidResult(t *testing.T) {
	f := newFixture()
	defer f.Close()

	go func() {
		r := bufio.NewReader(f.StdinPR)
		line, _ := r.ReadBytes('\n')
		var req map[string]any
		json.Unmarshal(line, &req)
		id := int64(req["id"].(float64))
		writeResponse(f.PW, id, "not-an-object", nil)
	}()

	_, err := f.transport.NewSession("", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse thread/start result")
}

func TestSendPrompt_RpcErrorInBackground(t *testing.T) {
	f := newFixture()
	defer f.Close()
	f.transport.sessionMu.Lock()
	f.transport.sessionID = "s1"
	f.transport.sessionMu.Unlock()

	go func() {
		r := bufio.NewReader(f.StdinPR)
		line, _ := r.ReadBytes('\n')
		var req map[string]any
		json.Unmarshal(line, &req)
		id := int64(req["id"].(float64))
		writeResponse(f.PW, id, nil, &acp.JSONRPCError{Code: -3, Message: "turn err"})
	}()

	err := f.transport.SendPrompt("s1", "hello")
	assert.NoError(t, err)
	f.Drain()
}

func TestCancelSession_ViaFixture(t *testing.T) {
	f := newFixture()
	defer f.Close()
	f.transport.sessionMu.Lock()
	f.transport.sessionID = "s1"
	f.transport.sessionMu.Unlock()

	go func() {
		r := bufio.NewReader(f.StdinPR)
		line, _ := r.ReadBytes('\n')
		var req map[string]any
		json.Unmarshal(line, &req)
		id := int64(req["id"].(float64))
		writeResponse(f.PW, id, map[string]any{}, nil)
	}()

	err := f.transport.CancelSession("s1")
	assert.NoError(t, err)
	f.Drain()
}

func TestApprovalRequest_ParseError(t *testing.T) {
	f := newFixture()
	defer f.Close()

	msg := map[string]any{
		"jsonrpc": "2.0", "id": 99,
		"method": "item/commandExecution/requestApproval",
		"params": "not-an-object",
	}
	data, _ := json.Marshal(msg)
	f.PW.Write(append(data, '\n'))
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	assert.Empty(t, f.PermissionReqs)
}

func TestApprovalRequest_WithDescription(t *testing.T) {
	f := newFixture()
	defer f.Close()

	msg := map[string]any{
		"jsonrpc": "2.0", "id": 100,
		"method": "item/commandExecution/requestApproval",
		"params": map[string]any{"description": "run tests", "command": "go test"},
	}
	data, _ := json.Marshal(msg)
	f.PW.Write(append(data, '\n'))
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if assert.Len(t, f.PermissionReqs, 1) {
		assert.Equal(t, "run tests", f.PermissionReqs[0].Description)
	}
}

func TestParseCodexSessionsDir_Nonexistent(t *testing.T) {
	usage := tokenusage.NewTokenUsage()
	parseCodexSessionsDir("/nonexistent", timeZero, usage)
	assert.True(t, usage.IsEmpty())
}

func TestParseCodexJSONLFile_OpenError(t *testing.T) {
	usage := tokenusage.NewTokenUsage()
	err := parseCodexJSONLFile("/nonexistent/file.jsonl", usage)
	assert.Error(t, err)
}

func TestParseCodexJSONLFile_EmptyAndMixed(t *testing.T) {
	dir := t.TempDir()
	file := dir + "/test.jsonl"
	content := "\n\n{\"model\":\"m\",\"usage\":{\"input_tokens\":1,\"output_tokens\":1}}\n"
	require.NoError(t, os.WriteFile(file, []byte(content), 0644))

	usage := tokenusage.NewTokenUsage()
	err := parseCodexJSONLFile(file, usage)
	assert.NoError(t, err)
	assert.False(t, usage.IsEmpty())
}
