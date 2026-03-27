package extension

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// computeDirSHA
// =============================================================================

func TestComputeDirSHA_Deterministic(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("hello"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("world"), 0644))

	sha1, err := computeDirSHA(dir)
	require.NoError(t, err)

	sha2, err := computeDirSHA(dir)
	require.NoError(t, err)

	assert.Equal(t, sha1, sha2, "same contents should produce same SHA")
	assert.Len(t, sha1, 64, "SHA256 hex should be 64 characters")
}

func TestComputeDirSHA_DifferentContent(t *testing.T) {
	dir1 := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir1, "file.txt"), []byte("content-a"), 0644))

	dir2 := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir2, "file.txt"), []byte("content-b"), 0644))

	sha1, err := computeDirSHA(dir1)
	require.NoError(t, err)

	sha2, err := computeDirSHA(dir2)
	require.NoError(t, err)

	assert.NotEqual(t, sha1, sha2, "different content should produce different SHA")
}

func TestComputeDirSHA_FileOrderDoesNotMatter(t *testing.T) {
	// Create two directories with the same files but created in different order
	dir1 := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir1, "aaa.txt"), []byte("first"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir1, "zzz.txt"), []byte("second"), 0644))

	dir2 := t.TempDir()
	// Write in reverse order
	require.NoError(t, os.WriteFile(filepath.Join(dir2, "zzz.txt"), []byte("second"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir2, "aaa.txt"), []byte("first"), 0644))

	sha1, err := computeDirSHA(dir1)
	require.NoError(t, err)

	sha2, err := computeDirSHA(dir2)
	require.NoError(t, err)

	assert.Equal(t, sha1, sha2, "file creation order should not matter (sorted internally)")
}

func TestComputeDirSHA_IgnoresGitDirectory(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644))

	sha1, err := computeDirSHA(dir)
	require.NoError(t, err)

	// Add a .git directory
	gitDir := filepath.Join(dir, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main"), 0644))

	sha2, err := computeDirSHA(dir)
	require.NoError(t, err)

	assert.Equal(t, sha1, sha2, ".git directory should be ignored")
}

func TestComputeDirSHA_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	sha, err := computeDirSHA(dir)
	require.NoError(t, err)
	assert.Len(t, sha, 64, "SHA256 hex should be 64 characters")
}

func TestComputeDirSHA_WithSubdirectory(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "file.txt"), []byte("data"), 0644))

	sha, err := computeDirSHA(dir)
	require.NoError(t, err)
	assert.Len(t, sha, 64)
}

func TestComputeDirSHA_NonexistentDir(t *testing.T) {
	_, err := computeDirSHA("/nonexistent/dir")
	assert.Error(t, err)
}

func TestComputeDirSHA_WalkError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}
	dir := t.TempDir()
	unreadableDir := filepath.Join(dir, "secrets")
	require.NoError(t, os.MkdirAll(unreadableDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(unreadableDir, "key.pem"), []byte("data"), 0644))
	require.NoError(t, os.Chmod(unreadableDir, 0000))
	defer os.Chmod(unreadableDir, 0755)

	_, err := computeDirSHA(dir)
	assert.Error(t, err, "should fail with unreadable subdirectory")
}

func TestComputeDirSHA_UnreadableFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}
	dir := t.TempDir()
	f := filepath.Join(dir, "secret.txt")
	require.NoError(t, os.WriteFile(f, []byte("secret"), 0644))
	require.NoError(t, os.Chmod(f, 0000))
	defer os.Chmod(f, 0644)

	_, err := computeDirSHA(dir)
	assert.Error(t, err, "should fail with unreadable file")
}

// =============================================================================
// packageSkillDir
// =============================================================================

func TestPackageSkillDir_CreatesValidTarGz(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\n---"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "subdir"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "subdir", "helper.txt"), []byte("helper content"), 0644))

	data, err := packageSkillDir(dir)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Decompress and verify tar entries
	gr, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer gr.Close()

	tr := tar.NewReader(gr)
	files := map[string]string{}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		if !header.FileInfo().IsDir() {
			content, err := io.ReadAll(tr)
			require.NoError(t, err)
			files[header.Name] = string(content)
		}
	}

	assert.Contains(t, files, "SKILL.md")
	assert.Equal(t, "---\nname: test\n---", files["SKILL.md"])
	assert.Contains(t, files, filepath.Join("subdir", "helper.txt"))
	assert.Equal(t, "helper content", files[filepath.Join("subdir", "helper.txt")])
}

