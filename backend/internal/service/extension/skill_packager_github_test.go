package extension

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// --- Tests for PackageFromGitHub ---

func TestPackageFromGitHub_Success(t *testing.T) {
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)
	packager.gitCloneFn = fakePackagerGitClone(map[string]string{
		"SKILL.md": "---\nname: github-skill\ndescription: From GitHub\n---\n# GitHub Skill",
	})

	pkg, err := packager.PackageFromGitHub(context.Background(), "https://github.com/org/skill", "", "")
	if err != nil {
		t.Fatalf("PackageFromGitHub failed: %v", err)
	}
	if pkg.Slug != "github-skill" {
		t.Errorf("expected slug 'github-skill', got %q", pkg.Slug)
	}
	if pkg.ContentSha == "" {
		t.Error("expected non-empty content SHA")
	}
	if len(store.uploaded) != 1 {
		t.Errorf("expected 1 upload, got %d", len(store.uploaded))
	}
}

func TestPackageFromGitHub_WithPath(t *testing.T) {
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)
	packager.gitCloneFn = fakePackagerGitClone(map[string]string{
		"skills/my-skill/SKILL.md": "---\nname: path-skill\n---\n",
		"README.md":                "# Root readme",
	})

	pkg, err := packager.PackageFromGitHub(context.Background(), "https://github.com/org/repo", "", "skills/my-skill")
	if err != nil {
		t.Fatalf("PackageFromGitHub with path failed: %v", err)
	}
	if pkg.Slug != "path-skill" {
		t.Errorf("expected slug 'path-skill', got %q", pkg.Slug)
	}
}

func TestPackageFromGitHub_PathNotFound(t *testing.T) {
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)
	packager.gitCloneFn = fakePackagerGitClone(map[string]string{
		"SKILL.md": "---\nname: test\n---\n",
	})

	_, err := packager.PackageFromGitHub(context.Background(), "https://github.com/org/repo", "", "nonexistent/path")
	if err == nil {
		t.Fatal("expected error for path not found, got nil")
	}
	if !strings.Contains(err.Error(), "not found in repository") {
		t.Errorf("expected 'not found in repository' error, got %q", err.Error())
	}
}

func TestPackageFromGitHub_NoSkillMD(t *testing.T) {
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)
	packager.gitCloneFn = fakePackagerGitClone(map[string]string{
		"README.md": "# No skill",
	})

	_, err := packager.PackageFromGitHub(context.Background(), "https://github.com/org/repo", "", "")
	if err == nil {
		t.Fatal("expected error for missing SKILL.md, got nil")
	}
	if !strings.Contains(err.Error(), "SKILL.md not found") {
		t.Errorf("expected 'SKILL.md not found' error, got %q", err.Error())
	}
}

func TestPackageFromGitHub_CloneError(t *testing.T) {
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)
	packager.gitCloneFn = func(_ context.Context, _, _, _ string) error {
		return errors.New("clone failed")
	}

	_, err := packager.PackageFromGitHub(context.Background(), "https://github.com/org/repo", "", "")
	if err == nil {
		t.Fatal("expected error for clone failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to clone") {
		t.Errorf("expected 'failed to clone' error, got %q", err.Error())
	}
}

func TestPackageFromGitHub_PathTraversal(t *testing.T) {
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)
	packager.gitCloneFn = fakePackagerGitClone(map[string]string{
		"SKILL.md": "---\nname: test\n---\n# Test",
	})

	_, err := packager.PackageFromGitHub(context.Background(), "https://github.com/org/repo", "", "../../etc")
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "escapes repository directory") {
		t.Errorf("expected 'escapes repository directory' error, got %q", err.Error())
	}
}

func TestPackageFromGitHub_PathTraversal_DotDot(t *testing.T) {
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)
	packager.gitCloneFn = fakePackagerGitClone(map[string]string{
		"SKILL.md": "---\nname: test\n---\n# Test",
	})

	_, err := packager.PackageFromGitHub(context.Background(), "https://github.com/org/repo", "", "../escape")
	if err == nil {
		t.Fatal("expected error for path traversal with .., got nil")
	}
	if !strings.Contains(err.Error(), "escapes repository directory") {
		t.Errorf("expected 'escapes repository directory' error, got %q", err.Error())
	}
}

func TestPackageFromGitHub_PathTraversal_AbsolutePath(t *testing.T) {
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)
	packager.gitCloneFn = fakePackagerGitClone(map[string]string{
		"SKILL.md": "---\nname: test\n---\n# Test",
	})

	_, err := packager.PackageFromGitHub(context.Background(), "https://github.com/org/repo", "", "skills/../../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "escapes repository directory") && !strings.Contains(err.Error(), "not found in repository") {
		t.Errorf("expected path traversal or not found error, got %q", err.Error())
	}
}

