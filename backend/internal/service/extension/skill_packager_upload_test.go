package extension

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// --- Tests for PackageFromUpload ---

func TestPackageFromUpload(t *testing.T) {
	skillMd := "---\nname: test-skill\ndescription: A test skill\n---\n# Test Skill\nInstructions here.\n"

	t.Run("valid_tar_gz_upload", func(t *testing.T) {
		data := createTestTarGzBytes(t, map[string]string{
			"SKILL.md":   skillMd,
			"handler.py": "print('hello')",
		})

		store := newPackagerMockStorage()
		repo := newPackagerMockRepo()
		packager := NewSkillPackager(repo, store)

		pkg, err := packager.PackageFromUpload(context.Background(), bytes.NewReader(data), "skill.tar.gz")
		if err != nil {
			t.Fatalf("PackageFromUpload failed: %v", err)
		}

		if pkg.Slug != "test-skill" {
			t.Errorf("expected slug 'test-skill', got %q", pkg.Slug)
		}
		if pkg.DisplayName != "test-skill" {
			t.Errorf("expected display name 'test-skill', got %q", pkg.DisplayName)
		}
		if pkg.Description != "A test skill" {
			t.Errorf("expected description 'A test skill', got %q", pkg.Description)
		}
		if pkg.ContentSha == "" {
			t.Error("expected non-empty content SHA")
		}
		if pkg.StorageKey == "" {
			t.Error("expected non-empty storage key")
		}
		if !strings.Contains(pkg.StorageKey, "test-skill") {
			t.Errorf("expected storage key to contain slug, got %q", pkg.StorageKey)
		}
		if pkg.PackageSize <= 0 {
			t.Errorf("expected positive package size, got %d", pkg.PackageSize)
		}

		// Verify storage received the upload
		if len(store.uploaded) != 1 {
			t.Errorf("expected 1 upload, got %d", len(store.uploaded))
		}
	})

	t.Run("unsupported_format", func(t *testing.T) {
		store := newPackagerMockStorage()
		repo := newPackagerMockRepo()
		packager := NewSkillPackager(repo, store)

		_, err := packager.PackageFromUpload(context.Background(), bytes.NewReader([]byte("dummy")), "skill.zip")
		if err == nil {
			t.Fatal("expected error for .zip file, got nil")
		}
		if !strings.Contains(err.Error(), "unsupported file format") {
			t.Errorf("expected 'unsupported file format' error, got %q", err.Error())
		}
	})

	t.Run("unsupported_format_txt", func(t *testing.T) {
		store := newPackagerMockStorage()
		repo := newPackagerMockRepo()
		packager := NewSkillPackager(repo, store)

		_, err := packager.PackageFromUpload(context.Background(), bytes.NewReader([]byte("dummy")), "skill.txt")
		if err == nil {
			t.Fatal("expected error for .txt file, got nil")
		}
		if !strings.Contains(err.Error(), "unsupported file format") {
			t.Errorf("expected 'unsupported file format' error, got %q", err.Error())
		}
	})

	t.Run("valid_tgz_extension", func(t *testing.T) {
		data := createTestTarGzBytes(t, map[string]string{
			"SKILL.md":   skillMd,
			"handler.py": "print('hello')",
		})

		store := newPackagerMockStorage()
		repo := newPackagerMockRepo()
		packager := NewSkillPackager(repo, store)

		pkg, err := packager.PackageFromUpload(context.Background(), bytes.NewReader(data), "skill.tgz")
		if err != nil {
			t.Fatalf("PackageFromUpload with .tgz failed: %v", err)
		}

		if pkg.Slug != "test-skill" {
			t.Errorf("expected slug 'test-skill', got %q", pkg.Slug)
		}
	})
}

// --- Tests for PackageFromUpload error paths ---

func TestPackageFromUpload_InvalidGzip(t *testing.T) {
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)

	_, err := packager.PackageFromUpload(
		context.Background(),
		bytes.NewReader([]byte("this is not gzip data")),
		"skill.tar.gz",
	)
	if err == nil {
		t.Fatal("expected error for invalid gzip, got nil")
	}
	if !strings.Contains(err.Error(), "extract tar.gz") {
		t.Errorf("expected extract error, got %q", err.Error())
	}
}

func TestPackageFromUpload_NoSkillMD(t *testing.T) {
	data := createTestTarGzBytes(t, map[string]string{
		"README.md":  "# No Skill",
		"handler.py": "print('hello')",
	})

	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)

	_, err := packager.PackageFromUpload(
		context.Background(),
		bytes.NewReader(data),
		"skill.tar.gz",
	)
	if err == nil {
		t.Fatal("expected error for missing SKILL.md, got nil")
	}
	if !strings.Contains(err.Error(), "SKILL.md not found") {
		t.Errorf("expected 'SKILL.md not found' error, got %q", err.Error())
	}
}

func TestPackageFromUpload_StorageUploadError(t *testing.T) {
	skillMd := "---\nname: upload-fail-skill\ndescription: Storage fails\n---\n# Fail Skill\n"
	data := createTestTarGzBytes(t, map[string]string{
		"SKILL.md":   skillMd,
		"handler.py": "print('hello')",
	})

	store := &failingPackagerStorage{}
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)

	_, err := packager.PackageFromUpload(context.Background(), bytes.NewReader(data), "skill.tar.gz")
	if err == nil {
		t.Fatal("expected error for storage upload failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to upload") {
		t.Errorf("expected 'failed to upload' error, got %q", err.Error())
	}
}

