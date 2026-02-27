package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	extensionservice "github.com/anthropics/agentsmesh/backend/internal/service/extension"
)

// ---------------------------------------------------------------------------
// Mock: ExtensionProvider
// ---------------------------------------------------------------------------

type mockExtensionProvider struct {
	mcpServers []*extension.InstalledMcpServer
	mcpErr     error
	skills     []*extensionservice.ResolvedSkill
	skillsErr  error
}

func (m *mockExtensionProvider) GetEffectiveMcpServers(ctx context.Context, orgID, userID, repoID int64, agentSlug string) ([]*extension.InstalledMcpServer, error) {
	return m.mcpServers, m.mcpErr
}

func (m *mockExtensionProvider) GetEffectiveSkills(ctx context.Context, orgID, userID, repoID int64, agentSlug string) ([]*extensionservice.ResolvedSkill, error) {
	return m.skills, m.skillsErr
}

// ---------------------------------------------------------------------------
// Mock: AgentConfigProvider (for loadExtensions integration tests)
// ---------------------------------------------------------------------------

type mockAgentConfigProviderForExt struct {
	agentType   *agent.AgentType
	credentials agent.EncryptedCredentials
	isRunner    bool
}

func (m *mockAgentConfigProviderForExt) GetAgentType(ctx context.Context, id int64) (*agent.AgentType, error) {
	if m.agentType == nil {
		return nil, fmt.Errorf("agent type not found")
	}
	return m.agentType, nil
}

func (m *mockAgentConfigProviderForExt) GetUserEffectiveConfig(ctx context.Context, userID, agentTypeID int64, overrides agent.ConfigValues) agent.ConfigValues {
	result := make(agent.ConfigValues)
	for k, v := range overrides {
		result[k] = v
	}
	return result
}

