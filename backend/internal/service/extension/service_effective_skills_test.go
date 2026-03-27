package extension

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// ---------------------------------------------------------------------------
// Tests: GetEffectiveSkills
// ---------------------------------------------------------------------------

func TestGetEffectiveSkills_Success(t *testing.T) {
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "skill-a",
					InstallSource: "github",
					ContentSha:    "sha-abc",
					StorageKey:    "skills/skill-a/v1.tar.gz",
					PackageSize:   2048,
				},
			}, nil
		},
	}
	stor := &svcMockStorage{
		getURLFn: func(_ context.Context, key string, expiry time.Duration) (string, error) {
			if expiry != presignedURLExpiry {
				t.Errorf("expected expiry %v, got %v", presignedURLExpiry, expiry)
			}
			return "https://cdn.example.com/" + key + "?token=abc", nil
		},
	}
	svc := newTestService(repo, stor, nil)

	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved skill, got %d", len(resolved))
	}
	r := resolved[0]
	if r.Slug != "skill-a" {
		t.Errorf("expected slug 'skill-a', got %q", r.Slug)
	}
	if r.ContentSha != "sha-abc" {
		t.Errorf("expected sha 'sha-abc', got %q", r.ContentSha)
	}
	if r.DownloadURL != "https://cdn.example.com/skills/skill-a/v1.tar.gz?token=abc" {
		t.Errorf("unexpected download URL: %q", r.DownloadURL)
	}
	if r.PackageSize != 2048 {
		t.Errorf("expected package size 2048, got %d", r.PackageSize)
	}
	if r.TargetDir != "skills/skill-a" {
		t.Errorf("expected target dir 'skills/skill-a', got %q", r.TargetDir)
	}
}

func TestGetEffectiveSkills_SkipsEmptySHA(t *testing.T) {
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "no-sha",
					InstallSource: "github",
					ContentSha:    "", // empty SHA
					StorageKey:    "skills/no-sha/v1.tar.gz",
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 0 {
		t.Errorf("expected 0 resolved skills (empty SHA), got %d", len(resolved))
	}
}

func TestGetEffectiveSkills_SkipsEmptyStorageKey(t *testing.T) {
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "no-key",
					InstallSource: "github",
					ContentSha:    "sha-abc",
					StorageKey:    "", // empty key
				},
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 0 {
		t.Errorf("expected 0 resolved skills (empty storage key), got %d", len(resolved))
	}
}

func TestGetEffectiveSkills_SkipsWhenPresignedURLFails(t *testing.T) {
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "fail-url",
					InstallSource: "github",
					ContentSha:    "sha-abc",
					StorageKey:    "skills/fail-url/v1.tar.gz",
				},
			}, nil
		},
	}
	stor := &svcMockStorage{
		getURLFn: func(_ context.Context, key string, expiry time.Duration) (string, error) {
			return "", errors.New("storage unavailable")
		},
	}
	svc := newTestService(repo, stor, nil)

	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 0 {
		t.Errorf("expected 0 resolved skills (URL failure), got %d", len(resolved))
	}
}

// ---------------------------------------------------------------------------
// Tests: GetEffectiveSkills (repo error + market source + mixed)
// ---------------------------------------------------------------------------

func TestGetEffectiveSkills_RepoError(t *testing.T) {
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return nil, errors.New("connection refused")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "claude-code")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetEffectiveSkills_MarketSourceUsesMarketItem(t *testing.T) {
	// When install_source=market and pinned_version=nil, sha/storageKey come from MarketItem
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			marketItemID := int64(100)
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "market-skill",
					InstallSource: "market",
					MarketItemID:  &marketItemID,
					ContentSha:    "old-sha",     // should be overridden by MarketItem
					StorageKey:    "old-key",      // should be overridden by MarketItem
					PackageSize:   100,            // should be overridden by MarketItem
					PinnedVersion: nil,            // not pinned → use MarketItem
					MarketItem: &extension.SkillMarketItem{
						ID:          100,
						ContentSha:  "market-sha-latest",
						StorageKey:  "market/skills/latest.tar.gz",
						PackageSize: 4096,
					},
				},
			}, nil
		},
	}
	stor := &svcMockStorage{
		getURLFn: func(_ context.Context, key string, _ time.Duration) (string, error) {
			return "https://cdn.example.com/" + key + "?signed=1", nil
		},
	}
	svc := newTestService(repo, stor, nil)

	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved skill, got %d", len(resolved))
	}
	r := resolved[0]
	if r.ContentSha != "market-sha-latest" {
		t.Errorf("expected market SHA 'market-sha-latest', got %q", r.ContentSha)
	}
	if r.DownloadURL != "https://cdn.example.com/market/skills/latest.tar.gz?signed=1" {
		t.Errorf("expected market download URL, got %q", r.DownloadURL)
	}
	if r.PackageSize != 4096 {
		t.Errorf("expected market package size 4096, got %d", r.PackageSize)
	}
}

func TestGetEffectiveSkills_MixedValidAndInvalid(t *testing.T) {
	repo := &svcMockRepo{
		getEffectiveSkillsFn: func(_ context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
			return []*extension.InstalledSkill{
				{
					ID:            1,
					Slug:          "valid-skill",
					InstallSource: "github",
					ContentSha:    "sha-valid",
					StorageKey:    "skills/valid/v1.tar.gz",
					PackageSize:   512,
				},
				{
					ID:            2,
					Slug:          "no-sha-skill",
					InstallSource: "github",
					ContentSha:    "",
					StorageKey:    "skills/no-sha/v1.tar.gz",
				},
				{
					ID:            3,
					Slug:          "no-key-skill",
					InstallSource: "github",
					ContentSha:    "sha-exists",
					StorageKey:    "",
				},
				{
					ID:            4,
					Slug:          "url-fail-skill",
					InstallSource: "github",
					ContentSha:    "sha-fail",
					StorageKey:    "skills/fail/v1.tar.gz",
					PackageSize:   256,
				},
				{
					ID:            5,
					Slug:          "another-valid",
					InstallSource: "github",
					ContentSha:    "sha-another",
					StorageKey:    "skills/another/v1.tar.gz",
					PackageSize:   1024,
				},
			}, nil
		},
	}
	stor := &svcMockStorage{
		getURLFn: func(_ context.Context, key string, _ time.Duration) (string, error) {
			if key == "skills/fail/v1.tar.gz" {
				return "", errors.New("presign failed")
			}
			return "https://cdn.example.com/" + key + "?signed=1", nil
		},
	}
	svc := newTestService(repo, stor, nil)

	resolved, err := svc.GetEffectiveSkills(context.Background(), 1, 2, 3, "claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only valid-skill and another-valid should be resolved
	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved skills (skipping no-sha, no-key, url-fail), got %d", len(resolved))
	}
	if resolved[0].Slug != "valid-skill" {
		t.Errorf("expected first resolved slug 'valid-skill', got %q", resolved[0].Slug)
	}
	if resolved[1].Slug != "another-valid" {
		t.Errorf("expected second resolved slug 'another-valid', got %q", resolved[1].Slug)
	}
}

// ---------------------------------------------------------------------------
// Tests: InstallMcpFromMarket (market item not found)
// ---------------------------------------------------------------------------
