package extension

import (
	"encoding/json"
	"reflect"
	"testing"
)

// ---------------------------------------------------------------------------
// 1. InstalledMcpServer.ToMcpConfig()
// ---------------------------------------------------------------------------

func TestInstalledMcpServer_ToMcpConfig(t *testing.T) {
	tests := []struct {
		name   string
		server InstalledMcpServer
		want   map[string]interface{}
	}{
		{
			name: "stdio_no_market_item",
			server: InstalledMcpServer{
				TransportType: TransportTypeStdio,
				Command:       "npx",
				Args:          json.RawMessage(`["--yes","@modelcontextprotocol/server"]`),
			},
			want: map[string]interface{}{
				"command": "npx",
				"args":    []string{"--yes", "@modelcontextprotocol/server"},
			},
		},
		{
			name: "stdio_with_market_item_fallback",
			server: InstalledMcpServer{
				TransportType: TransportTypeStdio,
				Command:       "",
				Args:          nil,
				MarketItem: &McpMarketItem{
					Command:     "uvx",
					DefaultArgs: json.RawMessage(`["mcp-server-fetch"]`),
				},
			},
			want: map[string]interface{}{
				"command": "uvx",
				"args":    []string{"mcp-server-fetch"},
			},
		},
		{
			name: "stdio_with_args_override",
			server: InstalledMcpServer{
				TransportType: TransportTypeStdio,
				Command:       "npx",
				Args:          json.RawMessage(`["--custom-flag"]`),
				MarketItem: &McpMarketItem{
					Command:     "uvx",
					DefaultArgs: json.RawMessage(`["default-arg"]`),
				},
			},
			want: map[string]interface{}{
				"command": "npx",
				"args":    []string{"--custom-flag"},
			},
		},
		{
			name: "http_transport",
			server: InstalledMcpServer{
				TransportType: TransportTypeHTTP,
				HttpURL:       "https://example.com/mcp",
			},
			want: map[string]interface{}{
				"type": TransportTypeHTTP,
				"url":  "https://example.com/mcp",
			},
		},
		{
			name: "http_with_market_item_url_fallback",
			server: InstalledMcpServer{
				TransportType: TransportTypeHTTP,
				HttpURL:       "",
				MarketItem: &McpMarketItem{
					DefaultHttpURL: "https://default.example.com/mcp",
				},
			},
			want: map[string]interface{}{
				"type": TransportTypeHTTP,
				"url":  "https://default.example.com/mcp",
			},
		},
		{
			name: "sse_transport",
			server: InstalledMcpServer{
				TransportType: TransportTypeSSE,
				HttpURL:       "https://sse.example.com/events",
			},
			want: map[string]interface{}{
				"type": TransportTypeSSE,
				"url":  "https://sse.example.com/events",
			},
		},
		{
			name: "http_with_headers",
			server: InstalledMcpServer{
				TransportType: TransportTypeHTTP,
				HttpURL:       "https://example.com/mcp",
				HttpHeaders:   json.RawMessage(`{"Authorization":"Bearer tok","X-Custom":"val"}`),
			},
			want: map[string]interface{}{
				"type": TransportTypeHTTP,
				"url":  "https://example.com/mcp",
				"headers": map[string]string{
					"Authorization": "Bearer tok",
					"X-Custom":      "val",
				},
			},
		},
		{
			name: "with_env_vars",
			server: InstalledMcpServer{
				TransportType: TransportTypeStdio,
				Command:       "node",
				EnvVars:       json.RawMessage(`{"API_KEY":"secret123"}`),
			},
			want: map[string]interface{}{
				"command": "node",
				"env":     map[string]string{"API_KEY": "secret123"},
			},
		},
		{
			name: "empty_args_and_headers",
			server: InstalledMcpServer{
				TransportType: TransportTypeStdio,
				Command:       "node",
				Args:          json.RawMessage(`[]`),
			},
			want: map[string]interface{}{
				"command": "node",
			},
		},
		{
			name: "stdio_empty_everything",
			server: InstalledMcpServer{
				TransportType: TransportTypeStdio,
				Command:       "",
			},
			want: map[string]interface{}{
				"command": "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.server.ToMcpConfig()
			assertMapEqual(t, tt.want, got)
		})
	}
}

