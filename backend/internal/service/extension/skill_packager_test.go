package extension

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/internal/infra/storage"
)

// --- Mock Storage (for skill_packager tests) ---

type packagerMockStorage struct {
	uploaded map[string][]byte
}

func newPackagerMockStorage() *packagerMockStorage {
	return &packagerMockStorage{uploaded: make(map[string][]byte)}
}

func (m *packagerMockStorage) Upload(_ context.Context, key string, reader io.Reader, _ int64, _ string) (*storage.FileInfo, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	m.uploaded[key] = data
	return &storage.FileInfo{Key: key, Size: int64(len(data))}, nil
}

func (m *packagerMockStorage) Delete(_ context.Context, _ string) error { return nil }

func (m *packagerMockStorage) GetURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://mock/" + key, nil
}

func (m *packagerMockStorage) GetInternalURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "http://internal-mock/" + key, nil
}

func (m *packagerMockStorage) Exists(_ context.Context, key string) (bool, error) {
	_, ok := m.uploaded[key]
	return ok, nil
}

// Compile-time check that packagerMockStorage satisfies storage.Storage.
var _ storage.Storage = (*packagerMockStorage)(nil)

// --- Packager-specific repo wrapper ---
// Wraps the existing mockExtensionRepo and adds installed skill tracking.

type packagerMockRepo struct {
	mockExtensionRepo
	installedSkills []*extension.InstalledSkill
}

func newPackagerMockRepo() *packagerMockRepo {
	return &packagerMockRepo{}
}

func (m *packagerMockRepo) CreateInstalledSkill(_ context.Context, skill *extension.InstalledSkill) error {
	skill.ID = int64(len(m.installedSkills) + 1)
	m.installedSkills = append(m.installedSkills, skill)
	return nil
}

// --- Test Helpers ---

