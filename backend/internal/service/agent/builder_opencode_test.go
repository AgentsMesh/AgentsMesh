package agent

import (
	"encoding/json"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

func newOpenCodeBuildContext() *BuildContext {
	return &BuildContext{
		Request: &ConfigBuildRequest{
			MCPPort: 0,
			PodKey:  "pod-1",
		},
		AgentType: &agent.AgentType{
			Slug:          OpenCodeSlug,
			LaunchCommand: "opencode",
		},
		Config:      agent.ConfigValues{},
		Credentials: agent.EncryptedCredentials{},
		TemplateCtx: map[string]interface{}{
			"config":   agent.ConfigValues{},
			"mcp_port": 0,
			"pod_key":  "pod-1",
		},
	}
}

// parseOpenCodeConfig extracts the OPENCODE_CONFIG_CONTENT JSON from env vars.
func parseOpenCodeConfig(t *testing.T, envVars map[string]string) map[string]interface{} {
	t.Helper()
	raw, ok := envVars["OPENCODE_CONFIG_CONTENT"]
	if !ok {
		t.Fatal("OPENCODE_CONFIG_CONTENT not set")
	}
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		t.Fatalf("Failed to parse OPENCODE_CONFIG_CONTENT: %v", err)
	}
	return config
}

func TestOpenCodeBuilder_BuildEnvVars_McpDisabled(t *testing.T) {
	builder := NewOpenCodeBuilder()
	ctx := newOpenCodeBuildContext()
	ctx.Config["mcp_enabled"] = false

	envVars, err := builder.BuildEnvVars(ctx)
	if err != nil {
		t.Fatalf("BuildEnvVars failed: %v", err)
	}
	if _, ok := envVars["OPENCODE_CONFIG_CONTENT"]; ok {
		t.Error("OPENCODE_CONFIG_CONTENT should not be set when mcp_enabled=false and no other config")
	}
}

func TestOpenCodeBuilder_BuildEnvVars_McpWithAgentsMesh(t *testing.T) {
	builder := NewOpenCodeBuilder()
	ctx := newOpenCodeBuildContext()
	ctx.Config["mcp_enabled"] = true
	ctx.TemplateCtx["mcp_port"] = 9999
	ctx.TemplateCtx["pod_key"] = "test-pod-key"

	envVars, err := builder.BuildEnvVars(ctx)
	if err != nil {
		t.Fatalf("BuildEnvVars failed: %v", err)
	}

	config := parseOpenCodeConfig(t, envVars)
	mcp, ok := config["mcp"].(map[string]interface{})
	if !ok {
		t.Fatal("mcp key missing or not a map")
	}

	am, ok := mcp["agentsmesh"].(map[string]interface{})
	if !ok {
		t.Fatal("agentsmesh server missing")
	}
	if am["type"] != "remote" {
		t.Errorf("type = %v, want remote", am["type"])
	}
	if am["url"] != "http://127.0.0.1:9999/mcp" {
		t.Errorf("url = %v, want http://127.0.0.1:9999/mcp", am["url"])
	}

	headers, ok := am["headers"].(map[string]interface{})
	if !ok {
		t.Fatal("headers missing")
	}
	if headers["X-Pod-Key"] != "test-pod-key" {
		t.Errorf("X-Pod-Key = %v, want test-pod-key", headers["X-Pod-Key"])
	}
}

func TestOpenCodeBuilder_BuildEnvVars_McpWithUserServers(t *testing.T) {
	builder := NewOpenCodeBuilder()
	ctx := newOpenCodeBuildContext()
	ctx.Config["mcp_enabled"] = true
	ctx.TemplateCtx["mcp_port"] = nil // no agentsmesh server
	ctx.McpServers = []*extension.InstalledMcpServer{
		{
			Slug:          "http-server",
			TransportType: "http",
			HttpURL:       "https://example.com/mcp",
			HttpHeaders:   json.RawMessage(`{"Authorization":"Bearer tok"}`),
			IsEnabled:     true,
		},
		{
			Slug:          "stdio-server",
			TransportType: "stdio",
			Command:       "npx",
			Args:          json.RawMessage(`["-y","@my/server"]`),
			IsEnabled:     true,
		},
	}

	envVars, err := builder.BuildEnvVars(ctx)
	if err != nil {
		t.Fatalf("BuildEnvVars failed: %v", err)
	}

	config := parseOpenCodeConfig(t, envVars)
	mcp := config["mcp"].(map[string]interface{})

	// HTTP server
	httpSrv, ok := mcp["http-server"].(map[string]interface{})
	if !ok {
		t.Fatal("http-server missing")
	}
	if httpSrv["url"] != "https://example.com/mcp" {
		t.Errorf("http url = %v", httpSrv["url"])
	}

	// Stdio server
	stdioSrv, ok := mcp["stdio-server"].(map[string]interface{})
	if !ok {
		t.Fatal("stdio-server missing")
	}
	if stdioSrv["command"] != "npx" {
		t.Errorf("stdio command = %v", stdioSrv["command"])
	}
}

func TestOpenCodeBuilder_BuildEnvVars_DisabledServersFiltered(t *testing.T) {
	builder := NewOpenCodeBuilder()
	ctx := newOpenCodeBuildContext()
	ctx.Config["mcp_enabled"] = true
	ctx.TemplateCtx["mcp_port"] = nil
	ctx.McpServers = []*extension.InstalledMcpServer{
		{Slug: "enabled", TransportType: "http", HttpURL: "https://a.com", IsEnabled: true},
		{Slug: "disabled", TransportType: "http", HttpURL: "https://b.com", IsEnabled: false},
	}

	envVars, err := builder.BuildEnvVars(ctx)
	if err != nil {
		t.Fatalf("BuildEnvVars failed: %v", err)
	}

	config := parseOpenCodeConfig(t, envVars)
	mcp := config["mcp"].(map[string]interface{})

	if _, ok := mcp["enabled"]; !ok {
		t.Error("enabled server should be present")
	}
	if _, ok := mcp["disabled"]; ok {
		t.Error("disabled server should be filtered out")
	}
}