// assertMapEqual deeply compares two maps, handling both map[string]string and
// []string value types that come from json.Unmarshal.
func assertMapEqual(t *testing.T, want, got map[string]interface{}) {
	t.Helper()

	if len(want) != len(got) {
		t.Fatalf("map length mismatch: want %d keys %v, got %d keys %v", len(want), want, len(got), got)
	}

	for k, wv := range want {
		gv, ok := got[k]
		if !ok {
			t.Errorf("missing key %q in result", k)
			continue
		}

		switch wantVal := wv.(type) {
		case string:
			gotVal, ok := gv.(string)
			if !ok || gotVal != wantVal {
				t.Errorf("key %q: want %q, got %v", k, wantVal, gv)
			}
		case []string:
			gotVal, ok := gv.([]string)
			if !ok || !reflect.DeepEqual(gotVal, wantVal) {
				t.Errorf("key %q: want %v, got %v", k, wantVal, gv)
			}
		case map[string]string:
			gotVal, ok := gv.(map[string]string)
			if !ok || !reflect.DeepEqual(gotVal, wantVal) {
				t.Errorf("key %q: want %v, got %v", k, wantVal, gv)
			}
		default:
			if !reflect.DeepEqual(wv, gv) {
				t.Errorf("key %q: want %v, got %v", k, wv, gv)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// 1b. InstalledMcpServer.ToMcpConfig — invalid JSON resilience
// ---------------------------------------------------------------------------

func TestInstalledMcpServer_ToMcpConfig_InvalidJSON(t *testing.T) {
	t.Run("invalid_args_json", func(t *testing.T) {
		server := InstalledMcpServer{
			TransportType: TransportTypeStdio,
			Command:       "node",
			Args:          json.RawMessage(`{not valid json`),
		}
		config := server.ToMcpConfig()
		// Should not panic; args should be absent (empty after failed unmarshal)
		if config["command"] != "node" {
			t.Errorf("expected command 'node', got %v", config["command"])
		}
		if _, exists := config["args"]; exists {
			t.Error("expected no 'args' key when JSON is invalid")
		}
	})

	t.Run("invalid_http_headers_json", func(t *testing.T) {
		server := InstalledMcpServer{
			TransportType: TransportTypeHTTP,
			HttpURL:       "https://example.com/mcp",
			HttpHeaders:   json.RawMessage(`[broken`),
		}
		config := server.ToMcpConfig()
		if config["url"] != "https://example.com/mcp" {
			t.Errorf("expected url, got %v", config["url"])
		}
		// Headers should be absent due to invalid JSON
		if _, exists := config["headers"]; exists {
			t.Error("expected no 'headers' key when JSON is invalid")
		}
	})

	t.Run("invalid_env_vars_json", func(t *testing.T) {
		server := InstalledMcpServer{
			TransportType: TransportTypeStdio,
			Command:       "node",
			EnvVars:       json.RawMessage(`{malformed`),
		}
		config := server.ToMcpConfig()
		if config["command"] != "node" {
			t.Errorf("expected command 'node', got %v", config["command"])
		}
		// Env should be absent due to invalid JSON
		if _, exists := config["env"]; exists {
			t.Error("expected no 'env' key when JSON is invalid")
		}
	})

	t.Run("invalid_market_item_default_args_json", func(t *testing.T) {
		server := InstalledMcpServer{
			TransportType: TransportTypeStdio,
			Command:       "npx",
			Args:          nil, // empty, falls back to market item
			MarketItem: &McpMarketItem{
				Command:     "uvx",
				DefaultArgs: json.RawMessage(`{invalid`),
			},
		}
		config := server.ToMcpConfig()
		// Command comes from installed server (non-empty)
		if config["command"] != "npx" {
			t.Errorf("expected command 'npx', got %v", config["command"])
		}
		// Args from market item are also invalid, so no args key
		if _, exists := config["args"]; exists {
			t.Error("expected no 'args' key when market item default args JSON is invalid")
		}
	})
}
