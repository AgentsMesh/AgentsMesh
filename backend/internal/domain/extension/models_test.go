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

// ---------------------------------------------------------------------------
// 2. InstalledSkill.GetEffectiveSha / GetEffectiveStorageKey / GetEffectivePackageSize
// ---------------------------------------------------------------------------

func TestInstalledSkill_GetEffectiveSha(t *testing.T) {
	pinnedVersion := 5

	tests := []struct {
		name  string
		skill InstalledSkill
		want  string
	}{
		{
			name: "market_tracking_latest",
			skill: InstalledSkill{
				InstallSource: InstallSourceMarket,
				PinnedVersion: nil,
				ContentSha:    "own-sha",
				MarketItem:    &SkillMarketItem{ContentSha: "market-sha"},
			},
			want: "market-sha",
		},
		{
			name: "market_pinned_version",
			skill: InstalledSkill{
				InstallSource: InstallSourceMarket,
				PinnedVersion: &pinnedVersion,
				ContentSha:    "own-sha",
				MarketItem:    &SkillMarketItem{ContentSha: "market-sha"},
			},
			want: "own-sha",
		},
		{
			name: "market_no_market_item",
			skill: InstalledSkill{
				InstallSource: InstallSourceMarket,
				PinnedVersion: nil,
				ContentSha:    "own-sha",
				MarketItem:    nil,
			},
			want: "own-sha",
		},
		{
			name: "github_source",
			skill: InstalledSkill{
				InstallSource: InstallSourceGitHub,
				ContentSha:    "github-sha",
				MarketItem:    &SkillMarketItem{ContentSha: "market-sha"},
			},
			want: "github-sha",
		},
		{
			name: "upload_source",
			skill: InstalledSkill{
				InstallSource: InstallSourceUpload,
				ContentSha:    "upload-sha",
			},
			want: "upload-sha",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.skill.GetEffectiveSha()
			if got != tt.want {
				t.Errorf("GetEffectiveSha() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInstalledSkill_GetEffectiveStorageKey(t *testing.T) {
	pinnedVersion := 5

	tests := []struct {
		name  string
		skill InstalledSkill
		want  string
	}{
		{
			name: "market_tracking_latest",
			skill: InstalledSkill{
				InstallSource: InstallSourceMarket,
				PinnedVersion: nil,
				StorageKey:    "own-key",
				MarketItem:    &SkillMarketItem{StorageKey: "market-key"},
			},
			want: "market-key",
		},
		{
			name: "market_pinned_version",
			skill: InstalledSkill{
				InstallSource: InstallSourceMarket,
				PinnedVersion: &pinnedVersion,
				StorageKey:    "own-key",
				MarketItem:    &SkillMarketItem{StorageKey: "market-key"},
			},
			want: "own-key",
		},
		{
			name: "market_no_market_item",
			skill: InstalledSkill{
				InstallSource: InstallSourceMarket,
				PinnedVersion: nil,
				StorageKey:    "own-key",
				MarketItem:    nil,
			},
			want: "own-key",
		},
		{
			name: "github_source",
			skill: InstalledSkill{
				InstallSource: InstallSourceGitHub,
				StorageKey:    "github-key",
				MarketItem:    &SkillMarketItem{StorageKey: "market-key"},
			},
			want: "github-key",
		},
		{
			name: "upload_source",
			skill: InstalledSkill{
				InstallSource: InstallSourceUpload,
				StorageKey:    "upload-key",
			},
			want: "upload-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.skill.GetEffectiveStorageKey()
			if got != tt.want {
				t.Errorf("GetEffectiveStorageKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInstalledSkill_GetEffectivePackageSize(t *testing.T) {
	pinnedVersion := 5

	tests := []struct {
		name  string
		skill InstalledSkill
		want  int64
	}{
		{
			name: "market_tracking_latest",
			skill: InstalledSkill{
				InstallSource: InstallSourceMarket,
				PinnedVersion: nil,
				PackageSize:   100,
				MarketItem:    &SkillMarketItem{PackageSize: 999},
			},
			want: 999,
		},
		{
			name: "market_pinned_version",
			skill: InstalledSkill{
				InstallSource: InstallSourceMarket,
				PinnedVersion: &pinnedVersion,
				PackageSize:   100,
				MarketItem:    &SkillMarketItem{PackageSize: 999},
			},
			want: 100,
		},
		{
			name: "market_no_market_item",
			skill: InstalledSkill{
				InstallSource: InstallSourceMarket,
				PinnedVersion: nil,
				PackageSize:   100,
				MarketItem:    nil,
			},
			want: 100,
		},
		{
			name: "github_source",
			skill: InstalledSkill{
				InstallSource: InstallSourceGitHub,
				PackageSize:   200,
				MarketItem:    &SkillMarketItem{PackageSize: 999},
			},
			want: 200,
		},
		{
			name: "upload_source",
			skill: InstalledSkill{
				InstallSource: InstallSourceUpload,
				PackageSize:   300,
			},
			want: 300,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.skill.GetEffectivePackageSize()
			if got != tt.want {
				t.Errorf("GetEffectivePackageSize() = %d, want %d", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 3. SkillRegistry.IsPlatformLevel()
// ---------------------------------------------------------------------------

func TestSkillRegistry_IsPlatformLevel(t *testing.T) {
	orgID := int64(42)

	tests := []struct {
		name   string
		source SkillRegistry
		want   bool
	}{
		{
			name:   "platform_level",
			source: SkillRegistry{OrganizationID: nil},
			want:   true,
		},
		{
			name:   "org_level",
			source: SkillRegistry{OrganizationID: &orgID},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.source.IsPlatformLevel()
			if got != tt.want {
				t.Errorf("IsPlatformLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 4. TableName() methods
// ---------------------------------------------------------------------------

func TestTableNames(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		want      string
	}{
		{"InstalledMcpServer", InstalledMcpServer{}.TableName(), "installed_mcp_servers"},
		{"InstalledSkill", InstalledSkill{}.TableName(), "installed_skills"},
		{"SkillRegistry", SkillRegistry{}.TableName(), "skill_registries"},
		{"SkillMarketItem", SkillMarketItem{}.TableName(), "skill_market_items"},
		{"McpMarketItem", McpMarketItem{}.TableName(), "mcp_market_items"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.tableName != tt.want {
				t.Errorf("TableName() = %q, want %q", tt.tableName, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 5. Constants
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// 4b. SkillRegistryOverride.TableName
// ---------------------------------------------------------------------------

func TestSkillRegistryOverride_TableName(t *testing.T) {
	override := SkillRegistryOverride{}
	if override.TableName() != "skill_registry_overrides" {
		t.Errorf("TableName() = %q, want %q", override.TableName(), "skill_registry_overrides")
	}
}

// ---------------------------------------------------------------------------
// 6. McpMarketItem.GetAgentTypeFilter
// ---------------------------------------------------------------------------

func TestMcpMarketItem_GetAgentTypeFilter_Valid(t *testing.T) {
	item := McpMarketItem{
		AgentTypeFilter: json.RawMessage(`["claude-code","aider"]`),
	}
	result := item.GetAgentTypeFilter()
	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
	if result[0] != "claude-code" {
		t.Errorf("expected first item 'claude-code', got %q", result[0])
	}
	if result[1] != "aider" {
		t.Errorf("expected second item 'aider', got %q", result[1])
	}
}

func TestMcpMarketItem_GetAgentTypeFilter_Empty(t *testing.T) {
	item := McpMarketItem{
		AgentTypeFilter: nil,
	}
	result := item.GetAgentTypeFilter()
	if result != nil {
		t.Errorf("expected nil for empty filter, got %v", result)
	}

	// Also test with empty json.RawMessage
	item2 := McpMarketItem{
		AgentTypeFilter: json.RawMessage{},
	}
	result2 := item2.GetAgentTypeFilter()
	if result2 != nil {
		t.Errorf("expected nil for empty RawMessage, got %v", result2)
	}
}

func TestMcpMarketItem_GetAgentTypeFilter_Invalid(t *testing.T) {
	item := McpMarketItem{
		AgentTypeFilter: json.RawMessage(`{invalid json`),
	}
	result := item.GetAgentTypeFilter()
	if result != nil {
		t.Errorf("expected nil for invalid JSON, got %v", result)
	}
}

// ---------------------------------------------------------------------------
// 7. SkillMarketItem.GetAgentTypeFilter
// ---------------------------------------------------------------------------

func TestSkillMarketItem_GetAgentTypeFilter_Valid(t *testing.T) {
	item := SkillMarketItem{
		AgentTypeFilter: json.RawMessage(`["claude-code"]`),
	}
	result := item.GetAgentTypeFilter()
	if len(result) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result))
	}
	if result[0] != "claude-code" {
		t.Errorf("expected 'claude-code', got %q", result[0])
	}
}

func TestSkillMarketItem_GetAgentTypeFilter_Empty(t *testing.T) {
	item := SkillMarketItem{
		AgentTypeFilter: nil,
	}
	result := item.GetAgentTypeFilter()
	if result != nil {
		t.Errorf("expected nil for empty filter, got %v", result)
	}

	item2 := SkillMarketItem{
		AgentTypeFilter: json.RawMessage{},
	}
	result2 := item2.GetAgentTypeFilter()
	if result2 != nil {
		t.Errorf("expected nil for empty RawMessage, got %v", result2)
	}
}

func TestSkillMarketItem_GetAgentTypeFilter_Invalid(t *testing.T) {
	item := SkillMarketItem{
		AgentTypeFilter: json.RawMessage(`not-json`),
	}
	result := item.GetAgentTypeFilter()
	if result != nil {
		t.Errorf("expected nil for invalid JSON, got %v", result)
	}
}

// ---------------------------------------------------------------------------
// 8. SkillRegistry.GetCompatibleAgents
// ---------------------------------------------------------------------------

func TestSkillRegistry_GetCompatibleAgents_Valid(t *testing.T) {
	sr := SkillRegistry{
		CompatibleAgents: json.RawMessage(`["claude-code","aider","codex"]`),
	}
	result := sr.GetCompatibleAgents()
	if len(result) != 3 {
		t.Fatalf("expected 3 agents, got %d", len(result))
	}
	if result[0] != "claude-code" {
		t.Errorf("expected first agent 'claude-code', got %q", result[0])
	}
	if result[1] != "aider" {
		t.Errorf("expected second agent 'aider', got %q", result[1])
	}
	if result[2] != "codex" {
		t.Errorf("expected third agent 'codex', got %q", result[2])
	}
}

func TestSkillRegistry_GetCompatibleAgents_Empty(t *testing.T) {
	sr := SkillRegistry{
		CompatibleAgents: nil,
	}
	result := sr.GetCompatibleAgents()
	if result != nil {
		t.Errorf("expected nil for empty compatible_agents, got %v", result)
	}

	sr2 := SkillRegistry{
		CompatibleAgents: json.RawMessage{},
	}
	result2 := sr2.GetCompatibleAgents()
	if result2 != nil {
		t.Errorf("expected nil for empty RawMessage, got %v", result2)
	}
}

func TestSkillRegistry_GetCompatibleAgents_Invalid(t *testing.T) {
	sr := SkillRegistry{
		CompatibleAgents: json.RawMessage(`{broken-json`),
	}
	result := sr.GetCompatibleAgents()
	if result != nil {
		t.Errorf("expected nil for invalid JSON, got %v", result)
	}
}

// ---------------------------------------------------------------------------
// 9. SkillRegistry.HasAuth
// ---------------------------------------------------------------------------

func TestSkillRegistry_HasAuth_True(t *testing.T) {
	sr := SkillRegistry{
		AuthType: AuthTypeGitHubPAT,
	}
	if !sr.HasAuth() {
		t.Error("expected HasAuth() to return true for github_pat")
	}

	sr2 := SkillRegistry{
		AuthType: AuthTypeGitLabPAT,
	}
	if !sr2.HasAuth() {
		t.Error("expected HasAuth() to return true for gitlab_pat")
	}

	sr3 := SkillRegistry{
		AuthType: AuthTypeSSHKey,
	}
	if !sr3.HasAuth() {
		t.Error("expected HasAuth() to return true for ssh_key")
	}
}

func TestSkillRegistry_HasAuth_False_None(t *testing.T) {
	sr := SkillRegistry{
		AuthType: AuthTypeNone,
	}
	if sr.HasAuth() {
		t.Error("expected HasAuth() to return false for 'none'")
	}
}

func TestSkillRegistry_HasAuth_False_Empty(t *testing.T) {
	sr := SkillRegistry{
		AuthType: "",
	}
	if sr.HasAuth() {
		t.Error("expected HasAuth() to return false for empty string")
	}
}

// ---------------------------------------------------------------------------
// 10. SkillRegistry.HasAuthConfigured
// ---------------------------------------------------------------------------

func TestSkillRegistry_HasAuthConfigured_True(t *testing.T) {
	sr := SkillRegistry{
		AuthType:       AuthTypeGitHubPAT,
		AuthCredential: "encrypted-credential-value",
	}
	if !sr.HasAuthConfigured() {
		t.Error("expected HasAuthConfigured() to return true when auth_type is set and credential is non-empty")
	}
}

func TestSkillRegistry_HasAuthConfigured_False_NoAuth(t *testing.T) {
	sr := SkillRegistry{
		AuthType:       AuthTypeNone,
		AuthCredential: "some-value",
	}
	if sr.HasAuthConfigured() {
		t.Error("expected HasAuthConfigured() to return false when auth_type is 'none'")
	}
}

func TestSkillRegistry_HasAuthConfigured_False_NoCred(t *testing.T) {
	sr := SkillRegistry{
		AuthType:       AuthTypeGitHubPAT,
		AuthCredential: "",
	}
	if sr.HasAuthConfigured() {
		t.Error("expected HasAuthConfigured() to return false when credential is empty")
	}
}

// ---------------------------------------------------------------------------
// 5. Constants (continued)
// ---------------------------------------------------------------------------

func TestConstants(t *testing.T) {
	// Scope constants
	if ScopeOrg != "org" {
		t.Errorf("ScopeOrg = %q, want %q", ScopeOrg, "org")
	}
	if ScopeUser != "user" {
		t.Errorf("ScopeUser = %q, want %q", ScopeUser, "user")
	}

	// Install source constants
	if InstallSourceMarket != "market" {
		t.Errorf("InstallSourceMarket = %q, want %q", InstallSourceMarket, "market")
	}
	if InstallSourceGitHub != "github" {
		t.Errorf("InstallSourceGitHub = %q, want %q", InstallSourceGitHub, "github")
	}
	if InstallSourceUpload != "upload" {
		t.Errorf("InstallSourceUpload = %q, want %q", InstallSourceUpload, "upload")
	}

	// Transport type constants
	if TransportTypeStdio != "stdio" {
		t.Errorf("TransportTypeStdio = %q, want %q", TransportTypeStdio, "stdio")
	}
	if TransportTypeHTTP != "http" {
		t.Errorf("TransportTypeHTTP = %q, want %q", TransportTypeHTTP, "http")
	}
	if TransportTypeSSE != "sse" {
		t.Errorf("TransportTypeSSE = %q, want %q", TransportTypeSSE, "sse")
	}

	// Sync status constants
	if SyncStatusPending != "pending" {
		t.Errorf("SyncStatusPending = %q, want %q", SyncStatusPending, "pending")
	}
	if SyncStatusSyncing != "syncing" {
		t.Errorf("SyncStatusSyncing = %q, want %q", SyncStatusSyncing, "syncing")
	}
	if SyncStatusSuccess != "success" {
		t.Errorf("SyncStatusSuccess = %q, want %q", SyncStatusSuccess, "success")
	}
	if SyncStatusFailed != "failed" {
		t.Errorf("SyncStatusFailed = %q, want %q", SyncStatusFailed, "failed")
	}

	// Source type constants
	if SourceTypeAuto != "auto" {
		t.Errorf("SourceTypeAuto = %q, want %q", SourceTypeAuto, "auto")
	}
	if SourceTypeCollection != "collection" {
		t.Errorf("SourceTypeCollection = %q, want %q", SourceTypeCollection, "collection")
	}
	if SourceTypeSingle != "single" {
		t.Errorf("SourceTypeSingle = %q, want %q", SourceTypeSingle, "single")
	}
}