func TestPackageFromGitHub_FindSkillDir_Error(t *testing.T) {
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)
	packager.gitCloneFn = fakePackagerGitClone(map[string]string{
		"only-readme.md": "# No skill here",
	})

	_, err := packager.PackageFromGitHub(context.Background(), "https://github.com/org/repo", "", "")
	if err == nil {
		t.Fatal("expected error for missing SKILL.md, got nil")
	}
}

// --- Tests for CompleteGitHubInstall ---

func TestCompleteGitHubInstall(t *testing.T) {
	t.Run("invalid_scope", func(t *testing.T) {
		store := newPackagerMockStorage()
		repo := newPackagerMockRepo()
		packager := NewSkillPackager(repo, store)

		_, err := packager.CompleteGitHubInstall(
			context.Background(),
			1, 2, 3,
			"https://github.com/example/skill", "", "",
			"invalid",
		)
		if err == nil {
			t.Fatal("expected error for invalid scope, got nil")
		}
		if !strings.Contains(err.Error(), "invalid scope") {
			t.Errorf("expected 'invalid scope' error, got %q", err.Error())
		}
	})
}

func TestCompleteGitHubInstall_Success(t *testing.T) {
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)
	packager.gitCloneFn = fakePackagerGitClone(map[string]string{
		"SKILL.md": "---\nname: gh-install\ndescription: GitHub install\n---\n# GH Skill",
	})

	skill, err := packager.CompleteGitHubInstall(
		context.Background(),
		10, 20, 30,
		"https://github.com/org/skill", "", "",
		"org",
	)
	if err != nil {
		t.Fatalf("CompleteGitHubInstall failed: %v", err)
	}
	if skill.Slug != "gh-install" {
		t.Errorf("expected slug 'gh-install', got %q", skill.Slug)
	}
	if skill.InstallSource != "github" {
		t.Errorf("expected install_source 'github', got %q", skill.InstallSource)
	}
	if skill.SourceURL != "https://github.com/org/skill" {
		t.Errorf("expected source URL 'https://github.com/org/skill', got %q", skill.SourceURL)
	}
	if skill.Scope != "org" {
		t.Errorf("expected scope 'org', got %q", skill.Scope)
	}
	if len(repo.installedSkills) != 1 {
		t.Errorf("expected 1 installed skill, got %d", len(repo.installedSkills))
	}
}

func TestCompleteGitHubInstall_WithBranchAndPath(t *testing.T) {
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)
	packager.gitCloneFn = fakePackagerGitClone(map[string]string{
		"sub/SKILL.md": "---\nname: branch-path-skill\n---\n",
	})

	skill, err := packager.CompleteGitHubInstall(
		context.Background(),
		10, 20, 30,
		"https://github.com/org/repo", "develop", "sub",
		"user",
	)
	if err != nil {
		t.Fatalf("CompleteGitHubInstall failed: %v", err)
	}
	expectedURL := "https://github.com/org/repo@develop#sub"
	if skill.SourceURL != expectedURL {
		t.Errorf("expected source URL %q, got %q", expectedURL, skill.SourceURL)
	}
	if skill.Scope != "user" {
		t.Errorf("expected scope 'user', got %q", skill.Scope)
	}
}

func TestCompleteGitHubInstall_PackageError(t *testing.T) {
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)
	packager.gitCloneFn = fakePackagerGitClone(map[string]string{
		"README.md": "# No skill",
	})

	_, err := packager.CompleteGitHubInstall(
		context.Background(),
		10, 20, 30,
		"https://github.com/org/repo", "", "",
		"org",
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCompleteGitHubInstall_CreateSkillError(t *testing.T) {
	store := newPackagerMockStorage()
	repo := &packagerMockRepoWithHook{
		createInstalledSkillFn: func(_ context.Context, _ *extension.InstalledSkill) error {
			return errors.New("db insert failed")
		},
	}
	packager := NewSkillPackager(repo, store)
	packager.gitCloneFn = fakePackagerGitClone(map[string]string{
		"SKILL.md": "---\nname: fail-skill\n---\n",
	})

	_, err := packager.CompleteGitHubInstall(
		context.Background(),
		10, 20, 30,
		"https://github.com/org/repo", "", "",
		"org",
	)
	if err == nil {
		t.Fatal("expected error for create failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create installed skill") {
		t.Errorf("expected 'failed to create installed skill' error, got %q", err.Error())
	}
}

// --- fakePackagerGitClone helper ---

func fakePackagerGitClone(skills map[string]string) func(ctx context.Context, url, branch, targetDir string) error {
	return func(ctx context.Context, url, branch, targetDir string) error {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return err
		}
		for name, content := range skills {
			filePath := filepath.Join(targetDir, name)
			dir := filepath.Dir(filePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				return err
			}
		}
		return nil
	}
}
