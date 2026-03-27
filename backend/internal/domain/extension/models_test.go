package extension

import (
	"encoding/json"
	"testing"
)

// ---------------------------------------------------------------------------
// SkillRegistry.IsPlatformLevel()
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
// TableName() methods
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

func TestSkillRegistryOverride_TableName(t *testing.T) {
	override := SkillRegistryOverride{}
	if override.TableName() != "skill_registry_overrides" {
		t.Errorf("TableName() = %q, want %q", override.TableName(), "skill_registry_overrides")
	}
}

// ---------------------------------------------------------------------------
// McpMarketItem.GetAgentFilter
// ---------------------------------------------------------------------------

func TestMcpMarketItem_GetAgentFilter_Valid(t *testing.T) {
	item := McpMarketItem{
		AgentFilter: json.RawMessage(`["claude-code","aider"]`),
	}
	result := item.GetAgentFilter()
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

func TestMcpMarketItem_GetAgentFilter_Empty(t *testing.T) {
	item := McpMarketItem{
		AgentFilter: nil,
	}
	result := item.GetAgentFilter()
	if result != nil {
		t.Errorf("expected nil for empty filter, got %v", result)
	}

	// Also test with empty json.RawMessage
	item2 := McpMarketItem{
		AgentFilter: json.RawMessage{},
	}
	result2 := item2.GetAgentFilter()
	if result2 != nil {
		t.Errorf("expected nil for empty RawMessage, got %v", result2)
	}
}

func TestMcpMarketItem_GetAgentFilter_Invalid(t *testing.T) {
	item := McpMarketItem{
		AgentFilter: json.RawMessage(`{invalid json`),
	}
	result := item.GetAgentFilter()
	if result != nil {
		t.Errorf("expected nil for invalid JSON, got %v", result)
	}
}

// ---------------------------------------------------------------------------
// SkillMarketItem.GetAgentFilter
// ---------------------------------------------------------------------------

func TestSkillMarketItem_GetAgentFilter_Valid(t *testing.T) {
	item := SkillMarketItem{
		AgentFilter: json.RawMessage(`["claude-code"]`),
	}
	result := item.GetAgentFilter()
	if len(result) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result))
	}
	if result[0] != "claude-code" {
		t.Errorf("expected 'claude-code', got %q", result[0])
	}
}

func TestSkillMarketItem_GetAgentFilter_Empty(t *testing.T) {
	item := SkillMarketItem{
		AgentFilter: nil,
	}
	result := item.GetAgentFilter()
	if result != nil {
		t.Errorf("expected nil for empty filter, got %v", result)
	}

	item2 := SkillMarketItem{
		AgentFilter: json.RawMessage{},
	}
	result2 := item2.GetAgentFilter()
	if result2 != nil {
		t.Errorf("expected nil for empty RawMessage, got %v", result2)
	}
}

func TestSkillMarketItem_GetAgentFilter_Invalid(t *testing.T) {
	item := SkillMarketItem{
		AgentFilter: json.RawMessage(`not-json`),
	}
	result := item.GetAgentFilter()
	if result != nil {
		t.Errorf("expected nil for invalid JSON, got %v", result)
	}
}

// ---------------------------------------------------------------------------
// SkillRegistry.GetCompatibleAgents
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
// SkillRegistry.HasAuth
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
// SkillRegistry.HasAuthConfigured
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
// Constants
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
