package mcp

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkHTTPServerHandleMCP(b *testing.B) {
	server := NewHTTPServer(nil, 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	bodyStr := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := bytes.NewBufferString(bodyStr)
		req := httptest.NewRequest(http.MethodPost, "/mcp", body)
		req.Header.Set("X-Pod-Key", "test-pod")
		rec := httptest.NewRecorder()

		server.handleMCP(rec, req)
	}
}

func BenchmarkHTTPServerRegisterPod(b *testing.B) {
	server := NewHTTPServer(nil, 9090)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.RegisterPod("test-pod", "test-org", nil, nil, "claude")
	}
}