func (m *mockAgentConfigProviderForExt) GetEffectiveCredentialsForPod(ctx context.Context, userID, agentTypeID int64, profileID *int64) (agent.EncryptedCredentials, bool, error) {
	return m.credentials, m.isRunner, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func int64Ptr(v int64) *int64 { return &v }

func newMinimalBuildContext() *BuildContext {
	return &BuildContext{
		Request: &ConfigBuildRequest{
			MCPPort: 0,
			PodKey:  "pod-1",
		},
		AgentType: &agent.AgentType{
			ID:            1,
			Slug:          ClaudeCodeSlug,
			LaunchCommand: "claude",
		},
		Config:      agent.ConfigValues{},
		Credentials: agent.EncryptedCredentials{},
		TemplateCtx: map[string]interface{}{
			"config": agent.ConfigValues{},
			"sandbox": map[string]interface{}{
				"root_path": "{{.sandbox.root_path}}",
				"work_dir":  "{{.sandbox.work_dir}}",
			},
			"mcp_port": 0,
			"pod_key":  "pod-1",
		},
	}
}

// ---------------------------------------------------------------------------
// 1. ClaudeCodeBuilder.BuildResourcesToDownload
// ---------------------------------------------------------------------------

func TestClaudeCodeBuilder_BuildResourcesToDownload(t *testing.T) {
	builder := NewClaudeCodeBuilder()

	t.Run("empty_skills", func(t *testing.T) {
		ctx := newMinimalBuildContext()
		ctx.ResolvedSkills = nil

		resources, err := builder.BuildResourcesToDownload(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resources != nil {
			t.Errorf("expected nil, got %v", resources)
		}
	})

	t.Run("with_skills", func(t *testing.T) {
		ctx := newMinimalBuildContext()
		ctx.ResolvedSkills = []*extensionservice.ResolvedSkill{
			{
				Slug:        "skill-alpha",
				ContentSha:  "sha256-aaa",
				DownloadURL: "https://cdn.example.com/skills/alpha.tar.gz",
				PackageSize: 1024,
				TargetDir:   "skills/skill-alpha",
			},
			{
				Slug:        "skill-beta",
				ContentSha:  "sha256-bbb",
				DownloadURL: "https://cdn.example.com/skills/beta.tar.gz",
				PackageSize: 2048,
				TargetDir:   "skills/skill-beta",
			},
		}

		resources, err := builder.BuildResourcesToDownload(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resources) != 2 {
			t.Fatalf("expected 2 resources, got %d", len(resources))
		}

		// Verify first resource
		r0 := resources[0]
		if r0.Sha != "sha256-aaa" {
			t.Errorf("resources[0].Sha = %q, want %q", r0.Sha, "sha256-aaa")
		}
		if r0.DownloadUrl != "https://cdn.example.com/skills/alpha.tar.gz" {
			t.Errorf("resources[0].DownloadUrl = %q, want %q", r0.DownloadUrl, "https://cdn.example.com/skills/alpha.tar.gz")
		}
		if r0.ResourceType != "skill_package" {
			t.Errorf("resources[0].ResourceType = %q, want %q", r0.ResourceType, "skill_package")
		}
		if r0.SizeBytes != 1024 {
			t.Errorf("resources[0].SizeBytes = %d, want %d", r0.SizeBytes, 1024)
		}

		// Verify second resource
		r1 := resources[1]
		if r1.Sha != "sha256-bbb" {
			t.Errorf("resources[1].Sha = %q, want %q", r1.Sha, "sha256-bbb")
		}
		if r1.SizeBytes != 2048 {
			t.Errorf("resources[1].SizeBytes = %d, want %d", r1.SizeBytes, 2048)
		}
	})

	t.Run("target_path_format", func(t *testing.T) {
		ctx := newMinimalBuildContext()
		ctx.ResolvedSkills = []*extensionservice.ResolvedSkill{
			{
				Slug:        "my-skill",
				ContentSha:  "sha-xyz",
				DownloadURL: "https://cdn.example.com/skills/my-skill.tar.gz",
				PackageSize: 512,
				TargetDir:   "skills/my-skill",
			},
		}

		resources, err := builder.BuildResourcesToDownload(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resources) != 1 {
			t.Fatalf("expected 1 resource, got %d", len(resources))
		}

		expected := "{{.sandbox.root_path}}/agentsmesh-plugin/skills/my-skill"
		if resources[0].TargetPath != expected {
			t.Errorf("TargetPath = %q, want %q", resources[0].TargetPath, expected)
		}
	})
}

// ---------------------------------------------------------------------------
// 2. ClaudeCodeBuilder.buildMcpConfig
// ---------------------------------------------------------------------------

func TestClaudeCodeBuilder_BuildMcpConfig(t *testing.T) {
	builder := NewClaudeCodeBuilder()

	t.Run("no_mcp_port", func(t *testing.T) {
		ctx := newMinimalBuildContext()
		ctx.TemplateCtx["mcp_port"] = nil

		config := builder.buildMcpConfig(ctx)
		servers, ok := config["mcpServers"].(map[string]interface{})
		if !ok {
			t.Fatalf("mcpServers is not map[string]interface{}, got %T", config["mcpServers"])
		}
		if len(servers) != 0 {
			t.Errorf("expected 0 servers when mcp_port=nil, got %d", len(servers))
		}
	})

	t.Run("mcp_port_zero", func(t *testing.T) {
		ctx := newMinimalBuildContext()
		ctx.TemplateCtx["mcp_port"] = 0

		config := builder.buildMcpConfig(ctx)
		servers := config["mcpServers"].(map[string]interface{})
		if _, exists := servers["agentsmesh"]; exists {
			t.Error("agentsmesh server should NOT be present when mcp_port=0")
		}
	})

	t.Run("with_mcp_port", func(t *testing.T) {
		ctx := newMinimalBuildContext()
		ctx.TemplateCtx["mcp_port"] = 19000

		config := builder.buildMcpConfig(ctx)
		servers := config["mcpServers"].(map[string]interface{})
		amServer, exists := servers["agentsmesh"]
		if !exists {
			t.Fatal("agentsmesh server should be present when mcp_port=19000")
		}

		serverMap := amServer.(map[string]interface{})
		if serverMap["type"] != "http" {
			t.Errorf("agentsmesh type = %q, want %q", serverMap["type"], "http")
		}
		expectedURL := "http://127.0.0.1:19000/mcp"
		if serverMap["url"] != expectedURL {
			t.Errorf("agentsmesh url = %q, want %q", serverMap["url"], expectedURL)
		}
		// Verify X-Pod-Key header is included
		headers, hasHeaders := serverMap["headers"]
		if !hasHeaders {
			t.Fatal("agentsmesh should have headers with X-Pod-Key")
		}
		headerMap := headers.(map[string]string)
		if headerMap["X-Pod-Key"] != "pod-1" {
			t.Errorf("X-Pod-Key = %q, want %q", headerMap["X-Pod-Key"], "pod-1")
		}
	})

	t.Run("with_installed_servers", func(t *testing.T) {
		ctx := newMinimalBuildContext()
		ctx.TemplateCtx["mcp_port"] = nil // no built-in server
		ctx.McpServers = []*extension.InstalledMcpServer{
			{
				Slug:          "server-a",
				TransportType: "http",
				HttpURL:       "https://a.example.com/mcp",
				IsEnabled:     true,
			},
			{
				Slug:          "server-b",
				TransportType: "stdio",
				Command:       "npx",
				Args:          json.RawMessage(`["@server-b/mcp"]`),
				IsEnabled:     true,
			},
		}

		config := builder.buildMcpConfig(ctx)
		servers := config["mcpServers"].(map[string]interface{})
		if len(servers) != 2 {
			t.Fatalf("expected 2 servers, got %d", len(servers))
		}
		if _, exists := servers["server-a"]; !exists {
			t.Error("server-a should be present")
		}
		if _, exists := servers["server-b"]; !exists {
			t.Error("server-b should be present")
		}
	})

	t.Run("disabled_server_excluded", func(t *testing.T) {
		ctx := newMinimalBuildContext()
		ctx.TemplateCtx["mcp_port"] = nil
		ctx.McpServers = []*extension.InstalledMcpServer{
			{
				Slug:          "enabled-one",
				TransportType: "http",
				HttpURL:       "https://enabled.example.com",
				IsEnabled:     true,
			},
			{
				Slug:          "disabled-one",
				TransportType: "http",
				HttpURL:       "https://disabled.example.com",
				IsEnabled:     false,
			},
		}

		config := builder.buildMcpConfig(ctx)
		servers := config["mcpServers"].(map[string]interface{})
		if _, exists := servers["enabled-one"]; !exists {
			t.Error("enabled-one should be present")
		}
		if _, exists := servers["disabled-one"]; exists {
			t.Error("disabled-one should NOT be present")
		}
	})

	t.Run("combined", func(t *testing.T) {
		ctx := newMinimalBuildContext()
		ctx.TemplateCtx["mcp_port"] = 18000
		ctx.McpServers = []*extension.InstalledMcpServer{
			{
				Slug:          "external-mcp",
				TransportType: "http",
				HttpURL:       "https://ext.example.com/mcp",
				IsEnabled:     true,
			},
		}

		config := builder.buildMcpConfig(ctx)
		servers := config["mcpServers"].(map[string]interface{})

		// Both agentsmesh built-in + external-mcp
		if len(servers) != 2 {
			t.Fatalf("expected 2 servers, got %d: %v", len(servers), servers)
		}
		if _, exists := servers["agentsmesh"]; !exists {
			t.Error("agentsmesh should be present")
		}
		if _, exists := servers["external-mcp"]; !exists {
			t.Error("external-mcp should be present")
		}
	})
}

// ---------------------------------------------------------------------------
// 3. ClaudeCodeBuilder.BuildFilesToCreate
// ---------------------------------------------------------------------------

func TestClaudeCodeBuilder_BuildFilesToCreate(t *testing.T) {
	builder := NewClaudeCodeBuilder()

	t.Run("generates_plugin_structure", func(t *testing.T) {
		ctx := newMinimalBuildContext()

		files, err := builder.BuildFilesToCreate(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Expect at least: agentsmesh-plugin dir, plugin.json, .mcp.json, skills dir = 4 entries
		if len(files) < 4 {
			t.Fatalf("expected at least 4 files, got %d", len(files))
		}

		// Check for the directory entry
		foundPluginDir := false
		foundPluginJSON := false
		foundMcpJSON := false
		foundSkillsDir := false

		for _, f := range files {
			switch {
			case f.IsDirectory && strings.HasSuffix(f.Path, "agentsmesh-plugin"):
				foundPluginDir = true
			case strings.HasSuffix(f.Path, ".claude-plugin/plugin.json"):
				foundPluginJSON = true
			case strings.HasSuffix(f.Path, ".mcp.json"):
				foundMcpJSON = true
			case f.IsDirectory && strings.HasSuffix(f.Path, "/skills"):
				foundSkillsDir = true
			}
		}

		if !foundPluginDir {
			t.Error("missing agentsmesh-plugin directory entry")
		}
		if !foundPluginJSON {
			t.Error("missing .claude-plugin/plugin.json file entry")
		}
		if !foundMcpJSON {
			t.Error("missing .mcp.json file entry")
		}
		if !foundSkillsDir {
			t.Error("missing skills/ directory entry")
		}
	})

	t.Run("plugin_json_content", func(t *testing.T) {
		ctx := newMinimalBuildContext()

		files, err := builder.BuildFilesToCreate(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var pluginJSONContent string
		for _, f := range files {
			if strings.HasSuffix(f.Path, ".claude-plugin/plugin.json") {
				pluginJSONContent = f.Content
				break
			}
		}

		if pluginJSONContent == "" {
			t.Fatal("plugin.json content is empty or not found")
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(pluginJSONContent), &parsed); err != nil {
			t.Fatalf("plugin.json is not valid JSON: %v", err)
		}

		if name, ok := parsed["name"].(string); !ok || name == "" {
			t.Errorf("plugin.json missing or empty 'name', got %v", parsed["name"])
		}
		if desc, ok := parsed["description"].(string); !ok || desc == "" {
			t.Errorf("plugin.json missing or empty 'description', got %v", parsed["description"])
		}
		if ver, ok := parsed["version"].(string); !ok || ver == "" {
			t.Errorf("plugin.json missing or empty 'version', got %v", parsed["version"])
		}
	})
}

// ---------------------------------------------------------------------------
// 4. Builder interface compliance
// ---------------------------------------------------------------------------

func TestBuilderInterface_Compliance(t *testing.T) {
	claude := NewClaudeCodeBuilder()
	aider := NewAiderBuilder()
	base := NewBaseAgentBuilder("generic")

	t.Run("claude_code_supports_plugin", func(t *testing.T) {
		if !claude.SupportsPlugin() {
			t.Error("ClaudeCodeBuilder.SupportsPlugin() should return true")
		}
	})

	t.Run("claude_code_supports_mcp", func(t *testing.T) {
		if !claude.SupportsMcp() {
			t.Error("ClaudeCodeBuilder.SupportsMcp() should return true")
		}
	})

	t.Run("claude_code_supports_skills", func(t *testing.T) {
		if !claude.SupportsSkills() {
			t.Error("ClaudeCodeBuilder.SupportsSkills() should return true")
		}
	})

	t.Run("aider_no_mcp", func(t *testing.T) {
		if aider.SupportsMcp() {
			t.Error("AiderBuilder.SupportsMcp() should return false")
		}
	})

	t.Run("aider_no_skills", func(t *testing.T) {
		if aider.SupportsSkills() {
			t.Error("AiderBuilder.SupportsSkills() should return false")
		}
	})

	t.Run("aider_no_plugin", func(t *testing.T) {
		if aider.SupportsPlugin() {
			t.Error("AiderBuilder.SupportsPlugin() should return false")
		}
	})

	t.Run("base_no_plugin", func(t *testing.T) {
		if base.SupportsPlugin() {
			t.Error("BaseAgentBuilder.SupportsPlugin() should return false")
		}
	})
}

// ---------------------------------------------------------------------------
// 5. ConfigBuilder.loadExtensions
// ---------------------------------------------------------------------------

func TestConfigBuilder_LoadExtensions(t *testing.T) {
	bgCtx := context.Background()

	// Minimal agent type for Claude Code (supports MCP + Skills)
	claudeAgentType := &agent.AgentType{
		ID:            1,
		Slug:          ClaudeCodeSlug,
		LaunchCommand: "claude",
	}

	// Minimal agent type for Aider (no MCP, no Skills)
	aiderAgentType := &agent.AgentType{
		ID:            2,
		Slug:          AiderSlug,
		LaunchCommand: "aider",
	}

	t.Run("no_extension_provider", func(t *testing.T) {
		provider := &mockAgentConfigProviderForExt{
			agentType:   claudeAgentType,
			credentials: agent.EncryptedCredentials{},
		}
		cb := NewConfigBuilder(provider)
		// extensionProvider is nil by default

		repoID := int64(10)
		req := &ConfigBuildRequest{
			AgentTypeID:    1,
			UserID:         1,
			OrganizationID: 1,
			RepositoryID:   &repoID,
			PodKey:         "pod-no-ext",
		}

		cmd, err := cb.BuildPodCommand(bgCtx, req)
		if err != nil {
			t.Fatalf("BuildPodCommand failed: %v", err)
		}
		// Should still succeed; no extensions loaded
		if cmd == nil {
			t.Fatal("cmd should not be nil")
		}
	})

	t.Run("no_repository_id", func(t *testing.T) {
		provider := &mockAgentConfigProviderForExt{
			agentType:   claudeAgentType,
			credentials: agent.EncryptedCredentials{},
		}
		cb := NewConfigBuilder(provider)
		cb.SetExtensionProvider(&mockExtensionProvider{
			mcpServers: []*extension.InstalledMcpServer{
				{Slug: "should-not-appear", IsEnabled: true, TransportType: "http", HttpURL: "https://x.com"},
			},
			skills: []*extensionservice.ResolvedSkill{
				{Slug: "also-should-not-appear"},
			},
		})

		req := &ConfigBuildRequest{
			AgentTypeID:    1,
			UserID:         1,
			OrganizationID: 1,
			RepositoryID:   nil, // <-- nil
			PodKey:         "pod-no-repo",
		}

		cmd, err := cb.BuildPodCommand(bgCtx, req)
		if err != nil {
			t.Fatalf("BuildPodCommand failed: %v", err)
		}

		// MCP config should NOT contain installed servers (only built-in if mcp_port set)
		// Check that resources to download is empty (no skills loaded)
		if len(cmd.ResourcesToDownload) != 0 {
			t.Errorf("expected 0 ResourcesToDownload when RepositoryID is nil, got %d", len(cmd.ResourcesToDownload))
		}
	})

	t.Run("builder_no_mcp_support", func(t *testing.T) {
		provider := &mockAgentConfigProviderForExt{
			agentType:   aiderAgentType,
			credentials: agent.EncryptedCredentials{},
		}

		extProvider := &mockExtensionProvider{
			mcpServers: []*extension.InstalledMcpServer{
				{Slug: "mcp-srv", IsEnabled: true, TransportType: "http", HttpURL: "https://x.com"},
			},
			skills: []*extensionservice.ResolvedSkill{
				{Slug: "some-skill"},
			},
		}

		cb := NewConfigBuilder(provider)
		cb.SetExtensionProvider(extProvider)

		repoID := int64(10)
		req := &ConfigBuildRequest{
			AgentTypeID:    2,
			UserID:         1,
			OrganizationID: 1,
			RepositoryID:   &repoID,
			PodKey:         "pod-aider-nomcp",
		}

		cmd, err := cb.BuildPodCommand(bgCtx, req)
		if err != nil {
			t.Fatalf("BuildPodCommand failed: %v", err)
		}

		// Aider doesn't support MCP/Skills, so no resources to download
		if len(cmd.ResourcesToDownload) != 0 {
			t.Errorf("expected 0 ResourcesToDownload for Aider, got %d", len(cmd.ResourcesToDownload))
		}
	})

	t.Run("builder_no_skills_support", func(t *testing.T) {
		provider := &mockAgentConfigProviderForExt{
			agentType:   aiderAgentType,
			credentials: agent.EncryptedCredentials{},
		}

		extProvider := &mockExtensionProvider{
			skills: []*extensionservice.ResolvedSkill{
				{Slug: "some-skill", ContentSha: "sha1", DownloadURL: "https://x.com/skill.tar.gz", PackageSize: 100, TargetDir: "skills/some-skill"},
			},
		}

		cb := NewConfigBuilder(provider)
		cb.SetExtensionProvider(extProvider)

		repoID := int64(10)
		req := &ConfigBuildRequest{
			AgentTypeID:    2,
			UserID:         1,
			OrganizationID: 1,
			RepositoryID:   &repoID,
			PodKey:         "pod-aider-noskills",
		}

		cmd, err := cb.BuildPodCommand(bgCtx, req)
		if err != nil {
			t.Fatalf("BuildPodCommand failed: %v", err)
		}

		// Aider doesn't support skills
		if len(cmd.ResourcesToDownload) != 0 {
			t.Errorf("expected 0 ResourcesToDownload for Aider, got %d", len(cmd.ResourcesToDownload))
		}
	})

	t.Run("extension_error_logged_not_fatal", func(t *testing.T) {
		provider := &mockAgentConfigProviderForExt{
			agentType:   claudeAgentType,
			credentials: agent.EncryptedCredentials{},
		}

		extProvider := &mockExtensionProvider{
			mcpErr:    fmt.Errorf("MCP fetch error"),
			skillsErr: fmt.Errorf("Skills fetch error"),
		}

		cb := NewConfigBuilder(provider)
		cb.SetExtensionProvider(extProvider)

		repoID := int64(10)
		req := &ConfigBuildRequest{
			AgentTypeID:    1,
			UserID:         1,
			OrganizationID: 1,
			RepositoryID:   &repoID,
			PodKey:         "pod-ext-error",
		}

		// Should NOT return error — errors are logged but not fatal
		cmd, err := cb.BuildPodCommand(bgCtx, req)
		if err != nil {
			t.Fatalf("BuildPodCommand should succeed even when extension provider fails, got: %v", err)
		}
		if cmd == nil {
			t.Fatal("cmd should not be nil")
		}

		// No resources since skills fetch failed
		if len(cmd.ResourcesToDownload) != 0 {
			t.Errorf("expected 0 ResourcesToDownload when skills fetch fails, got %d", len(cmd.ResourcesToDownload))
		}
	})

	t.Run("full_extension_flow_with_claude", func(t *testing.T) {
		provider := &mockAgentConfigProviderForExt{
			agentType:   claudeAgentType,
			credentials: agent.EncryptedCredentials{},
		}

		extProvider := &mockExtensionProvider{
			mcpServers: []*extension.InstalledMcpServer{
				{
					Slug:          "ext-mcp-1",
					TransportType: "http",
					HttpURL:       "https://mcp1.example.com",
					IsEnabled:     true,
				},
			},
			skills: []*extensionservice.ResolvedSkill{
				{
					Slug:        "my-skill",
					ContentSha:  "sha256-abc",
					DownloadURL: "https://cdn.example.com/my-skill.tar.gz",
					PackageSize: 4096,
					TargetDir:   "skills/my-skill",
				},
			},
		}

		cb := NewConfigBuilder(provider)
		cb.SetExtensionProvider(extProvider)

		repoID := int64(42)
		req := &ConfigBuildRequest{
			AgentTypeID:    1,
			UserID:         1,
			OrganizationID: 1,
			RepositoryID:   &repoID,
			MCPPort:        19000,
			PodKey:         "pod-full-ext",
		}

		cmd, err := cb.BuildPodCommand(bgCtx, req)
		if err != nil {
			t.Fatalf("BuildPodCommand failed: %v", err)
		}

		// Verify resources contain the skill
		if len(cmd.ResourcesToDownload) != 1 {
			t.Fatalf("expected 1 ResourcesToDownload, got %d", len(cmd.ResourcesToDownload))
		}
		r := cmd.ResourcesToDownload[0]
		if r.Sha != "sha256-abc" {
			t.Errorf("resource SHA = %q, want %q", r.Sha, "sha256-abc")
		}
		if r.DownloadUrl != "https://cdn.example.com/my-skill.tar.gz" {
			t.Errorf("resource DownloadUrl = %q, want correct URL", r.DownloadUrl)
		}

		// Verify MCP config in files includes the installed server
		foundMcpJSON := false
		for _, f := range cmd.FilesToCreate {
			if strings.HasSuffix(f.Path, ".mcp.json") {
				foundMcpJSON = true
				var mcpConfig map[string]interface{}
				if err := json.Unmarshal([]byte(f.Content), &mcpConfig); err != nil {
					t.Fatalf("failed to parse .mcp.json: %v", err)
				}
				servers, ok := mcpConfig["mcpServers"].(map[string]interface{})
				if !ok {
					t.Fatalf("mcpServers is not a map")
				}
				// Should contain agentsmesh + ext-mcp-1
				if _, exists := servers["agentsmesh"]; !exists {
					t.Error("agentsmesh server should be present in .mcp.json")
				}
				if _, exists := servers["ext-mcp-1"]; !exists {
					t.Error("ext-mcp-1 server should be present in .mcp.json")
				}
				break
			}
		}
		if !foundMcpJSON {
			t.Error("expected .mcp.json in FilesToCreate")
		}
	})
}
