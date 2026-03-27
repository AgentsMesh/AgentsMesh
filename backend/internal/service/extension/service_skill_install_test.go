package extension

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// ---------------------------------------------------------------------------
// Tests: InstallSkillFromMarket
// ---------------------------------------------------------------------------

func TestInstallSkillFromMarket_Success(t *testing.T) {
	marketItemID := int64(100)
	repo := &svcMockRepo{
		getSkillMarketItemFn: func(_ context.Context, id int64) (*extension.SkillMarketItem, error) {
			return &extension.SkillMarketItem{
				ID:          id,
				Slug:        "test-skill",
				ContentSha:  "abc123",
				StorageKey:  "skills/test-skill/v1.tar.gz",
				PackageSize: 1024,
			}, nil
		},
		createInstalledSkillFn: func(_ context.Context, skill *extension.InstalledSkill) error {
			if skill.Slug != "test-skill" {
				t.Errorf("expected slug 'test-skill', got %q", skill.Slug)
			}
			if skill.InstallSource != "market" {
				t.Errorf("expected install_source 'market', got %q", skill.InstallSource)
			}
			if skill.ContentSha != "abc123" {
				t.Errorf("expected content_sha 'abc123', got %q", skill.ContentSha)
			}
			if skill.StorageKey != "skills/test-skill/v1.tar.gz" {
				t.Errorf("expected storage_key, got %q", skill.StorageKey)
			}
			if skill.PackageSize != 1024 {
				t.Errorf("expected package_size 1024, got %d", skill.PackageSize)
			}
			if skill.MarketItemID == nil || *skill.MarketItemID != marketItemID {
				t.Errorf("expected market_item_id %d, got %v", marketItemID, skill.MarketItemID)
			}
			if !skill.IsEnabled {
				t.Error("expected is_enabled true")
			}
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	result, err := svc.InstallSkillFromMarket(context.Background(), 1, 2, 3, marketItemID, "org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Slug != "test-skill" {
		t.Errorf("expected slug 'test-skill', got %q", result.Slug)
	}
	if result.OrganizationID != 1 {
		t.Errorf("expected org_id 1, got %d", result.OrganizationID)
	}
	if result.RepositoryID != 2 {
		t.Errorf("expected repo_id 2, got %d", result.RepositoryID)
	}
	if result.Scope != "org" {
		t.Errorf("expected scope 'org', got %q", result.Scope)
	}
}

func TestInstallSkillFromMarket_InvalidScope(t *testing.T) {
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, nil)

	_, err := svc.InstallSkillFromMarket(context.Background(), 1, 2, 3, 100, "all")
	if err == nil {
		t.Fatal("expected error for invalid scope, got nil")
	}
}

func TestInstallSkillFromMarket_MarketItemNotFound(t *testing.T) {
	repo := &svcMockRepo{
		getSkillMarketItemFn: func(_ context.Context, id int64) (*extension.SkillMarketItem, error) {
			return nil, fmt.Errorf("record not found")
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.InstallSkillFromMarket(context.Background(), 1, 2, 3, 999, "org")
	if err == nil {
		t.Fatal("expected error for missing market item, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: InstallSkillFromGitHub
// ---------------------------------------------------------------------------

func TestInstallSkillFromGitHub_URLOnly(t *testing.T) {
	svc := newTestServiceWithPackager(&svcMockRepo{}, &svcMockStorage{}, nil)

	result, err := svc.InstallSkillFromGitHub(context.Background(), 1, 2, 3, "https://github.com/org/repo", "", "", "org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SourceURL != "https://github.com/org/repo" {
		t.Errorf("expected source URL 'https://github.com/org/repo', got %q", result.SourceURL)
	}
	if result.InstallSource != "github" {
		t.Errorf("expected install source 'github', got %q", result.InstallSource)
	}
}

func TestInstallSkillFromGitHub_URLAndBranch(t *testing.T) {
	svc := newTestServiceWithPackager(&svcMockRepo{}, &svcMockStorage{}, nil)

	result, err := svc.InstallSkillFromGitHub(context.Background(), 1, 2, 3, "https://github.com/org/repo", "develop", "", "org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SourceURL != "https://github.com/org/repo@develop" {
		t.Errorf("expected source URL with branch, got %q", result.SourceURL)
	}
}

func TestInstallSkillFromGitHub_URLBranchAndPath(t *testing.T) {
	repo := &svcMockRepo{}
	stor := &svcMockStorage{}
	svc := newTestServiceWithPackager(repo, stor, nil)

	// The mock git clone creates SKILL.md at the repo root, so path sub-dir
	// must also contain SKILL.md. Override gitCloneFn to place it under the path.
	svc.packager.gitCloneFn = func(_ context.Context, url, branch, targetDir string) error {
		skillDir := filepath.Join(targetDir, "skills", "my-skill")
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return err
		}
		content := "---\nslug: my-skill\nname: My Skill\n---\n# My Skill\nA test skill."
		return os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644)
	}

	result, err := svc.InstallSkillFromGitHub(context.Background(), 1, 2, 3, "https://github.com/org/repo", "main", "skills/my-skill", "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "https://github.com/org/repo@main#skills/my-skill"
	if result.SourceURL != expected {
		t.Errorf("expected source URL %q, got %q", expected, result.SourceURL)
	}
}

// ---------------------------------------------------------------------------
// Tests: UpdateSkill
// ---------------------------------------------------------------------------

func TestUpdateSkill_Success(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
			return &extension.InstalledSkill{
				ID:             id,
				OrganizationID: 1,
				IsEnabled:      true,
			}, nil
		},
		updateInstalledSkillFn: func(_ context.Context, skill *extension.InstalledSkill) error {
			if skill.IsEnabled != false {
				t.Error("expected IsEnabled to be false after update")
			}
			if skill.PinnedVersion == nil || *skill.PinnedVersion != 3 {
				t.Errorf("expected pinned version 3, got %v", skill.PinnedVersion)
			}
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	result, err := svc.UpdateSkill(context.Background(), 1, 0, 10, 100, "admin", ptrBool(false), ptrInt(3))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsEnabled != false {
		t.Error("expected IsEnabled=false")
	}
	if result.PinnedVersion == nil || *result.PinnedVersion != 3 {
		t.Errorf("expected pinned version 3, got %v", result.PinnedVersion)
	}
}

func TestUpdateSkill_IDORDifferentOrg(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
			return &extension.InstalledSkill{
				ID:             id,
				OrganizationID: 99,
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	_, err := svc.UpdateSkill(context.Background(), 1, 0, 10, 100, "admin", ptrBool(true), nil)
	if err == nil {
		t.Fatal("expected IDOR error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: UninstallSkill
// ---------------------------------------------------------------------------

func TestUninstallSkill_Success(t *testing.T) {
	deleteCalled := false
	repo := &svcMockRepo{
		getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
			return &extension.InstalledSkill{
				ID:             id,
				OrganizationID: 1,
			}, nil
		},
		deleteInstalledSkillFn: func(_ context.Context, id int64) error {
			deleteCalled = true
			if id != 10 {
				t.Errorf("expected id 10, got %d", id)
			}
			return nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.UninstallSkill(context.Background(), 1, 0, 10, 100, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleteCalled {
		t.Error("repo.DeleteInstalledSkill was not called")
	}
}

func TestUninstallSkill_IDORDifferentOrg(t *testing.T) {
	repo := &svcMockRepo{
		getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
			return &extension.InstalledSkill{
				ID:             id,
				OrganizationID: 99,
			}, nil
		},
	}
	svc := newTestService(repo, &svcMockStorage{}, nil)

	err := svc.UninstallSkill(context.Background(), 1, 0, 10, 100, "admin")
	if err == nil {
		t.Fatal("expected IDOR error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: InstallSkillFromGitHub (additional paths)
// ---------------------------------------------------------------------------

func TestInstallSkillFromGitHub_PathOnlyNoBranch(t *testing.T) {
	repo := &svcMockRepo{}
	stor := &svcMockStorage{}
	svc := newTestServiceWithPackager(repo, stor, nil)

	// Override gitCloneFn to place SKILL.md under the path sub-directory
	svc.packager.gitCloneFn = func(_ context.Context, url, branch, targetDir string) error {
		skillDir := filepath.Join(targetDir, "skills", "my-skill")
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return err
		}
		content := "---\nslug: my-skill\nname: My Skill\n---\n# My Skill\nA test skill."
		return os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644)
	}

	result, err := svc.InstallSkillFromGitHub(context.Background(), 1, 2, 3, "https://github.com/org/repo", "", "skills/my-skill", "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// When branch is empty but path is given, path is appended with #
	expected := "https://github.com/org/repo#skills/my-skill"
	if result.SourceURL != expected {
		t.Errorf("expected source URL %q, got %q", expected, result.SourceURL)
	}
}

// ---------------------------------------------------------------------------
// Tests: InstallCustomMcpServer (no env vars)
// ---------------------------------------------------------------------------

func TestInstallSkillFromGitHub_WithBranchAndPath(t *testing.T) {
	var captured *extension.InstalledSkill
	repo := &svcMockRepo{
		createInstalledSkillFn: func(_ context.Context, skill *extension.InstalledSkill) error {
			skill.ID = 1
			captured = skill
			return nil
		},
	}
	svc := newTestServiceWithPackager(repo, &svcMockStorage{}, nil)

	// Override gitCloneFn to place SKILL.md under the path sub-directory
	svc.packager.gitCloneFn = func(_ context.Context, url, branch, targetDir string) error {
		skillDir := filepath.Join(targetDir, "skills", "my-skill")
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return err
		}
		content := "---\nslug: my-skill\nname: My Skill\n---\n# My Skill\nA test skill."
		return os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644)
	}

	result, err := svc.InstallSkillFromGitHub(context.Background(), 1, 2, 3, "https://github.com/org/repo", "develop", "skills/my-skill", "org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SourceURL != "https://github.com/org/repo@develop#skills/my-skill" {
		t.Errorf("expected source URL with branch and path, got %q", result.SourceURL)
	}
	if captured.InstallSource != "github" {
		t.Errorf("expected install_source 'github', got %q", captured.InstallSource)
	}
}

func TestInstallSkillFromGitHub_WithBranchOnly(t *testing.T) {
	repo := &svcMockRepo{
		createInstalledSkillFn: func(_ context.Context, skill *extension.InstalledSkill) error {
			skill.ID = 1
			return nil
		},
	}
	svc := newTestServiceWithPackager(repo, &svcMockStorage{}, nil)

	result, err := svc.InstallSkillFromGitHub(context.Background(), 1, 2, 3, "https://github.com/org/repo", "main", "", "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SourceURL != "https://github.com/org/repo@main" {
		t.Errorf("expected source URL with branch, got %q", result.SourceURL)
	}
}

func TestInstallSkillFromGitHub_NoPathNoBranch(t *testing.T) {
	repo := &svcMockRepo{
		createInstalledSkillFn: func(_ context.Context, skill *extension.InstalledSkill) error {
			skill.ID = 1
			return nil
		},
	}
	svc := newTestServiceWithPackager(repo, &svcMockStorage{}, nil)

	result, err := svc.InstallSkillFromGitHub(context.Background(), 1, 2, 3, "https://github.com/org/repo", "", "", "org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SourceURL != "https://github.com/org/repo" {
		t.Errorf("expected plain source URL, got %q", result.SourceURL)
	}
}

// ---------------------------------------------------------------------------
// Tests: UpdateMcpServer — orgID mismatch
// ---------------------------------------------------------------------------
