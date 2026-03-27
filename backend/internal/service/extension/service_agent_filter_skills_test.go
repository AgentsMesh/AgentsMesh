package extension

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// ---------------------------------------------------------------------------
// Tests: GetEffectiveSkills (agent filter)
// ---------------------------------------------------------------------------

func TestGetEffectiveSkills_AgentFilter_MatchingAgent(t *testing.T) {
	// Skill with MarketItem filter ["claude-code"] should be included when agentSlug="claude-code"
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "filtered-skill",
					InstallSource: "market",
					ContentSha:    "abc123",
					StorageKey:    "skills/filtered-skill/v1.tar.gz",
					PackageSize:   1024,
					MarketItemID:  int64Ptr(100),
					MarketItem: &extension.SkillMarketItem{
						ID:              100,
						Slug:            "filtered-skill",
						AgentFilter: json.RawMessage(`["claude-code"]`),
						ContentSha:      "abc123",
						StorageKey:      "skills/filtered-skill/v1.tar.gz",
						PackageSize:     1024,
					},
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(resolved))
	}
	if resolved[0].Slug != "filtered-skill" {
		t.Errorf("expected slug 'filtered-skill', got %q", resolved[0].Slug)
	}
}

func TestGetEffectiveSkills_AgentFilter_NonMatchingAgent(t *testing.T) {
	// Skill with MarketItem filter ["claude-code"] should be excluded when agentSlug="aider"
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "claude-only-skill",
					InstallSource: "market",
					ContentSha:    "abc123",
					StorageKey:    "skills/claude-only-skill/v1.tar.gz",
					PackageSize:   1024,
					MarketItemID:  int64Ptr(100),
					MarketItem: &extension.SkillMarketItem{
						ID:              100,
						Slug:            "claude-only-skill",
						AgentFilter: json.RawMessage(`["claude-code"]`),
						ContentSha:      "abc123",
						StorageKey:      "skills/claude-only-skill/v1.tar.gz",
						PackageSize:     1024,
					},
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "aider")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 0 {
		t.Fatalf("expected 0 skills (filtered out), got %d", len(resolved))
	}
}

func TestGetEffectiveSkills_AgentFilter_GitHubInstallAlwaysIncluded(t *testing.T) {
	// Skill without MarketItem (github install) should always be included
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "github-skill",
					InstallSource: "github",
					ContentSha:    "def456",
					StorageKey:    "skills/github-skill/v1.tar.gz",
					PackageSize:   2048,
					MarketItem:    nil, // github install, no market item
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "aider")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 skill (github always included), got %d", len(resolved))
	}
}

func TestGetEffectiveSkills_AgentFilter_NullFilterAllowsAll(t *testing.T) {
	// Skill with MarketItem that has null agent_filter should be included for any agent
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "universal-skill",
					InstallSource: "market",
					ContentSha:    "ghi789",
					StorageKey:    "skills/universal-skill/v1.tar.gz",
					PackageSize:   512,
					MarketItemID:  int64Ptr(100),
					MarketItem: &extension.SkillMarketItem{
						ID:              100,
						Slug:            "universal-skill",
						AgentFilter: nil,
						ContentSha:      "ghi789",
						StorageKey:      "skills/universal-skill/v1.tar.gz",
						PackageSize:     512,
					},
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "any-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 skill (null filter = all agents), got %d", len(resolved))
	}
}

func TestGetEffectiveSkills_AgentFilter_EmptySlugDisablesFilter(t *testing.T) {
	// When agentSlug is empty, no filtering should happen
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "claude-only-skill",
					InstallSource: "market",
					ContentSha:    "abc123",
					StorageKey:    "skills/claude-only-skill/v1.tar.gz",
					PackageSize:   1024,
					MarketItemID:  int64Ptr(100),
					MarketItem: &extension.SkillMarketItem{
						ID:              100,
						Slug:            "claude-only-skill",
						AgentFilter: json.RawMessage(`["claude-code"]`),
						ContentSha:      "abc123",
						StorageKey:      "skills/claude-only-skill/v1.tar.gz",
						PackageSize:     1024,
					},
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 skill (empty agentSlug = no filtering), got %d", len(resolved))
	}
}

func TestGetEffectiveSkills_AgentFilter_MixedSkills(t *testing.T) {
	// Mix of filtered, unfiltered, and non-market skills
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "claude-only",
					InstallSource: "market",
					ContentSha:    "sha1",
					StorageKey:    "skills/claude-only/v1.tar.gz",
					PackageSize:   100,
					MarketItemID:  int64Ptr(100),
					MarketItem: &extension.SkillMarketItem{
						ID:              100,
						AgentFilter: json.RawMessage(`["claude-code"]`),
						ContentSha:      "sha1",
						StorageKey:      "skills/claude-only/v1.tar.gz",
						PackageSize:     100,
					},
				},
				{
					ID:            2,
					Slug:          "universal",
					InstallSource: "market",
					ContentSha:    "sha2",
					StorageKey:    "skills/universal/v1.tar.gz",
					PackageSize:   200,
					MarketItemID:  int64Ptr(101),
					MarketItem: &extension.SkillMarketItem{
						ID:              101,
						AgentFilter: nil,
						ContentSha:      "sha2",
						StorageKey:      "skills/universal/v1.tar.gz",
						PackageSize:     200,
					},
				},
				{
					ID:            3,
					Slug:          "github-skill",
					InstallSource: "github",
					ContentSha:    "sha3",
					StorageKey:    "skills/github-skill/v1.tar.gz",
					PackageSize:   300,
					MarketItem:    nil,
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	// For aider: should get universal + github-skill (not claude-only)
	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "aider")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 2 {
		t.Fatalf("expected 2 skills for aider, got %d", len(resolved))
	}
	slugs := make(map[string]bool)
	for _, r := range resolved {
		slugs[r.Slug] = true
	}
	if slugs["claude-only"] {
		t.Error("claude-only skill should have been filtered out for aider")
	}
	if !slugs["universal"] {
		t.Error("universal skill should be included")
	}
	if !slugs["github-skill"] {
		t.Error("github-skill should be included")
	}
}
