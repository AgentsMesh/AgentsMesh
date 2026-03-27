package extension

import (
	"context"
	"errors"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// ---------------------------------------------------------------------------
// Tests: ListRepoSkills (scope conversion)
// ---------------------------------------------------------------------------

func TestListRepoSkills_ScopeAllConvertsToEmpty(t *testing.T) {
	var capturedScope string
	repo := &svcMockRepo{
		listInstalledSkillsFn: func(_ context.Context, orgID, repoID int64, scope string) ([]*extension.InstalledSkill, error) {
			capturedScope = scope
			return nil, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.ListRepoSkills(context.Background(), 1, 2, 100, "all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedScope != "" {
		t.Errorf("expected empty scope, got %q", capturedScope)
	}
}

func TestListRepoSkills_ScopeOrgPassesThrough(t *testing.T) {
	var capturedScope string
	repo := &svcMockRepo{
		listInstalledSkillsFn: func(_ context.Context, orgID, repoID int64, scope string) ([]*extension.InstalledSkill, error) {
			capturedScope = scope
			return nil, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.ListRepoSkills(context.Background(), 1, 2, 100, "org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedScope != "org" {
		t.Errorf("expected scope 'org', got %q", capturedScope)
	}
}

// ---------------------------------------------------------------------------
// Tests: ListMarketSkills
// ---------------------------------------------------------------------------

func TestListMarketSkills_Success(t *testing.T) {
	called := false
	repo := &svcMockRepo{
		listSkillMarketItemsFn: func(_ context.Context, orgID *int64, query string, category string) ([]*extension.SkillMarketItem, error) {
			called = true
			if orgID == nil || *orgID != 10 {
				t.Errorf("expected orgID 10, got %v", orgID)
			}
			if query != "search" {
				t.Errorf("expected query 'search', got %q", query)
			}
			if category != "dev" {
				t.Errorf("expected category 'dev', got %q", category)
			}
			return []*extension.SkillMarketItem{
				{ID: 1, Slug: "skill-1"},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	result, err := svc.ListMarketSkills(context.Background(), 10, "search", "dev")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("repo.ListSkillMarketItems was not called")
	}
	if len(result) != 1 {
		t.Errorf("expected 1 item, got %d", len(result))
	}
}

func TestListMarketSkills_Error(t *testing.T) {
	repo := &svcMockRepo{
		listSkillMarketItemsFn: func(_ context.Context, orgID *int64, query string, category string) ([]*extension.SkillMarketItem, error) {
			return nil, errors.New("query failed")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.ListMarketSkills(context.Background(), 1, "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: ListMarketMcpServers
// ---------------------------------------------------------------------------

func TestListMarketMcpServers_Success(t *testing.T) {
	called := false
	repo := &svcMockRepo{
		listMcpMarketItemsFn: func(_ context.Context, query string, category string, limit, offset int) ([]*extension.McpMarketItem, int64, error) {
			called = true
			if query != "mcp" {
				t.Errorf("expected query 'mcp', got %q", query)
			}
			return []*extension.McpMarketItem{
				{ID: 1, Slug: "mcp-1"},
			}, 1, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	result, total, err := svc.ListMarketMcpServers(context.Background(), "mcp", "tools", 50, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("repo.ListMcpMarketItems was not called")
	}
	if len(result) != 1 {
		t.Errorf("expected 1 item, got %d", len(result))
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
}

func TestListMarketMcpServers_Error(t *testing.T) {
	repo := &svcMockRepo{
		listMcpMarketItemsFn: func(_ context.Context, query string, category string, limit, offset int) ([]*extension.McpMarketItem, int64, error) {
			return nil, 0, errors.New("market unavailable")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, _, err := svc.ListMarketMcpServers(context.Background(), "", "", 50, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: ListRepoMcpServers (scope conversion)
// ---------------------------------------------------------------------------

func TestListRepoMcpServers_ScopeAllConvertsToEmpty(t *testing.T) {
	var capturedScope string
	repo := &svcMockRepo{
		listInstalledMcpServersFn: func(_ context.Context, orgID, repoID int64, scope string) ([]*extension.InstalledMcpServer, error) {
			capturedScope = scope
			return nil, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.ListRepoMcpServers(context.Background(), 1, 2, 100, "all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedScope != "" {
		t.Errorf("expected empty scope, got %q", capturedScope)
	}
}

func TestListRepoMcpServers_ScopeOrgPassesThrough(t *testing.T) {
	var capturedScope string
	repo := &svcMockRepo{
		listInstalledMcpServersFn: func(_ context.Context, orgID, repoID int64, scope string) ([]*extension.InstalledMcpServer, error) {
			capturedScope = scope
			return nil, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.ListRepoMcpServers(context.Background(), 1, 2, 100, "org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedScope != "org" {
		t.Errorf("expected scope 'org', got %q", capturedScope)
	}
}
