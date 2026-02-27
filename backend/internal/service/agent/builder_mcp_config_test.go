package agent

import (
	"encoding/json"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// ---------------------------------------------------------------------------
// buildMcpConfig — detailed coverage for HTTP, stdio, unknown, env vars, etc.
// ---------------------------------------------------------------------------

func TestBuildMcpConfig_HttpServer_WithHeaders(t *testing.T) {
	builder := NewClaudeCodeBuilder()

	ctx := newMinimalBuildContext()
	ctx.TemplateCtx["mcp_port"] = nil
	ctx.McpServers = []*extension.InstalledMcpServer{
		{
			Slug:          "http-with-headers",
			TransportType: "http",
			HttpURL:       "https://headers.example.com/mcp",
			HttpHeaders:   json.RawMessage(`{"Authorization":"Bearer tok123","X-Custom":"val"}`),
			IsEnabled:     true,
		},
	}

	config := builder.buildMcpConfig(ctx)
	servers := config["mcpServers"].(map[string]interface{})

	srv, exists := servers["http-with-headers"]
	if !exists {
		t.Fatal("http-with-headers server should be present")
	}

	srvMap := srv.(map[string]interface{})
	if srvMap["type"] != "http" {
		t.Errorf("type = %q, want %q", srvMap["type"], "http")
	}
	if srvMap["url"] != "https://headers.example.com/mcp" {
		t.Errorf("url = %q, want %q", srvMap["url"], "https://headers.example.com/mcp")
	}

	headers, ok := srvMap["headers"].(map[string]string)
	if !ok {
		t.Fatalf("headers should be map[string]string, got %T", srvMap["headers"])
	}
	if headers["Authorization"] != "Bearer tok123" {
		t.Errorf("Authorization header = %q, want %q", headers["Authorization"], "Bearer tok123")
	}
	if headers["X-Custom"] != "val" {
		t.Errorf("X-Custom header = %q, want %q", headers["X-Custom"], "val")
	}
}

func TestBuildMcpConfig_HttpServer_EmptyHeaders(t *testing.T) {
	builder := NewClaudeCodeBuilder()

	ctx := newMinimalBuildContext()
	ctx.TemplateCtx["mcp_port"] = nil
	ctx.McpServers = []*extension.InstalledMcpServer{
		{
			Slug:          "http-no-headers",
			TransportType: "http",
			HttpURL:       "https://noheaders.example.com/mcp",
			HttpHeaders:   nil, // no headers
			IsEnabled:     true,
		},
	}

	config := builder.buildMcpConfig(ctx)
	servers := config["mcpServers"].(map[string]interface{})

	srv := servers["http-no-headers"].(map[string]interface{})
	if _, hasHeaders := srv["headers"]; hasHeaders {
		t.Error("headers key should NOT be present when HttpHeaders is nil")
	}
}

func TestBuildMcpConfig_HttpServer_InvalidHeaders(t *testing.T) {
	builder := NewClaudeCodeBuilder()

	ctx := newMinimalBuildContext()
	ctx.TemplateCtx["mcp_port"] = nil
	ctx.McpServers = []*extension.InstalledMcpServer{
		{
			Slug:          "http-bad-headers",
			TransportType: "http",
			HttpURL:       "https://badheaders.example.com/mcp",
			HttpHeaders:   json.RawMessage(`not-valid-json`),
			IsEnabled:     true,
		},
	}

	config := builder.buildMcpConfig(ctx)
	servers := config["mcpServers"].(map[string]interface{})

	srv := servers["http-bad-headers"].(map[string]interface{})
	// Should still exist, but without headers (unmarshal silently fails)
	if _, hasHeaders := srv["headers"]; hasHeaders {
		t.Error("headers key should NOT be present when HttpHeaders is invalid JSON")
	}
}

func TestBuildMcpConfig_StdioServer_WithArgs(t *testing.T) {
	builder := NewClaudeCodeBuilder()

	ctx := newMinimalBuildContext()
	ctx.TemplateCtx["mcp_port"] = nil
	ctx.McpServers = []*extension.InstalledMcpServer{
		{
			Slug:          "stdio-srv",
			TransportType: "stdio",
			Command:       "npx",
			Args:          json.RawMessage(`["-y","@modelcontextprotocol/server-github"]`),
			IsEnabled:     true,
		},
	}

	config := builder.buildMcpConfig(ctx)
	servers := config["mcpServers"].(map[string]interface{})

	srv := servers["stdio-srv"].(map[string]interface{})
	if srv["type"] != "stdio" {
		t.Errorf("type = %q, want %q", srv["type"], "stdio")
	}
	if srv["command"] != "npx" {
		t.Errorf("command = %q, want %q", srv["command"], "npx")
	}

	args, ok := srv["args"].([]string)
	if !ok {
		t.Fatalf("args should be []string, got %T", srv["args"])
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if args[0] != "-y" {
		t.Errorf("args[0] = %q, want %q", args[0], "-y")
	}
}

func TestBuildMcpConfig_StdioServer_NoArgs(t *testing.T) {
	builder := NewClaudeCodeBuilder()

	ctx := newMinimalBuildContext()
	ctx.TemplateCtx["mcp_port"] = nil
	ctx.McpServers = []*extension.InstalledMcpServer{
		{
			Slug:          "stdio-no-args",
			TransportType: "stdio",
			Command:       "my-mcp-server",
			Args:          nil, // no args
			IsEnabled:     true,
		},
	}

	config := builder.buildMcpConfig(ctx)
	servers := config["mcpServers"].(map[string]interface{})

	srv := servers["stdio-no-args"].(map[string]interface{})
	if _, hasArgs := srv["args"]; hasArgs {
		t.Error("args key should NOT be present when Args is nil")
	}
}

func TestBuildMcpConfig_StdioServer_InvalidArgs(t *testing.T) {
	builder := NewClaudeCodeBuilder()

	ctx := newMinimalBuildContext()
	ctx.TemplateCtx["mcp_port"] = nil
	ctx.McpServers = []*extension.InstalledMcpServer{
		{
			Slug:          "stdio-bad-args",
			TransportType: "stdio",
			Command:       "my-mcp-server",
			Args:          json.RawMessage(`not-valid-json`),
			IsEnabled:     true,
		},
	}

	config := builder.buildMcpConfig(ctx)
	servers := config["mcpServers"].(map[string]interface{})

	srv := servers["stdio-bad-args"].(map[string]interface{})
	// Should still exist, but without args (unmarshal silently fails)
	if _, hasArgs := srv["args"]; hasArgs {
		t.Error("args key should NOT be present when Args is invalid JSON")
	}
}

func TestBuildMcpConfig_UnknownTransportType(t *testing.T) {
	builder := NewClaudeCodeBuilder()

	ctx := newMinimalBuildContext()
	ctx.TemplateCtx["mcp_port"] = nil
	ctx.McpServers = []*extension.InstalledMcpServer{
		{
			Slug:          "unknown-type",
			TransportType: "grpc", // unknown
			IsEnabled:     true,
		},
		{
			Slug:          "valid-http",
			TransportType: "http",
			HttpURL:       "https://valid.example.com/mcp",
			IsEnabled:     true,
		},
	}

	config := builder.buildMcpConfig(ctx)
	servers := config["mcpServers"].(map[string]interface{})

	if _, exists := servers["unknown-type"]; exists {
		t.Error("unknown transport type server should NOT be present")
	}
	if _, exists := servers["valid-http"]; !exists {
		t.Error("valid-http server should be present")
	}
}

func TestBuildMcpConfig_EnvVars(t *testing.T) {
	builder := NewClaudeCodeBuilder()

	t.Run("http server with env vars", func(t *testing.T) {
		ctx := newMinimalBuildContext()
		ctx.TemplateCtx["mcp_port"] = nil
		ctx.McpServers = []*extension.InstalledMcpServer{
			{
				Slug:          "http-with-env",
				TransportType: "http",
				HttpURL:       "https://env.example.com/mcp",
				EnvVars:       json.RawMessage(`{"API_KEY":"secret123","REGION":"us-east"}`),
				IsEnabled:     true,
			},
		}

		config := builder.buildMcpConfig(ctx)
		servers := config["mcpServers"].(map[string]interface{})

		srv := servers["http-with-env"].(map[string]interface{})
		envVars, ok := srv["env"].(map[string]string)
		if !ok {
			t.Fatalf("env should be map[string]string, got %T", srv["env"])
		}
		if envVars["API_KEY"] != "secret123" {
			t.Errorf("API_KEY = %q, want %q", envVars["API_KEY"], "secret123")
		}
		if envVars["REGION"] != "us-east" {
			t.Errorf("REGION = %q, want %q", envVars["REGION"], "us-east")
		}
	})

	t.Run("stdio server with env vars", func(t *testing.T) {
		ctx := newMinimalBuildContext()
		ctx.TemplateCtx["mcp_port"] = nil
		ctx.McpServers = []*extension.InstalledMcpServer{
			{
				Slug:          "stdio-with-env",
				TransportType: "stdio",
				Command:       "my-tool",
				EnvVars:       json.RawMessage(`{"TOKEN":"abc"}`),
				IsEnabled:     true,
			},
		}

		config := builder.buildMcpConfig(ctx)
		servers := config["mcpServers"].(map[string]interface{})

		srv := servers["stdio-with-env"].(map[string]interface{})
		envVars, ok := srv["env"].(map[string]string)
		if !ok {
			t.Fatalf("env should be map[string]string, got %T", srv["env"])
		}
		if envVars["TOKEN"] != "abc" {
			t.Errorf("TOKEN = %q, want %q", envVars["TOKEN"], "abc")
		}
	})

	t.Run("server with empty env vars", func(t *testing.T) {
		ctx := newMinimalBuildContext()
		ctx.TemplateCtx["mcp_port"] = nil
		ctx.McpServers = []*extension.InstalledMcpServer{
			{
				Slug:          "no-env",
				TransportType: "http",
				HttpURL:       "https://noenv.example.com/mcp",
				EnvVars:       nil, // no env vars
				IsEnabled:     true,
			},
		}

		config := builder.buildMcpConfig(ctx)
		servers := config["mcpServers"].(map[string]interface{})

		srv := servers["no-env"].(map[string]interface{})
		if _, hasEnv := srv["env"]; hasEnv {
			t.Error("env key should NOT be present when EnvVars is nil")
		}
	})

	t.Run("server with invalid env vars JSON", func(t *testing.T) {
		ctx := newMinimalBuildContext()
		ctx.TemplateCtx["mcp_port"] = nil
		ctx.McpServers = []*extension.InstalledMcpServer{
			{
				Slug:          "bad-env",
				TransportType: "http",
				HttpURL:       "https://badenv.example.com/mcp",
				EnvVars:       json.RawMessage(`not-valid-json`),
				IsEnabled:     true,
			},
		}

		config := builder.buildMcpConfig(ctx)
		servers := config["mcpServers"].(map[string]interface{})

		srv := servers["bad-env"].(map[string]interface{})
		if _, hasEnv := srv["env"]; hasEnv {
			t.Error("env key should NOT be present when EnvVars is invalid JSON")
		}
	})

	t.Run("server with empty env vars object", func(t *testing.T) {
		ctx := newMinimalBuildContext()
		ctx.TemplateCtx["mcp_port"] = nil
		ctx.McpServers = []*extension.InstalledMcpServer{
			{
				Slug:          "empty-env",
				TransportType: "http",
				HttpURL:       "https://emptyenv.example.com/mcp",
				EnvVars:       json.RawMessage(`{}`),
				IsEnabled:     true,
			},
		}

		config := builder.buildMcpConfig(ctx)
		servers := config["mcpServers"].(map[string]interface{})

		srv := servers["empty-env"].(map[string]interface{})
		if _, hasEnv := srv["env"]; hasEnv {
			t.Error("env key should NOT be present when EnvVars is empty object")
		}
	})
}

func TestBuildMcpConfig_McpPortNotInt(t *testing.T) {
	builder := NewClaudeCodeBuilder()

	// mcp_port is present but not an int (e.g. string)
	ctx := newMinimalBuildContext()
	ctx.TemplateCtx["mcp_port"] = "not-an-int"

	config := builder.buildMcpConfig(ctx)
	servers := config["mcpServers"].(map[string]interface{})

	if _, exists := servers["agentsmesh"]; exists {
		t.Error("agentsmesh server should NOT be present when mcp_port is not an int")
	}
}

func TestBuildMcpConfig_McpPortMissing(t *testing.T) {
	builder := NewClaudeCodeBuilder()

	// mcp_port key not present at all in TemplateCtx
	ctx := newMinimalBuildContext()
	delete(ctx.TemplateCtx, "mcp_port")

	config := builder.buildMcpConfig(ctx)
	servers := config["mcpServers"].(map[string]interface{})

	if _, exists := servers["agentsmesh"]; exists {
		t.Error("agentsmesh server should NOT be present when mcp_port key is absent")
	}
}

func TestBuildMcpConfig_McpPortNegative(t *testing.T) {
	builder := NewClaudeCodeBuilder()

	ctx := newMinimalBuildContext()
	ctx.TemplateCtx["mcp_port"] = -1

	config := builder.buildMcpConfig(ctx)
	servers := config["mcpServers"].(map[string]interface{})

	if _, exists := servers["agentsmesh"]; exists {
		t.Error("agentsmesh server should NOT be present when mcp_port is negative")
	}
}

func TestBuildMcpConfig_HttpEmptyHeadersJSON(t *testing.T) {
	builder := NewClaudeCodeBuilder()

	ctx := newMinimalBuildContext()
	ctx.TemplateCtx["mcp_port"] = nil
	ctx.McpServers = []*extension.InstalledMcpServer{
		{
			Slug:          "http-empty-headers-json",
			TransportType: "http",
			HttpURL:       "https://emptyheaders.example.com/mcp",
			HttpHeaders:   json.RawMessage(`{}`), // valid JSON but empty
			IsEnabled:     true,
		},
	}

	config := builder.buildMcpConfig(ctx)
	servers := config["mcpServers"].(map[string]interface{})

	srv := servers["http-empty-headers-json"].(map[string]interface{})
	if _, hasHeaders := srv["headers"]; hasHeaders {
		t.Error("headers key should NOT be present when HttpHeaders JSON is empty object")
	}
}

func TestBuildMcpConfig_StdioEmptyArgsJSON(t *testing.T) {
	builder := NewClaudeCodeBuilder()

	ctx := newMinimalBuildContext()
	ctx.TemplateCtx["mcp_port"] = nil
	ctx.McpServers = []*extension.InstalledMcpServer{
		{
			Slug:          "stdio-empty-args-json",
			TransportType: "stdio",
			Command:       "tool",
			Args:          json.RawMessage(`[]`), // valid JSON but empty array
			IsEnabled:     true,
		},
	}

	config := builder.buildMcpConfig(ctx)
	servers := config["mcpServers"].(map[string]interface{})

	srv := servers["stdio-empty-args-json"].(map[string]interface{})
	// ToMcpConfig() correctly omits args when the unmarshaled slice is empty,
	// since there's no point sending an empty args array.
	if _, hasArgs := srv["args"]; hasArgs {
		t.Error("args key should NOT be present when Args unmarshal to empty slice")
	}
}

// ---------------------------------------------------------------------------
// MarketItem fallback — verifies ToMcpConfig delegation picks up defaults
// ---------------------------------------------------------------------------

func TestBuildMcpConfig_MarketItemFallback_StdioCommandAndArgs(t *testing.T) {
	builder := NewClaudeCodeBuilder()

	ctx := newMinimalBuildContext()
	ctx.TemplateCtx["mcp_port"] = nil
	ctx.McpServers = []*extension.InstalledMcpServer{
		{
			Slug:          "market-stdio",
			TransportType: "stdio",
			Command:       "",   // empty — should fallback to MarketItem
			Args:          nil,  // nil — should fallback to MarketItem
			IsEnabled:     true,
			MarketItem: &extension.McpMarketItem{
				Command:     "npx",
				DefaultArgs: json.RawMessage(`["-y","@mcp/server"]`),
			},
		},
	}

	config := builder.buildMcpConfig(ctx)
	servers := config["mcpServers"].(map[string]interface{})

	srv, exists := servers["market-stdio"]
	if !exists {
		t.Fatal("market-stdio server should be present")
	}

	srvMap := srv.(map[string]interface{})
	if srvMap["command"] != "npx" {
		t.Errorf("command = %q, want %q (from MarketItem fallback)", srvMap["command"], "npx")
	}

	args, ok := srvMap["args"].([]string)
	if !ok {
		t.Fatalf("args should be []string, got %T", srvMap["args"])
	}
	if len(args) != 2 || args[0] != "-y" || args[1] != "@mcp/server" {
		t.Errorf("args = %v, want [-y @mcp/server] (from MarketItem fallback)", args)
	}

	if srvMap["type"] != "stdio" {
		t.Errorf("type = %q, want %q", srvMap["type"], "stdio")
	}
}

func TestBuildMcpConfig_MarketItemFallback_HttpURL(t *testing.T) {
	builder := NewClaudeCodeBuilder()

	ctx := newMinimalBuildContext()
	ctx.TemplateCtx["mcp_port"] = nil
	ctx.McpServers = []*extension.InstalledMcpServer{
		{
			Slug:          "market-http",
			TransportType: "http",
			HttpURL:       "", // empty — should fallback to MarketItem
			IsEnabled:     true,
			MarketItem: &extension.McpMarketItem{
				DefaultHttpURL: "https://market-default.example.com/mcp",
			},
		},
	}

	config := builder.buildMcpConfig(ctx)
	servers := config["mcpServers"].(map[string]interface{})

	srv, exists := servers["market-http"]
	if !exists {
		t.Fatal("market-http server should be present")
	}

	srvMap := srv.(map[string]interface{})
	if srvMap["url"] != "https://market-default.example.com/mcp" {
		t.Errorf("url = %q, want %q (from MarketItem fallback)", srvMap["url"], "https://market-default.example.com/mcp")
	}

	if srvMap["type"] != "http" {
		t.Errorf("type = %q, want %q", srvMap["type"], "http")
	}
}

func TestBuildMcpConfig_MarketItemFallback_UserOverridesTakesPrecedence(t *testing.T) {
	builder := NewClaudeCodeBuilder()

	ctx := newMinimalBuildContext()
	ctx.TemplateCtx["mcp_port"] = nil
	ctx.McpServers = []*extension.InstalledMcpServer{
		{
			Slug:          "user-override",
			TransportType: "stdio",
			Command:       "my-custom-cmd",
			Args:          json.RawMessage(`["--flag"]`),
			IsEnabled:     true,
			MarketItem: &extension.McpMarketItem{
				Command:     "npx",
				DefaultArgs: json.RawMessage(`["-y","@mcp/server"]`),
			},
		},
	}

	config := builder.buildMcpConfig(ctx)
	servers := config["mcpServers"].(map[string]interface{})

	srv := servers["user-override"].(map[string]interface{})
	// User's own command/args should take precedence over MarketItem defaults
	if srv["command"] != "my-custom-cmd" {
		t.Errorf("command = %q, want %q (user override should take precedence)", srv["command"], "my-custom-cmd")
	}

	args, ok := srv["args"].([]string)
	if !ok {
		t.Fatalf("args should be []string, got %T", srv["args"])
	}
	if len(args) != 1 || args[0] != "--flag" {
		t.Errorf("args = %v, want [--flag] (user override should take precedence)", args)
	}
}

func TestBuildMcpConfig_SSETransportType(t *testing.T) {
	builder := NewClaudeCodeBuilder()

	ctx := newMinimalBuildContext()
	ctx.TemplateCtx["mcp_port"] = nil
	ctx.McpServers = []*extension.InstalledMcpServer{
		{
			Slug:          "sse-server",
			TransportType: "sse",
			HttpURL:       "https://sse.example.com/mcp",
			IsEnabled:     true,
		},
	}

	config := builder.buildMcpConfig(ctx)
	servers := config["mcpServers"].(map[string]interface{})

	srv, exists := servers["sse-server"]
	if !exists {
		t.Fatal("sse-server should be present")
	}

	srvMap := srv.(map[string]interface{})
	if srvMap["type"] != "sse" {
		t.Errorf("type = %q, want %q", srvMap["type"], "sse")
	}
	if srvMap["url"] != "https://sse.example.com/mcp" {
		t.Errorf("url = %q, want %q", srvMap["url"], "https://sse.example.com/mcp")
	}
}