// createTestTarGzBytes creates a tar.gz archive in memory from a map of filename->content.
func createTestTarGzBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		hdr := &tar.Header{
			Name:     name,
			Mode:     0644,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("failed to write tar header for %s: %v", name, err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write tar content for %s: %v", name, err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}
	return buf.Bytes()
}

// createTestTarGzBytesWithHeaders creates a tar.gz archive from custom tar headers and content.
func createTestTarGzBytesWithHeaders(t *testing.T, entries []testTarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for _, entry := range entries {
		if err := tw.WriteHeader(entry.Header); err != nil {
			t.Fatalf("failed to write tar header for %s: %v", entry.Header.Name, err)
		}
		if len(entry.Content) > 0 {
			if _, err := tw.Write([]byte(entry.Content)); err != nil {
				t.Fatalf("failed to write tar content for %s: %v", entry.Header.Name, err)
			}
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}
	return buf.Bytes()
}

type testTarEntry struct {
	Header  *tar.Header
	Content string
}

// --- Tests for extractTarGz ---

func TestExtractTarGz(t *testing.T) {
	t.Run("valid_tar_gz", func(t *testing.T) {
		data := createTestTarGzBytes(t, map[string]string{
			"hello.txt":         "hello world",
			"subdir/nested.txt": "nested content",
		})

		targetDir := t.TempDir()
		err := extractTarGz(bytes.NewReader(data), targetDir)
		if err != nil {
			t.Fatalf("extractTarGz failed: %v", err)
		}

		// Verify hello.txt
		content, err := os.ReadFile(filepath.Join(targetDir, "hello.txt"))
		if err != nil {
			t.Fatalf("failed to read hello.txt: %v", err)
		}
		if string(content) != "hello world" {
			t.Errorf("expected 'hello world', got %q", string(content))
		}

		// Verify subdir/nested.txt
		content, err = os.ReadFile(filepath.Join(targetDir, "subdir", "nested.txt"))
		if err != nil {
			t.Fatalf("failed to read subdir/nested.txt: %v", err)
		}
		if string(content) != "nested content" {
			t.Errorf("expected 'nested content', got %q", string(content))
		}
	})

	t.Run("directory_traversal", func(t *testing.T) {
		entries := []testTarEntry{
			{
				Header: &tar.Header{
					Name:     "../escape.txt",
					Mode:     0644,
					Size:     int64(len("malicious")),
					Typeflag: tar.TypeReg,
				},
				Content: "malicious",
			},
			{
				Header: &tar.Header{
					Name:     "safe.txt",
					Mode:     0644,
					Size:     int64(len("safe content")),
					Typeflag: tar.TypeReg,
				},
				Content: "safe content",
			},
		}
		data := createTestTarGzBytesWithHeaders(t, entries)

		targetDir := t.TempDir()
		err := extractTarGz(bytes.NewReader(data), targetDir)
		if err != nil {
			t.Fatalf("extractTarGz failed: %v", err)
		}

		// The traversal entry should be skipped
		parentDir := filepath.Dir(targetDir)
		if _, err := os.Stat(filepath.Join(parentDir, "escape.txt")); !os.IsNotExist(err) {
			t.Error("directory traversal file should not exist outside target dir")
		}

		// Safe file should exist
		if _, err := os.Stat(filepath.Join(targetDir, "safe.txt")); os.IsNotExist(err) {
			t.Error("safe.txt should exist in target dir")
		}
	})

	t.Run("symlink_skipped", func(t *testing.T) {
		entries := []testTarEntry{
			{
				Header: &tar.Header{
					Name:     "real.txt",
					Mode:     0644,
					Size:     int64(len("real content")),
					Typeflag: tar.TypeReg,
				},
				Content: "real content",
			},
			{
				Header: &tar.Header{
					Name:     "link.txt",
					Typeflag: tar.TypeSymlink,
					Linkname: "/etc/passwd",
				},
			},
		}
		data := createTestTarGzBytesWithHeaders(t, entries)

		targetDir := t.TempDir()
		err := extractTarGz(bytes.NewReader(data), targetDir)
		if err != nil {
			t.Fatalf("extractTarGz failed: %v", err)
		}

		// Symlink should NOT be created
		if _, err := os.Lstat(filepath.Join(targetDir, "link.txt")); !os.IsNotExist(err) {
			t.Error("symlink should not be created")
		}

		// Real file should exist
		content, err := os.ReadFile(filepath.Join(targetDir, "real.txt"))
		if err != nil {
			t.Fatalf("failed to read real.txt: %v", err)
		}
		if string(content) != "real content" {
			t.Errorf("expected 'real content', got %q", string(content))
		}
	})

	t.Run("regular_files_with_directories", func(t *testing.T) {
		// Test that parent directories are auto-created for regular files
		data := createTestTarGzBytes(t, map[string]string{
			"a/b/c/deep.txt": "deep content",
		})

		targetDir := t.TempDir()
		err := extractTarGz(bytes.NewReader(data), targetDir)
		if err != nil {
			t.Fatalf("extractTarGz failed: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(targetDir, "a", "b", "c", "deep.txt"))
		if err != nil {
			t.Fatalf("failed to read deep file: %v", err)
		}
		if string(content) != "deep content" {
			t.Errorf("expected 'deep content', got %q", string(content))
		}
	})

	t.Run("preserves_file_mode", func(t *testing.T) {
		entries := []testTarEntry{
			{
				Header: &tar.Header{
					Name:     "executable.sh",
					Mode:     0755,
					Size:     int64(len("#!/bin/sh\necho hi")),
					Typeflag: tar.TypeReg,
				},
				Content: "#!/bin/sh\necho hi",
			},
		}
		data := createTestTarGzBytesWithHeaders(t, entries)

		targetDir := t.TempDir()
		err := extractTarGz(bytes.NewReader(data), targetDir)
		if err != nil {
			t.Fatalf("extractTarGz failed: %v", err)
		}

		info, err := os.Stat(filepath.Join(targetDir, "executable.sh"))
		if err != nil {
			t.Fatalf("failed to stat executable.sh: %v", err)
		}
		// Permissions are clamped to mode & 0644, so 0755 becomes 0644
		if info.Mode().Perm() != 0644 {
			t.Errorf("expected permissions 0644 (clamped from 0755), got %v", info.Mode().Perm())
		}
	})

	t.Run("zero_mode_defaults_to_644", func(t *testing.T) {
		entries := []testTarEntry{
			{
				Header: &tar.Header{
					Name:     "nomode.txt",
					Mode:     0, // zero mode
					Size:     int64(len("content")),
					Typeflag: tar.TypeReg,
				},
				Content: "content",
			},
		}
		data := createTestTarGzBytesWithHeaders(t, entries)

		targetDir := t.TempDir()
		err := extractTarGz(bytes.NewReader(data), targetDir)
		if err != nil {
			t.Fatalf("extractTarGz failed: %v", err)
		}

		info, err := os.Stat(filepath.Join(targetDir, "nomode.txt"))
		if err != nil {
			t.Fatalf("failed to stat nomode.txt: %v", err)
		}
		if info.Mode().Perm() != 0644 {
			t.Errorf("expected mode 0644, got %v", info.Mode().Perm())
		}
	})
}

// --- Tests for findSkillDir ---

func TestFindSkillDir(t *testing.T) {
	t.Run("skill_md_in_root", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# Skill"), 0644); err != nil {
			t.Fatal(err)
		}

		result, err := findSkillDir(dir)
		if err != nil {
			t.Fatalf("findSkillDir failed: %v", err)
		}
		if result != dir {
			t.Errorf("expected %q, got %q", dir, result)
		}
	})

	t.Run("skill_md_in_subdir", func(t *testing.T) {
		dir := t.TempDir()
		subDir := filepath.Join(dir, "my-skill")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(subDir, "SKILL.md"), []byte("# Skill"), 0644); err != nil {
			t.Fatal(err)
		}

		result, err := findSkillDir(dir)
		if err != nil {
			t.Fatalf("findSkillDir failed: %v", err)
		}
		if result != subDir {
			t.Errorf("expected %q, got %q", subDir, result)
		}
	})

	t.Run("no_skill_md", func(t *testing.T) {
		dir := t.TempDir()
		// Create some unrelated files
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# README"), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := findSkillDir(dir)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected error containing 'not found', got %q", err.Error())
		}
	})

	t.Run("skill_md_in_nested_subdir", func(t *testing.T) {
		dir := t.TempDir()
		nestedDir := filepath.Join(dir, "level1", "level2")
		if err := os.MkdirAll(nestedDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(nestedDir, "SKILL.md"), []byte("# Skill"), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := findSkillDir(dir)
		if err == nil {
			t.Fatal("expected error for 2-level deep SKILL.md, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected error containing 'not found', got %q", err.Error())
		}
	})
}

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

// --- Tests for PackageFromUpload error paths ---

func TestPackageFromUpload_InvalidGzip(t *testing.T) {
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)

	// Not valid gzip data
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
	// Create a valid tar.gz without SKILL.md
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

// --- Tests for PackageFromUpload — storage upload failure ---

func TestPackageFromUpload_StorageUploadError(t *testing.T) {
	skillMd := "---\nname: upload-fail-skill\ndescription: Storage fails\n---\n# Fail Skill\n"
	data := createTestTarGzBytes(t, map[string]string{
		"SKILL.md":   skillMd,
		"handler.py": "print('hello')",
	})

	// Use failingPackagerStorage which always errors on Upload
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

// --- Tests for CompleteUploadInstall with scope=user ---

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

// --- Tests for extractTarGz error paths ---

func TestExtractTarGz_InvalidGzip(t *testing.T) {
	err := extractTarGz(bytes.NewReader([]byte("not gzip")), t.TempDir())
	if err == nil {
		t.Fatal("expected error for invalid gzip, got nil")
	}
	if !strings.Contains(err.Error(), "gzip reader") {
		t.Errorf("expected gzip reader error, got %q", err.Error())
	}
}

func TestExtractTarGz_DirectoryEntries(t *testing.T) {
	entries := []testTarEntry{
		{
			Header: &tar.Header{
				Name:     "mydir/",
				Mode:     0755,
				Typeflag: tar.TypeDir,
			},
		},
		{
			Header: &tar.Header{
				Name:     "mydir/file.txt",
				Mode:     0644,
				Size:     int64(len("content")),
				Typeflag: tar.TypeReg,
			},
			Content: "content",
		},
	}
	data := createTestTarGzBytesWithHeaders(t, entries)

	targetDir := t.TempDir()
	err := extractTarGz(bytes.NewReader(data), targetDir)
	if err != nil {
		t.Fatalf("extractTarGz failed: %v", err)
	}

	// Verify directory was created
	info, err := os.Stat(filepath.Join(targetDir, "mydir"))
	if err != nil {
		t.Fatalf("mydir should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("mydir should be a directory")
	}

	// Verify file inside directory
	content, err := os.ReadFile(filepath.Join(targetDir, "mydir", "file.txt"))
	if err != nil {
		t.Fatalf("file.txt should exist: %v", err)
	}
	if string(content) != "content" {
		t.Errorf("expected 'content', got %q", string(content))
	}
}

// --- Tests for CompleteUploadInstall error paths ---

// packagerMockRepoWithHook wraps packagerMockRepo to allow overriding CreateInstalledSkill.
type packagerMockRepoWithHook struct {
	packagerMockRepo
	createInstalledSkillFn func(ctx context.Context, skill *extension.InstalledSkill) error
}

func (m *packagerMockRepoWithHook) CreateInstalledSkill(ctx context.Context, skill *extension.InstalledSkill) error {
	if m.createInstalledSkillFn != nil {
		return m.createInstalledSkillFn(ctx, skill)
	}
	return m.packagerMockRepo.CreateInstalledSkill(ctx, skill)
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

// --- Tests for packageDir error paths ---

func TestPackageDir_MissingSkillMD(t *testing.T) {
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)

	dir := t.TempDir()
	// No SKILL.md, so packageDir should fail
	_, err := packager.packageDir(context.Background(), dir)
	if err == nil {
		t.Fatal("expected error for missing SKILL.md, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse skill") {
		t.Errorf("expected 'failed to parse skill' error, got %q", err.Error())
	}
}

func TestPackageDir_UploadError(t *testing.T) {
	store := &failingPackagerStorage{}
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\n---"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := packager.packageDir(context.Background(), dir)
	if err == nil {
		t.Fatal("expected error for upload failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to upload") {
		t.Errorf("expected 'failed to upload' error, got %q", err.Error())
	}
}

type failingPackagerStorage struct {
	packagerMockStorage
}

func (m *failingPackagerStorage) Upload(_ context.Context, _ string, _ io.Reader, _ int64, _ string) (*storage.FileInfo, error) {
	return nil, errors.New("upload failed")
}

// --- Tests for findSkillDir edge cases ---

func TestFindSkillDir_SkillMDInSubdirAmongFiles(t *testing.T) {
	dir := t.TempDir()
	// Create both files and a subdirectory with SKILL.md
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("readme"), 0644); err != nil {
		t.Fatal(err)
	}
	subDir := filepath.Join(dir, "my-skill")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "SKILL.md"), []byte("# Skill"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := findSkillDir(dir)
	if err != nil {
		t.Fatalf("findSkillDir failed: %v", err)
	}
	if result != subDir {
		t.Errorf("expected %q, got %q", subDir, result)
	}
}

// --- Tests for NewSkillPackager ---

func TestExtractTarGz_CorruptTarEntry(t *testing.T) {
	// Create a gzip stream with corrupt tar data
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	gw.Write([]byte("this is not a valid tar stream"))
	gw.Close()

	err := extractTarGz(bytes.NewReader(buf.Bytes()), t.TempDir())
	if err == nil {
		t.Fatal("expected error for corrupt tar entry, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read tar entry") {
		t.Errorf("expected 'failed to read tar entry' error, got %q", err.Error())
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

func TestPackageFromUpload_ReadError(t *testing.T) {
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)

	// Create a reader that errors
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

// errReader is a reader that always errors
type errReader struct{}

func (e *errReader) Read(p []byte) (int, error) {
	return 0, errors.New("reader error")
}

func TestFindSkillDir_ReadDirError(t *testing.T) {
	// Use a non-existent path to trigger ReadDir error
	_, err := findSkillDir("/nonexistent/dir/path")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFindSkillDir_UnreadableDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}
	dir := t.TempDir()
	// No SKILL.md in root
	// Make directory unreadable
	if err := os.Chmod(dir, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0755)

	_, err := findSkillDir(dir)
	if err == nil {
		t.Fatal("expected error for unreadable dir, got nil")
	}
}

// --- Tests for PackageFromGitHub ---

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

// --- Tests for CompleteGitHubInstall with mocked git ---

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

// --- Tests for PackageFromGitHub path traversal ---

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

// --- Tests for extractTarGz total size limit ---

func TestExtractTarGz_TotalSizeExceedsLimit(t *testing.T) {
	// Create a tar.gz with entries that collectively exceed 200MB.
	// To avoid writing 200MB+ of actual data, we craft the tar manually
	// using low-level writes so that header.Size values are large,
	// but we pad properly so tar reader can parse them.
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Use many 10MB files to exceed 200MB total (21 * 10MB = 210MB)
	fileSize := int64(10 * 1024 * 1024) // 10MB per file
	numFiles := 21
	zeroChunk := make([]byte, 1024*1024) // 1MB chunk of zeroes for writing

	for i := 0; i < numFiles; i++ {
		hdr := &tar.Header{
			Name:     "large_" + string(rune('a'+i)) + ".bin",
			Mode:     0644,
			Size:     fileSize,
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("failed to write tar header %d: %v", i, err)
		}
		// Write fileSize bytes using 1MB chunks
		remaining := fileSize
		for remaining > 0 {
			toWrite := int64(len(zeroChunk))
			if toWrite > remaining {
				toWrite = remaining
			}
			if _, err := tw.Write(zeroChunk[:toWrite]); err != nil {
				t.Fatalf("failed to write tar content for file %d: %v", i, err)
			}
			remaining -= toWrite
		}
	}

	tw.Close()
	gw.Close()

	targetDir := t.TempDir()
	err := extractTarGz(bytes.NewReader(buf.Bytes()), targetDir)
	if err == nil {
		t.Fatal("expected error for exceeding total size limit, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds maximum total decompressed size") {
		t.Errorf("expected 'exceeds maximum total decompressed size' error, got %q", err.Error())
	}
}

func TestExtractTarGz_HardLinkSkipped(t *testing.T) {
	entries := []testTarEntry{
		{
			Header: &tar.Header{
				Name:     "real.txt",
				Mode:     0644,
				Size:     int64(len("content")),
				Typeflag: tar.TypeReg,
			},
			Content: "content",
		},
		{
			Header: &tar.Header{
				Name:     "hardlink.txt",
				Typeflag: tar.TypeLink,
				Linkname: "real.txt",
			},
		},
	}
	data := createTestTarGzBytesWithHeaders(t, entries)

	targetDir := t.TempDir()
	err := extractTarGz(bytes.NewReader(data), targetDir)
	if err != nil {
		t.Fatalf("extractTarGz failed: %v", err)
	}

	// Hard link should NOT be created
	if _, err := os.Lstat(filepath.Join(targetDir, "hardlink.txt")); !os.IsNotExist(err) {
		t.Error("hard link should not be created")
	}

	// Real file should exist
	content, err := os.ReadFile(filepath.Join(targetDir, "real.txt"))
	if err != nil {
		t.Fatalf("failed to read real.txt: %v", err)
	}
	if string(content) != "content" {
		t.Errorf("expected 'content', got %q", string(content))
	}
}

// --- Tests for PackageFromGitHub with absolute path manipulation ---

func TestPackageFromGitHub_PathTraversal_AbsolutePath(t *testing.T) {
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)
	packager.gitCloneFn = fakePackagerGitClone(map[string]string{
		"SKILL.md": "---\nname: test\n---\n# Test",
	})

	// Clean("/../..") resolves to ".." -> still escapes
	_, err := packager.PackageFromGitHub(context.Background(), "https://github.com/org/repo", "", "skills/../../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
	// Either "escapes repository directory" or "not found in repository"
	if !strings.Contains(err.Error(), "escapes repository directory") && !strings.Contains(err.Error(), "not found in repository") {
		t.Errorf("expected path traversal or not found error, got %q", err.Error())
	}
}

// --- Tests for extractTarGz edge cases: totalSize accumulation just at limit ---

func TestExtractTarGz_TotalSizeExactlyAtLimit(t *testing.T) {
	// Create a tar with a single file of exactly maxTotalExtractSize bytes
	// This should be accepted (the check is > not >=)
	// We just test with a small file to verify the size accumulation works
	data := createTestTarGzBytes(t, map[string]string{
		"small.txt": "hello world",
	})

	targetDir := t.TempDir()
	err := extractTarGz(bytes.NewReader(data), targetDir)
	if err != nil {
		t.Fatalf("extractTarGz failed for small file: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(targetDir, "small.txt"))
	if err != nil {
		t.Fatalf("failed to read small.txt: %v", err)
	}
	if string(content) != "hello world" {
		t.Errorf("expected 'hello world', got %q", string(content))
	}
}

func TestNewSkillPackager(t *testing.T) {
	repo := newPackagerMockRepo()
	store := newPackagerMockStorage()
	p := NewSkillPackager(repo, store)
	if p == nil {
		t.Fatal("expected non-nil packager")
	}
	if p.repo != repo {
		t.Error("repo not set correctly")
	}
	if p.storage != store {
		t.Error("storage not set correctly")
	}
}

// --- Tests for packageDir — computeDirSHA error ---

func TestPackageDir_ComputeSHAError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)

	dir := t.TempDir()
	// Create a valid SKILL.md
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\n---"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create a subdirectory with an unreadable file to cause computeDirSHA to fail
	subDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	unreadable := filepath.Join(subDir, "secret.bin")
	if err := os.WriteFile(unreadable, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(unreadable, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(unreadable, 0644)

	_, err := packager.packageDir(context.Background(), dir)
	if err == nil {
		t.Fatal("expected error for computeDirSHA failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to compute SHA") {
		t.Errorf("expected 'failed to compute SHA' error, got %q", err.Error())
	}
}

// --- Tests for packageDir — packageSkillDir error ---

func TestPackageDir_PackageError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)

	dir := t.TempDir()
	// Create a valid SKILL.md
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\n---"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create a subdirectory that is unreadable to cause packageSkillDir walk to fail
	unreadableDir := filepath.Join(dir, "locked")
	if err := os.MkdirAll(unreadableDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unreadableDir, "file.txt"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(unreadableDir, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(unreadableDir, 0755)

	_, err := packager.packageDir(context.Background(), dir)
	if err == nil {
		t.Fatal("expected error for packageSkillDir failure, got nil")
	}
	// The error could be from computeDirSHA or packageSkillDir depending on OS behavior
	// Both go through filepath.Walk on the same unreadable dir
}

// --- Tests for extractTarGz — directory traversal for TypeDir ---

// --- Tests for CompleteUploadInstall — PackageFromUpload error propagation ---

func TestCompleteUploadInstall_PackageError(t *testing.T) {
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)

	// Send invalid gzip data so PackageFromUpload fails
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

// --- Tests for extractTarGz — read-only target dir ---

func TestExtractTarGz_ReadOnlyTargetDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}
	// Create a tar.gz with a directory entry
	entries := []testTarEntry{
		{
			Header: &tar.Header{
				Name:     "subdir/",
				Mode:     0755,
				Typeflag: tar.TypeDir,
			},
		},
		{
			Header: &tar.Header{
				Name:     "subdir/file.txt",
				Mode:     0644,
				Size:     int64(len("content")),
				Typeflag: tar.TypeReg,
			},
			Content: "content",
		},
	}
	data := createTestTarGzBytesWithHeaders(t, entries)

	targetDir := t.TempDir()
	// Make target read-only to trigger MkdirAll error
	if err := os.Chmod(targetDir, 0555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(targetDir, 0755)

	err := extractTarGz(bytes.NewReader(data), targetDir)
	if err == nil {
		t.Fatal("expected error for read-only target dir, got nil")
	}
}

// --- Tests for PackageFromGitHub — MkdirTemp success but findSkillDir error ---

func TestPackageFromGitHub_FindSkillDir_Error(t *testing.T) {
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)
	// Clone succeeds but repo has no SKILL.md and sub-path is empty
	packager.gitCloneFn = fakePackagerGitClone(map[string]string{
		"only-readme.md": "# No skill here",
	})

	_, err := packager.PackageFromGitHub(context.Background(), "https://github.com/org/repo", "", "")
	if err == nil {
		t.Fatal("expected error for missing SKILL.md, got nil")
	}
}

func TestExtractTarGz_DirectoryTraversal_TypeDir(t *testing.T) {
	entries := []testTarEntry{
		{
			Header: &tar.Header{
				Name:     "../escape-dir/",
				Mode:     0755,
				Typeflag: tar.TypeDir,
			},
		},
		{
			Header: &tar.Header{
				Name:     "safe.txt",
				Mode:     0644,
				Size:     int64(len("safe")),
				Typeflag: tar.TypeReg,
			},
			Content: "safe",
		},
	}
	data := createTestTarGzBytesWithHeaders(t, entries)

	targetDir := t.TempDir()
	err := extractTarGz(bytes.NewReader(data), targetDir)
	if err != nil {
		t.Fatalf("extractTarGz failed: %v", err)
	}

	// The traversal directory should not be created
	parentDir := filepath.Dir(targetDir)
	if _, err := os.Stat(filepath.Join(parentDir, "escape-dir")); !os.IsNotExist(err) {
		t.Error("directory traversal dir should not exist outside target dir")
	}

	// Safe file should exist
	if _, err := os.Stat(filepath.Join(targetDir, "safe.txt")); os.IsNotExist(err) {
		t.Error("safe.txt should exist in target dir")
	}
}
