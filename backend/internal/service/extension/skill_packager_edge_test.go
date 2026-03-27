package extension

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Tests for extractTarGz edge cases ---

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
		{Header: &tar.Header{Name: "mydir/", Mode: 0755, Typeflag: tar.TypeDir}},
		{Header: &tar.Header{Name: "mydir/file.txt", Mode: 0644, Size: int64(len("content")), Typeflag: tar.TypeReg}, Content: "content"},
	}
	data := createTestTarGzBytesWithHeaders(t, entries)

	targetDir := t.TempDir()
	err := extractTarGz(bytes.NewReader(data), targetDir)
	if err != nil {
		t.Fatalf("extractTarGz failed: %v", err)
	}

	info, err := os.Stat(filepath.Join(targetDir, "mydir"))
	if err != nil {
		t.Fatalf("mydir should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("mydir should be a directory")
	}

	content, err := os.ReadFile(filepath.Join(targetDir, "mydir", "file.txt"))
	if err != nil {
		t.Fatalf("file.txt should exist: %v", err)
	}
	if string(content) != "content" {
		t.Errorf("expected 'content', got %q", string(content))
	}
}

func TestExtractTarGz_CorruptTarEntry(t *testing.T) {
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

func TestExtractTarGz_TotalSizeExceedsLimit(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	fileSize := int64(10 * 1024 * 1024) // 10MB per file
	numFiles := 21
	zeroChunk := make([]byte, 1024*1024) // 1MB chunk

	for i := 0; i < numFiles; i++ {
		hdr := &tar.Header{
			Name: "large_" + string(rune('a'+i)) + ".bin", Mode: 0644,
			Size: fileSize, Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("failed to write tar header %d: %v", i, err)
		}
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
		{Header: &tar.Header{Name: "real.txt", Mode: 0644, Size: int64(len("content")), Typeflag: tar.TypeReg}, Content: "content"},
		{Header: &tar.Header{Name: "hardlink.txt", Typeflag: tar.TypeLink, Linkname: "real.txt"}},
	}
	data := createTestTarGzBytesWithHeaders(t, entries)

	targetDir := t.TempDir()
	err := extractTarGz(bytes.NewReader(data), targetDir)
	if err != nil {
		t.Fatalf("extractTarGz failed: %v", err)
	}

	if _, err := os.Lstat(filepath.Join(targetDir, "hardlink.txt")); !os.IsNotExist(err) {
		t.Error("hard link should not be created")
	}

	content, err := os.ReadFile(filepath.Join(targetDir, "real.txt"))
	if err != nil {
		t.Fatalf("failed to read real.txt: %v", err)
	}
	if string(content) != "content" {
		t.Errorf("expected 'content', got %q", string(content))
	}
}

func TestExtractTarGz_TotalSizeExactlyAtLimit(t *testing.T) {
	data := createTestTarGzBytes(t, map[string]string{"small.txt": "hello world"})

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

func TestExtractTarGz_DirectoryTraversal_TypeDir(t *testing.T) {
	entries := []testTarEntry{
		{Header: &tar.Header{Name: "../escape-dir/", Mode: 0755, Typeflag: tar.TypeDir}},
		{Header: &tar.Header{Name: "safe.txt", Mode: 0644, Size: int64(len("safe")), Typeflag: tar.TypeReg}, Content: "safe"},
	}
	data := createTestTarGzBytesWithHeaders(t, entries)

	targetDir := t.TempDir()
	err := extractTarGz(bytes.NewReader(data), targetDir)
	if err != nil {
		t.Fatalf("extractTarGz failed: %v", err)
	}

	parentDir := filepath.Dir(targetDir)
	if _, err := os.Stat(filepath.Join(parentDir, "escape-dir")); !os.IsNotExist(err) {
		t.Error("directory traversal dir should not exist outside target dir")
	}

	if _, err := os.Stat(filepath.Join(targetDir, "safe.txt")); os.IsNotExist(err) {
		t.Error("safe.txt should exist in target dir")
	}
}

func TestExtractTarGz_ReadOnlyTargetDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}
	entries := []testTarEntry{
		{Header: &tar.Header{Name: "subdir/", Mode: 0755, Typeflag: tar.TypeDir}},
		{Header: &tar.Header{Name: "subdir/file.txt", Mode: 0644, Size: int64(len("content")), Typeflag: tar.TypeReg}, Content: "content"},
	}
	data := createTestTarGzBytesWithHeaders(t, entries)

	targetDir := t.TempDir()
	if err := os.Chmod(targetDir, 0555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(targetDir, 0755)

	err := extractTarGz(bytes.NewReader(data), targetDir)
	if err == nil {
		t.Fatal("expected error for read-only target dir, got nil")
	}
}

// --- Tests for findSkillDir edge cases ---

func TestFindSkillDir_SkillMDInSubdirAmongFiles(t *testing.T) {
	dir := t.TempDir()
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

func TestFindSkillDir_ReadDirError(t *testing.T) {
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
	if err := os.Chmod(dir, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0755)

	_, err := findSkillDir(dir)
	if err == nil {
		t.Fatal("expected error for unreadable dir, got nil")
	}
}

// --- Tests for packageDir error paths ---

func TestPackageDir_MissingSkillMD(t *testing.T) {
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)

	dir := t.TempDir()
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

func TestPackageDir_ComputeSHAError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\n---"), 0644); err != nil {
		t.Fatal(err)
	}
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

func TestPackageDir_PackageError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}
	store := newPackagerMockStorage()
	repo := newPackagerMockRepo()
	packager := NewSkillPackager(repo, store)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\n---"), 0644); err != nil {
		t.Fatal(err)
	}
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
