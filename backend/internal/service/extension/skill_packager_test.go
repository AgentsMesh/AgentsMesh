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

func (m *packagerMockStorage) PresignPutURL(_ context.Context, _ string, _ string, _ time.Duration) (string, error) {
	return "", nil
}

func (m *packagerMockStorage) InternalPresignPutURL(_ context.Context, _ string, _ string, _ time.Duration) (string, error) {
	return "", nil
}

var _ storage.Storage = (*packagerMockStorage)(nil)

// --- Packager-specific repo wrapper ---

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

func createTestTarGzBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		hdr := &tar.Header{Name: name, Mode: 0644, Size: int64(len(content)), Typeflag: tar.TypeReg}
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

type failingPackagerStorage struct {
	packagerMockStorage
}

func (m *failingPackagerStorage) Upload(_ context.Context, _ string, _ io.Reader, _ int64, _ string) (*storage.FileInfo, error) {
	return nil, errors.New("upload failed")
}

type errReader struct{}

func (e *errReader) Read(p []byte) (int, error) {
	return 0, errors.New("reader error")
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
		content, err := os.ReadFile(filepath.Join(targetDir, "hello.txt"))
		if err != nil {
			t.Fatalf("failed to read hello.txt: %v", err)
		}
		if string(content) != "hello world" {
			t.Errorf("expected 'hello world', got %q", string(content))
		}
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
			{Header: &tar.Header{Name: "../escape.txt", Mode: 0644, Size: int64(len("malicious")), Typeflag: tar.TypeReg}, Content: "malicious"},
			{Header: &tar.Header{Name: "safe.txt", Mode: 0644, Size: int64(len("safe content")), Typeflag: tar.TypeReg}, Content: "safe content"},
		}
		data := createTestTarGzBytesWithHeaders(t, entries)
		targetDir := t.TempDir()
		err := extractTarGz(bytes.NewReader(data), targetDir)
		if err != nil {
			t.Fatalf("extractTarGz failed: %v", err)
		}
		parentDir := filepath.Dir(targetDir)
		if _, err := os.Stat(filepath.Join(parentDir, "escape.txt")); !os.IsNotExist(err) {
			t.Error("directory traversal file should not exist outside target dir")
		}
		if _, err := os.Stat(filepath.Join(targetDir, "safe.txt")); os.IsNotExist(err) {
			t.Error("safe.txt should exist in target dir")
		}
	})

	t.Run("symlink_skipped", func(t *testing.T) {
		entries := []testTarEntry{
			{Header: &tar.Header{Name: "real.txt", Mode: 0644, Size: int64(len("real content")), Typeflag: tar.TypeReg}, Content: "real content"},
			{Header: &tar.Header{Name: "link.txt", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd"}},
		}
		data := createTestTarGzBytesWithHeaders(t, entries)
		targetDir := t.TempDir()
		err := extractTarGz(bytes.NewReader(data), targetDir)
		if err != nil {
			t.Fatalf("extractTarGz failed: %v", err)
		}
		if _, err := os.Lstat(filepath.Join(targetDir, "link.txt")); !os.IsNotExist(err) {
			t.Error("symlink should not be created")
		}
		content, err := os.ReadFile(filepath.Join(targetDir, "real.txt"))
		if err != nil {
			t.Fatalf("failed to read real.txt: %v", err)
		}
		if string(content) != "real content" {
			t.Errorf("expected 'real content', got %q", string(content))
		}
	})

	t.Run("regular_files_with_directories", func(t *testing.T) {
		data := createTestTarGzBytes(t, map[string]string{"a/b/c/deep.txt": "deep content"})
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
			{Header: &tar.Header{Name: "executable.sh", Mode: 0755, Size: int64(len("#!/bin/sh\necho hi")), Typeflag: tar.TypeReg}, Content: "#!/bin/sh\necho hi"},
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
		if info.Mode().Perm() != 0644 {
			t.Errorf("expected permissions 0644 (clamped from 0755), got %v", info.Mode().Perm())
		}
	})

	t.Run("zero_mode_defaults_to_644", func(t *testing.T) {
		entries := []testTarEntry{
			{Header: &tar.Header{Name: "nomode.txt", Mode: 0, Size: int64(len("content")), Typeflag: tar.TypeReg}, Content: "content"},
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
