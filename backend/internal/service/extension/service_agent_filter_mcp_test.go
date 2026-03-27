package extension

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// ---------------------------------------------------------------------------
// Tests: GetEffectiveMcpServers (agent filter)
// ---------------------------------------------------------------------------

func TestGetEffectiveMcpServers_AgentFilter_MatchingAgent(t *testing.T) {
	// MCP server with MarketItem filter ["claude-code"] should be included when agentSlug="claude-code"
	repo := &svcMockRepo{
		getEffectiveMcpServersFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledMcpServer, error) {
			return []*extension.InstalledMcpServer{
				{
					ID:           1,
					Slug:         "filtered-server",
					MarketItemID: int64Ptr(100),
					MarketItem: &extension.McpMarketItem{
						ID:              100,
						Slug:            "filtered-server",
						AgentFilter: json.RawMessage(`["claude-code"]`),
					},
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	servers, err := svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	if servers[0].Slug != "filtered-server" {
		t.Errorf("expected slug 'filtered-server', got %q", servers[0].Slug)
	}
}

func TestGetEffectiveMcpServers_AgentFilter_NonMatchingAgent(t *testing.T) {
	// MCP server with MarketItem filter ["claude-code"] should be excluded when agentSlug="aider"
	repo := &svcMockRepo{
		getEffectiveMcpServersFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledMcpServer, error) {
			return []*extension.InstalledMcpServer{
				{
					ID:           1,
					Slug:         "claude-only-server",
					MarketItemID: int64Ptr(100),
					MarketItem: &extension.McpMarketItem{
						ID:              100,
						Slug:            "claude-only-server",
						AgentFilter: json.RawMessage(`["claude-code"]`),
					},
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	servers, err := svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "aider")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 0 {
		t.Fatalf("expected 0 servers (filtered out), got %d", len(servers))
	}
}

func TestGetEffectiveMcpServers_AgentFilter_CustomServerAlwaysIncluded(t *testing.T) {
	// MCP server without MarketItem (custom install) should always be included
	repo := &svcMockRepo{
		getEffectiveMcpServersFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledMcpServer, error) {
			return []*extension.InstalledMcpServer{
				{
					ID:         1,
					Slug:       "custom-server",
					MarketItem: nil, // custom install, no market item
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	servers, err := svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "aider")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server (custom always included), got %d", len(servers))
	}
}

func TestGetEffectiveMcpServers_AgentFilter_NullFilterAllowsAll(t *testing.T) {
	// MCP server with MarketItem that has null/empty agent_filter should be included for any agent
	repo := &svcMockRepo{
		getEffectiveMcpServersFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledMcpServer, error) {
			return []*extension.InstalledMcpServer{
				{
					ID:           1,
					Slug:         "universal-server",
					MarketItemID: int64Ptr(100),
					MarketItem: &extension.McpMarketItem{
						ID:              100,
						Slug:            "universal-server",
						AgentFilter: nil, // null = all agents
					},
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	servers, err := svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "any-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server (null filter = all agents), got %d", len(servers))
	}
}

func TestGetEffectiveMcpServers_AgentFilter_EmptySlugDisablesFilter(t *testing.T) {
	// When agentSlug is empty, no filtering should happen
	repo := &svcMockRepo{
		getEffectiveMcpServersFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledMcpServer, error) {
			return []*extension.InstalledMcpServer{
				{
					ID:           1,
					Slug:         "claude-only",
					MarketItemID: int64Ptr(100),
					MarketItem: &extension.McpMarketItem{
						ID:              100,
						Slug:            "claude-only",
						AgentFilter: json.RawMessage(`["claude-code"]`),
					},
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	servers, err := svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server (empty agentSlug = no filtering), got %d", len(servers))
	}
}

func TestGetEffectiveMcpServers_AgentFilter_MultipleAgents(t *testing.T) {
	// MCP server with filter ["claude-code", "aider"] should be included for both
	repo := &svcMockRepo{
		getEffectiveMcpServersFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledMcpServer, error) {
			return []*extension.InstalledMcpServer{
				{
					ID:           1,
					Slug:         "multi-agent-server",
					MarketItemID: int64Ptr(100),
					MarketItem: &extension.McpMarketItem{
						ID:              100,
						Slug:            "multi-agent-server",
						AgentFilter: json.RawMessage(`["claude-code", "aider"]`),
					},
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	// Should be included for claude-code
	servers, err := svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server for claude-code, got %d", len(servers))
	}

	// Should be included for aider
	servers, err = svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "aider")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server for aider, got %d", len(servers))
	}

	// Should NOT be included for codex
	servers, err = svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "codex")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 0 {
		t.Fatalf("expected 0 servers for codex, got %d", len(servers))
	}
}

func TestGetEffectiveMcpServers_AgentFilter_MixedServers(t *testing.T) {
	// Mix of filtered, unfiltered, and custom servers
	repo := &svcMockRepo{
		getEffectiveMcpServersFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledMcpServer, error) {
			return []*extension.InstalledMcpServer{
				{
					ID:           1,
					Slug:         "claude-only",
					MarketItemID: int64Ptr(100),
					MarketItem: &extension.McpMarketItem{
						ID:              100,
						AgentFilter: json.RawMessage(`["claude-code"]`),
					},
				},
				{
					ID:           2,
					Slug:         "universal",
					MarketItemID: int64Ptr(101),
					MarketItem: &extension.McpMarketItem{
						ID:              101,
						AgentFilter: nil,
					},
				},
				{
					ID:         3,
					Slug:       "custom",
					MarketItem: nil,
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	// For aider: should get universal + custom (not claude-only)
	servers, err := svc.GetEffectiveMcpServers(context.Background(), 1, 2, 3, "aider")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers for aider, got %d", len(servers))
	}
	slugs := make(map[string]bool)
	for _, s := range servers {
		slugs[s.Slug] = true
	}
	if slugs["claude-only"] {
		t.Error("claude-only server should have been filtered out for aider")
	}
	if !slugs["universal"] {
		t.Error("universal server should be included")
	}
	if !slugs["custom"] {
		t.Error("custom server should be included")
	}
}

// ---------------------------------------------------------------------------
// Tests: Agent filtering for GetEffectiveSkills
// ---------------------------------------------------------------------------