func TestOpenCodeBuilder_BuildEnvVars_UnsupportedTransportFiltered(t *testing.T) {
	builder := NewOpenCodeBuilder()
	ctx := newOpenCodeBuildContext()
	ctx.Config["mcp_enabled"] = true
	ctx.TemplateCtx["mcp_port"] = nil
	ctx.McpServers = []*extension.InstalledMcpServer{
		{Slug: "valid", TransportType: "sse", HttpURL: "https://sse.com", IsEnabled: true},
		{Slug: "unknown", TransportType: "grpc", IsEnabled: true},
	}

	envVars, err := builder.BuildEnvVars(ctx)
	if err != nil {
		t.Fatalf("BuildEnvVars failed: %v", err)
	}

	config := parseOpenCodeConfig(t, envVars)
	mcp := config["mcp"].(map[string]interface{})

	if _, ok := mcp["valid"]; !ok {
		t.Error("sse server should be present")
	}
	if _, ok := mcp["unknown"]; ok {
		t.Error("grpc server should be filtered out")
	}
}

func TestOpenCodeBuilder_BuildEnvVars_SkipPermissions(t *testing.T) {
	builder := NewOpenCodeBuilder()
	ctx := newOpenCodeBuildContext()
	ctx.Config["skip_permissions"] = true

	envVars, err := builder.BuildEnvVars(ctx)
	if err != nil {
		t.Fatalf("BuildEnvVars failed: %v", err)
	}

	config := parseOpenCodeConfig(t, envVars)
	perm, ok := config["permission"].(map[string]interface{})
	if !ok {
		t.Fatal("permission key missing")
	}
	if perm["*"] != "allow" {
		t.Errorf("permission[*] = %v, want allow", perm["*"])
	}
}

func TestOpenCodeBuilder_BuildEnvVars_McpAndPermissionsCombined(t *testing.T) {
	builder := NewOpenCodeBuilder()
	ctx := newOpenCodeBuildContext()
	ctx.Config["mcp_enabled"] = true
	ctx.Config["skip_permissions"] = true
	ctx.TemplateCtx["mcp_port"] = 8080

	envVars, err := builder.BuildEnvVars(ctx)
	if err != nil {
		t.Fatalf("BuildEnvVars failed: %v", err)
	}

	config := parseOpenCodeConfig(t, envVars)

	// Both keys should be present
	if _, ok := config["mcp"]; !ok {
		t.Error("mcp key should be present")
	}
	if _, ok := config["permission"]; !ok {
		t.Error("permission key should be present")
	}
}

func TestOpenCodeBuilder_BuildLaunchArgs_ModelFromConfig(t *testing.T) {
	builder := NewOpenCodeBuilder()
	ctx := newOpenCodeBuildContext()
	ctx.Config["model"] = "anthropic/claude-sonnet-4"

	args, err := builder.BuildLaunchArgs(ctx)
	if err != nil {
		t.Fatalf("BuildLaunchArgs failed: %v", err)
	}

	found := false
	for i, a := range args {
		if a == "--model" && i+1 < len(args) && args[i+1] == "anthropic/claude-sonnet-4" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected --model anthropic/claude-sonnet-4 in args, got %v", args)
	}
}

func TestOpenCodeBuilder_BuildLaunchArgs_FallbackToFirstModel(t *testing.T) {
	builder := NewOpenCodeBuilder()
	ctx := newOpenCodeBuildContext()
	// model is empty, but models list exists
	ctx.Config["models"] = []interface{}{"anthropic/claude-opus-4", "openai/gpt-4o"}

	args, err := builder.BuildLaunchArgs(ctx)
	if err != nil {
		t.Fatalf("BuildLaunchArgs failed: %v", err)
	}

	found := false
	for i, a := range args {
		if a == "--model" && i+1 < len(args) && args[i+1] == "anthropic/claude-opus-4" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected --model anthropic/claude-opus-4 (first in list), got %v", args)
	}
}

func TestOpenCodeBuilder_BuildLaunchArgs_ExplicitModelOverridesList(t *testing.T) {
	builder := NewOpenCodeBuilder()
	ctx := newOpenCodeBuildContext()
	ctx.Config["model"] = "openai/gpt-4o"
	ctx.Config["models"] = []interface{}{"anthropic/claude-opus-4", "openai/gpt-4o"}

	args, err := builder.BuildLaunchArgs(ctx)
	if err != nil {
		t.Fatalf("BuildLaunchArgs failed: %v", err)
	}

	// Explicit model should take precedence over first in list
	found := false
	for i, a := range args {
		if a == "--model" && i+1 < len(args) && args[i+1] == "openai/gpt-4o" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected --model openai/gpt-4o (explicit), got %v", args)
	}
}

func TestOpenCodeBuilder_BuildLaunchArgs_NoModel(t *testing.T) {
	builder := NewOpenCodeBuilder()
	ctx := newOpenCodeBuildContext()

	args, err := builder.BuildLaunchArgs(ctx)
	if err != nil {
		t.Fatalf("BuildLaunchArgs failed: %v", err)
	}

	for _, a := range args {
		if a == "--model" {
			t.Error("--model should not be present when no model configured")
		}
	}
}