func TestPackageFromUpload_ReadError(t *testing.T) {
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)

	_, err := packager.PackageFromUpload(
		context.Background(),
		&errReader{},
		"skill.tar.gz",
	)
	if err == nil {
		t.Fatal("expected error for read failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read upload") {
		t.Errorf("expected 'failed to read upload' error, got %q", err.Error())
	}
}

// --- Tests for CompleteUploadInstall ---

func TestCompleteUploadInstall(t *testing.T) {
	skillMd := "---\nname: upload-skill\ndescription: An uploaded skill\n---\n# Upload Skill\n"

	t.Run("invalid_scope", func(t *testing.T) {
		store := newPackagerMockStorage()
		repo := newPackagerMockRepo()
		packager := NewSkillPackager(repo, store)

		data := createTestTarGzBytes(t, map[string]string{
			"SKILL.md": skillMd,
		})

		_, err := packager.CompleteUploadInstall(
			context.Background(),
			1, 2, 3,
			bytes.NewReader(data), "skill.tar.gz",
			"invalid",
		)
		if err == nil {
			t.Fatal("expected error for invalid scope, got nil")
		}
		if !strings.Contains(err.Error(), "invalid scope") {
			t.Errorf("expected 'invalid scope' error, got %q", err.Error())
		}
	})

	t.Run("valid_upload_install", func(t *testing.T) {
		store := newPackagerMockStorage()
		repo := newPackagerMockRepo()
		packager := NewSkillPackager(repo, store)

		data := createTestTarGzBytes(t, map[string]string{
			"SKILL.md":   skillMd,
			"handler.py": "print('hello')",
		})

		skill, err := packager.CompleteUploadInstall(
			context.Background(),
			10, 20, 30,
			bytes.NewReader(data), "skill.tar.gz",
			"org",
		)
		if err != nil {
			t.Fatalf("CompleteUploadInstall failed: %v", err)
		}

		if skill.OrganizationID != 10 {
			t.Errorf("expected org_id 10, got %d", skill.OrganizationID)
		}
		if skill.RepositoryID != 20 {
			t.Errorf("expected repo_id 20, got %d", skill.RepositoryID)
		}
		if skill.InstalledBy == nil || *skill.InstalledBy != 30 {
			t.Errorf("expected installed_by 30, got %v", skill.InstalledBy)
		}
		if skill.Slug != "upload-skill" {
			t.Errorf("expected slug 'upload-skill', got %q", skill.Slug)
		}
		if skill.InstallSource != "upload" {
			t.Errorf("expected install_source 'upload', got %q", skill.InstallSource)
		}
		if skill.Scope != "org" {
			t.Errorf("expected scope 'org', got %q", skill.Scope)
		}
		if !skill.IsEnabled {
			t.Error("expected skill to be enabled")
		}
		if len(repo.installedSkills) != 1 {
			t.Errorf("expected 1 installed skill in repo, got %d", len(repo.installedSkills))
		}
	})
}

func TestCompleteUploadInstall_UserScope(t *testing.T) {
	skillMd := "---\nname: user-skill\ndescription: A user-scoped skill\n---\n# User Skill\n"
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)

	data := createTestTarGzBytes(t, map[string]string{
		"SKILL.md": skillMd,
	})

	skill, err := packager.CompleteUploadInstall(
		context.Background(),
		10, 20, 30,
		bytes.NewReader(data), "skill.tar.gz",
		"user",
	)
	if err != nil {
		t.Fatalf("CompleteUploadInstall failed: %v", err)
	}

	if skill.Scope != "user" {
		t.Errorf("expected scope 'user', got %q", skill.Scope)
	}
	if skill.Slug != "user-skill" {
		t.Errorf("expected slug 'user-skill', got %q", skill.Slug)
	}
	if skill.InstallSource != "upload" {
		t.Errorf("expected install_source 'upload', got %q", skill.InstallSource)
	}
	if len(repo.installedSkills) != 1 {
		t.Errorf("expected 1 installed skill, got %d", len(repo.installedSkills))
	}
}

func TestCompleteUploadInstall_CreateInstalledSkillError(t *testing.T) {
	skillMd := "---\nname: fail-skill\n---\n"
	store := newPackagerMockStorage()
	repo := &packagerMockRepoWithHook{
		createInstalledSkillFn: func(_ context.Context, _ *extension.InstalledSkill) error {
			return errors.New("db insert failed")
		},
	}
	packager := NewSkillPackager(repo, store)

	data := createTestTarGzBytes(t, map[string]string{
		"SKILL.md": skillMd,
	})

	_, err := packager.CompleteUploadInstall(
		context.Background(),
		10, 20, 30,
		bytes.NewReader(data), "skill.tar.gz",
		"org",
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create installed skill") {
		t.Errorf("expected 'failed to create installed skill' error, got %q", err.Error())
	}
}

func TestCompleteUploadInstall_InvalidScope(t *testing.T) {
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)

	_, err := packager.CompleteUploadInstall(
		context.Background(),
		10, 20, 30,
		bytes.NewReader(nil), "skill.tar.gz",
		"invalid",
	)
	if err == nil {
		t.Fatal("expected error for invalid scope, got nil")
	}
	if !strings.Contains(err.Error(), "invalid scope") {
		t.Errorf("expected 'invalid scope' error, got %q", err.Error())
	}
}

func TestCompleteUploadInstall_PackageError(t *testing.T) {
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)

	_, err := packager.CompleteUploadInstall(
		context.Background(),
		10, 20, 30,
		bytes.NewReader([]byte("not valid gzip")), "skill.tar.gz",
		"org",
	)
	if err == nil {
		t.Fatal("expected error for invalid tar.gz data, got nil")
	}
}
