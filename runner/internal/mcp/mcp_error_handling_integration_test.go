//go:build integration

package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMCP_ToolCallMissingArgs_Integration calls send_pod_input without
// required arguments and verifies an error response.
func TestMCP_ToolCallMissingArgs_Integration(t *testing.T) {
	spy := &spyPodProvider{}
	srv := NewHTTPServer(nil, 0)
	srv.RegisterPod("caller-pod", "org1", nil, nil, "claude")
	srv.SetPodProvider(spy)

	ts := startTestServer(t, srv)

	// Missing "pod_key" and "text" args — only send empty object.
	resp := doToolCall(t, ts.URL, "caller-pod", "send_pod_input", `{}`)

	// The tool handler should return an error result (isError: true).
	rm, ok := resp.Result.(map[string]interface{})
	require.True(t, ok, "result should be a map")
	isErr, _ := rm["isError"].(bool)
	assert.True(t, isErr, "result.isError should be true for missing args")
}

// TestMCP_UnknownToolCall_Integration calls a non-existent tool and verifies
// the server returns a JSON-RPC error.
func TestMCP_UnknownToolCall_Integration(t *testing.T) {
	srv := NewHTTPServer(nil, 0)
	srv.RegisterPod("caller-pod", "org1", nil, nil, "claude")

	ts := startTestServer(t, srv)

	resp := doToolCall(t, ts.URL, "caller-pod", "nonexistent_tool_xyz", `{}`)

	require.NotNil(t, resp.Error, "expected JSON-RPC error for unknown tool")
	assert.Equal(t, -32602, resp.Error.Code, "error code should be -32602")
	assert.Contains(t, resp.Error.Message, "Tool not found")
}

// perPodStatusProvider returns different statuses based on pod key.
type perPodStatusProvider struct {
	podStatuses map[string]fakeStatusProvider
}

func (p *perPodStatusProvider) GetPodStatus(podKey string) (string, string, int, bool) {
	if sp, ok := p.podStatuses[podKey]; ok {
		return sp.agentStatus, sp.podStatus, sp.shellPid, true
	}
	return "", "", 0, false
}

// TestMCP_MultiPodRouting_Integration registers two pods with different
// status providers and verifies correct routing per pod key.
func TestMCP_MultiPodRouting_Integration(t *testing.T) {
	srv := NewHTTPServer(nil, 0)
	srv.RegisterPod("caller-pod", "org1", nil, nil, "claude")

	provider := &perPodStatusProvider{
		podStatuses: map[string]fakeStatusProvider{
			"pod-alpha": {agentStatus: "waiting", podStatus: "running", shellPid: 100},
			"pod-beta":  {agentStatus: "executing", podStatus: "running", shellPid: 200},
		},
	}
	srv.SetStatusProvider(provider)
	ts := startTestServer(t, srv)

	resp1 := doToolCall(t, ts.URL, "caller-pod", "get_pod_status", `{"pod_key":"pod-alpha"}`)
	text1 := extractText(t, resp1)
	assertContains(t, text1, "Agent: waiting")

	resp2 := doToolCall(t, ts.URL, "caller-pod", "get_pod_status", `{"pod_key":"pod-beta"}`)
	text2 := extractText(t, resp2)
	assertContains(t, text2, "Agent: executing")
}

// TestMCP_ConcurrentToolCalls_Integration fires 10 concurrent get_pod_status
// calls and verifies all complete without error or race.
func TestMCP_ConcurrentToolCalls_Integration(t *testing.T) {
	srv := NewHTTPServer(nil, 0)
	srv.RegisterPod("caller-pod", "org1", nil, nil, "claude")
	srv.SetStatusProvider(&fakeStatusProvider{
		agentStatus: "idle",
		podStatus:   "running",
		shellPid:    42,
		knownPods:   map[string]bool{"target-pod": true},
	})
	ts := startTestServer(t, srv)

	const n = 10
	var wg sync.WaitGroup
	errs := make(chan error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			resp := doToolCall(t, ts.URL, "caller-pod", "get_pod_status",
				`{"pod_key":"target-pod"}`)
			if resp.Error != nil {
				errs <- fmt.Errorf("call %d: RPC error %d: %s",
					idx, resp.Error.Code, resp.Error.Message)
				return
			}
			rm, ok := resp.Result.(map[string]interface{})
			if !ok {
				errs <- fmt.Errorf("call %d: unexpected result type", idx)
				return
			}
			if _, hasErr := rm["isError"]; hasErr {
				errs <- fmt.Errorf("call %d: tool returned error", idx)
			}
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}

// TestMCP_MissingPodKeyHeader_Integration sends a request without X-Pod-Key
// header and verifies error response.
func TestMCP_MissingPodKeyHeader_Integration(t *testing.T) {
	srv := NewHTTPServer(nil, 0)
	ts := startTestServer(t, srv)

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`
	resp := doRawMCPCall(t, ts.URL, "", body)
	require.NotNil(t, resp.Error, "expected error when X-Pod-Key is missing")
	assert.Equal(t, -32600, resp.Error.Code)
}

// doRawMCPCall sends a raw JSON-RPC body to /mcp and returns MCPResponse.
func doRawMCPCall(t *testing.T, baseURL, podKey, rawBody string) MCPResponse {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, baseURL+"/mcp",
		strings.NewReader(rawBody))
	require.NoError(t, err)
	if podKey != "" {
		req.Header.Set("X-Pod-Key", podKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var mcpResp MCPResponse
	require.NoError(t, json.Unmarshal(data, &mcpResp))
	return mcpResp
}