func TestPackageSkillDir_SkipsGitAndIgnoredDirs(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("skill content"), 0644))

	// Create directories that should be ignored
	for _, ignored := range []string{".git", "node_modules", "__pycache__"} {
		ignoredPath := filepath.Join(dir, ignored)
		require.NoError(t, os.MkdirAll(ignoredPath, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(ignoredPath, "data.txt"), []byte("ignored"), 0644))
	}

	data, err := packageSkillDir(dir)
	require.NoError(t, err)

	// Extract and check
	gr, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer gr.Close()

	tr := tar.NewReader(gr)
	var names []string
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		names = append(names, header.Name)
	}

	assert.Equal(t, []string{"SKILL.md"}, names, "should only contain SKILL.md, not ignored dirs")
}

func TestPackageSkillDir_ExtractedFilesMatchOriginal(t *testing.T) {
	dir := t.TempDir()
	originalFiles := map[string]string{
		"SKILL.md":           "---\nname: verify-test\n---\n# Test",
		"config.yaml":        "key: value\n",
		"scripts/install.sh": "#!/bin/bash\necho hello\n",
	}
	for relPath, content := range originalFiles {
		absPath := filepath.Join(dir, relPath)
		require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0755))
		require.NoError(t, os.WriteFile(absPath, []byte(content), 0644))
	}

	data, err := packageSkillDir(dir)
	require.NoError(t, err)

	// Extract into a new temp dir
	extractDir := t.TempDir()
	gr, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		target := filepath.Join(extractDir, header.Name)
		if header.FileInfo().IsDir() {
			require.NoError(t, os.MkdirAll(target, 0755))
		} else {
			require.NoError(t, os.MkdirAll(filepath.Dir(target), 0755))
			content, err := io.ReadAll(tr)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(target, content, 0644))
		}
	}

	// Verify each original file exists with the same content in the extracted dir
	for relPath, expectedContent := range originalFiles {
		actual, err := os.ReadFile(filepath.Join(extractDir, relPath))
		require.NoError(t, err, "file %s should exist in archive", relPath)
		assert.Equal(t, expectedContent, string(actual), "content mismatch for %s", relPath)
	}
}

func TestPackageSkillDir_WithGitDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\n---"), 0644))

	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git", "objects"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".git", "HEAD"), []byte("ref: refs/heads/main"), 0644))

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "node_modules", "package"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "node_modules", "package", "index.js"), []byte("exports = {}"), 0644))

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "src", "main.py"), []byte("print('hello')"), 0644))

	data, err := packageSkillDir(dir)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	gz, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	tr := tar.NewReader(gz)

	var foundFiles []string
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		foundFiles = append(foundFiles, header.Name)
	}

	assert.Contains(t, foundFiles, "SKILL.md")
	assert.Contains(t, foundFiles, filepath.Join("src", "main.py"))

	for _, f := range foundFiles {
		assert.False(t, strings.HasPrefix(f, ".git"), "should not contain .git: %s", f)
		assert.False(t, strings.HasPrefix(f, "node_modules"), "should not contain node_modules: %s", f)
	}
}

func TestPackageSkillDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	data, err := packageSkillDir(dir)
	require.NoError(t, err)
	assert.NotEmpty(t, data, "even empty dir should produce a valid tar.gz")
}

func TestPackageSkillDir_WalkError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\n---"), 0644))

	unreadableDir := filepath.Join(dir, "secrets")
	require.NoError(t, os.MkdirAll(unreadableDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(unreadableDir, "key.pem"), []byte("data"), 0644))
	require.NoError(t, os.Chmod(unreadableDir, 0000))
	defer os.Chmod(unreadableDir, 0755)

	_, err := packageSkillDir(dir)
	assert.Error(t, err, "should fail with unreadable subdirectory")
}

func TestPackageSkillDir_NonexistentDir(t *testing.T) {
	_, err := packageSkillDir("/nonexistent/path/to/skill")
	assert.Error(t, err, "should fail with nonexistent directory")
}

func TestPackageSkillDir_FileOpenError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}
	dir := t.TempDir()
	f := filepath.Join(dir, "unreadable.txt")
	require.NoError(t, os.WriteFile(f, []byte("data"), 0644))
	require.NoError(t, os.Chmod(f, 0000))
	defer os.Chmod(f, 0644)

	_, err := packageSkillDir(dir)
	assert.Error(t, err, "should fail when file cannot be opened")
}

func TestPackageSkillDir_SocketFileError(t *testing.T) {
	// tar.FileInfoHeader returns an error for socket files.
	// Use a short temp dir path because Unix sockets have a path length limit.
	dir, err := os.MkdirTemp("/tmp", "skl-")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\n---"), 0644))

	socketPath := filepath.Join(dir, "s.sock")
	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer listener.Close()

	_, err = packageSkillDir(dir)
	assert.Error(t, err, "should fail when encountering a socket file")
	assert.Contains(t, err.Error(), "sockets not supported")
}
