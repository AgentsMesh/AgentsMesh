//go:build integration

package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// --- Mock providers ---

type spyPodProvider struct {
	mu       sync.Mutex
	snapshot string
	calls    []sendInputCall
}

type sendInputCall struct {
	PodKey string
	Text   string
	Keys   []string
}

func (p *spyPodProvider) GetPodSnapshot(podKey string, lines int) (string, error) {
	if p.snapshot != "" {
		return p.snapshot, nil
	}
	return "", fmt.Errorf("pod %s not found", podKey)
}

func (p *spyPodProvider) SendPodInput(podKey, text string, keys []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls = append(p.calls, sendInputCall{PodKey: podKey, Text: text, Keys: keys})
	return nil
}

func (p *spyPodProvider) lastCall() (sendInputCall, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.calls) == 0 {
		return sendInputCall{}, false
	}
	return p.calls[len(p.calls)-1], true
}

type fakeStatusProvider struct {
	agentStatus string
	podStatus   string
	shellPid    int
	knownPods   map[string]bool
}

func (f *fakeStatusProvider) GetPodStatus(podKey string) (string, string, int, bool) {
	if f.knownPods[podKey] {
		return f.agentStatus, f.podStatus, f.shellPid, true
	}
	return "", "", 0, false
}

// --- Helpers ---

func startTestServer(t *testing.T, s *HTTPServer) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", s.handleMCP)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

func doToolCall(t *testing.T, url, podKey, tool, argsJSON string) MCPResponse {
	t.Helper()
	body := fmt.Sprintf(
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"%s","arguments":%s}}`,
		tool, argsJSON,
	)
	req, err := http.NewRequest(http.MethodPost, url+"/mcp", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("X-Pod-Key", podKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var mcpResp MCPResponse
	if err := json.Unmarshal(data, &mcpResp); err != nil {
		t.Fatalf("decode response: %v\nbody: %s", err, string(data))
	}
	return mcpResp
}

func extractText(t *testing.T, resp MCPResponse) string {
	t.Helper()
	if resp.Error != nil {
		t.Fatalf("unexpected RPC error: %d %s", resp.Error.Code, resp.Error.Message)
	}
	rm, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected result type: %T", resp.Result)
	}
	content, ok := rm["content"].([]interface{})
	if !ok || len(content) == 0 {
		t.Fatal("no content in result")
	}
	first, ok := content[0].(map[string]interface{})
	if !ok {
		t.Fatal("unexpected content block type")
	}
	text, ok := first["text"].(string)
	if !ok {
		t.Fatal("no text field in content block")
	}
	return text
}

// --- Integration Tests ---

func TestMCPToolCall_GetStatus_Integration(t *testing.T) {
	srv := NewHTTPServer(nil, 0)
	srv.RegisterPod("caller-pod", "org1", nil, nil, "claude")
	srv.SetStatusProvider(&fakeStatusProvider{
		agentStatus: "waiting",
		podStatus:   "running",
		shellPid:    9999,
		knownPods:   map[string]bool{"target-pod": true},
	})

	ts := startTestServer(t, srv)

	// Known pod
	resp := doToolCall(t, ts.URL, "caller-pod", "get_pod_status", `{"pod_key":"target-pod"}`)
	text := extractText(t, resp)
	assertContains(t, text, "Pod: target-pod")
	assertContains(t, text, "Agent: waiting")
	assertContains(t, text, "Status: running")

	// Unknown pod
	resp2 := doToolCall(t, ts.URL, "caller-pod", "get_pod_status", `{"pod_key":"missing-pod"}`)
	text2 := extractText(t, resp2)
	assertContains(t, text2, "not_found")
}

func TestMCPToolCall_SendInput_Integration(t *testing.T) {
	spy := &spyPodProvider{}
	srv := NewHTTPServer(nil, 0)
	srv.RegisterPod("caller-pod", "org1", nil, nil, "claude")
	srv.SetPodProvider(spy)

	ts := startTestServer(t, srv)

	resp := doToolCall(t, ts.URL, "caller-pod", "send_pod_input",
		`{"pod_key":"worker-pod","text":"ls -la","keys":["enter"]}`)
	text := extractText(t, resp)

	if text != "Input sent successfully" {
		t.Errorf("expected success message, got: %s", text)
	}

	call, ok := spy.lastCall()
	if !ok {
		t.Fatal("SendPodInput was never called")
	}
	if call.PodKey != "worker-pod" {
		t.Errorf("pod_key: got %q, want %q", call.PodKey, "worker-pod")
	}
	if call.Text != "ls -la" {
		t.Errorf("text: got %q, want %q", call.Text, "ls -la")
	}
	if len(call.Keys) != 1 || call.Keys[0] != "enter" {
		t.Errorf("keys: got %v, want [enter]", call.Keys)
	}
}

func TestMCPToolCall_GetSnapshot_Integration(t *testing.T) {
	provider := &spyPodProvider{snapshot: "$ git status\nOn branch main\nnothing to commit"}
	srv := NewHTTPServer(nil, 0)
	srv.RegisterPod("caller-pod", "org1", nil, nil, "claude")
	srv.SetPodProvider(provider)

	ts := startTestServer(t, srv)

	resp := doToolCall(t, ts.URL, "caller-pod", "get_pod_snapshot",
		`{"pod_key":"worker-pod","lines":100}`)
	text := extractText(t, resp)

	assertContains(t, text, "git status")
	assertContains(t, text, "nothing to commit")
}
