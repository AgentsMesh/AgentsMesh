package cache

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// extractTarGz directory traversal protection
// ---------------------------------------------------------------------------

func TestExtractTarGz_DirectoryTraversal(t *testing.T) {
	t.Run("path_with_dotdot", func(t *testing.T) {
		targetDir := t.TempDir()

		// Create a tar.gz with a path containing ".."
		tarGzData := createTarGzWithRawEntries(t, []tarEntry{
			{name: "../evil.txt", content: "malicious content"},
			{name: "safe.txt", content: "safe content"},
		})

		err := extractTarGz(bytes.NewReader(tarGzData), targetDir)
		require.NoError(t, err)

		// The "../evil.txt" entry should be skipped
		_, err = os.Stat(filepath.Join(targetDir, "..", "evil.txt"))
		assert.True(t, os.IsNotExist(err), "file with .. path should not be extracted")

		// The safe file should be extracted
		content, err := os.ReadFile(filepath.Join(targetDir, "safe.txt"))
		require.NoError(t, err)
		assert.Equal(t, "safe content", string(content))
	})

	t.Run("path_outside_target", func(t *testing.T) {
		targetDir := t.TempDir()

		// Create a tar.gz with an absolute path that would escape target dir.
		// The ".." check catches most cases; this tests that the prefix check
		// also works for paths that Clean() resolves outside the target.
		tarGzData := createTarGzWithRawEntries(t, []tarEntry{
			{name: "sub/../../../etc/passwd", content: "root:x:0:0"},
			{name: "legit/data.txt", content: "ok"},
		})

		err := extractTarGz(bytes.NewReader(tarGzData), targetDir)
		require.NoError(t, err)

		// The traversal entry should be skipped
		_, err = os.Stat(filepath.Join(targetDir, "..", "..", "etc", "passwd"))
		assert.True(t, os.IsNotExist(err), "path escaping target directory should not be extracted")

		// Legitimate file should be extracted
		content, err := os.ReadFile(filepath.Join(targetDir, "legit", "data.txt"))
		require.NoError(t, err)
		assert.Equal(t, "ok", string(content))
	})
}

// tarEntry represents a raw tar entry for testing.
type tarEntry struct {
	name    string
	content string
}

// createTarGzWithRawEntries creates a tar.gz archive from raw entries,
// allowing crafted paths (including malicious ones) for testing traversal protection.
func createTarGzWithRawEntries(t *testing.T, entries []tarEntry) []byte {
	t.Helper()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for _, e := range entries {
		hdr := &tar.Header{
			Name: e.name,
			Mode: 0644,
			Size: int64(len(e.content)),
		}
		err := tw.WriteHeader(hdr)
		require.NoError(t, err)
		_, err = tw.Write([]byte(e.content))
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())

	return buf.Bytes()
}

// ---------------------------------------------------------------------------
// Additional coverage tests for extractTarGz
// ---------------------------------------------------------------------------

func TestExtractTarGz_ValidWithDirs(t *testing.T) {
	targetDir := t.TempDir()

	// Build a tar.gz with explicit directory entries followed by files
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Directory entry
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     "scripts/",
		Typeflag: tar.TypeDir,
		Mode:     0755,
	}))

	// Another nested directory entry
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     "scripts/lib/",
		Typeflag: tar.TypeDir,
		Mode:     0755,
	}))

	// File inside directory
	fileContent := "#!/bin/bash\necho hello"
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     "scripts/run.sh",
		Typeflag: tar.TypeReg,
		Mode:     0755,
		Size:     int64(len(fileContent)),
	}))
	_, err := tw.Write([]byte(fileContent))
	require.NoError(t, err)

	// File inside nested directory
	libContent := "package lib"
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     "scripts/lib/utils.go",
		Typeflag: tar.TypeReg,
		Mode:     0644,
		Size:     int64(len(libContent)),
	}))
	_, err = tw.Write([]byte(libContent))
	require.NoError(t, err)

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())

	err = extractTarGz(bytes.NewReader(buf.Bytes()), targetDir)
	require.NoError(t, err)

	// Verify directories were created
	info, err := os.Stat(filepath.Join(targetDir, "scripts"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	info, err = os.Stat(filepath.Join(targetDir, "scripts", "lib"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Verify files
	data, err := os.ReadFile(filepath.Join(targetDir, "scripts", "run.sh"))
	require.NoError(t, err)
	assert.Equal(t, fileContent, string(data))

	data, err = os.ReadFile(filepath.Join(targetDir, "scripts", "lib", "utils.go"))
	require.NoError(t, err)
	assert.Equal(t, libContent, string(data))
}

func TestExtractTarGz_ZeroMode(t *testing.T) {
	targetDir := t.TempDir()

	// Build a tar.gz with a file that has mode 0
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	content := "file with zero mode should get 0644"
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     "default-mode.txt",
		Typeflag: tar.TypeReg,
		Mode:     0, // intentionally zero
		Size:     int64(len(content)),
	}))
	_, err := tw.Write([]byte(content))
	require.NoError(t, err)

	// Also add a file with explicit non-zero mode for comparison
	content2 := "file with explicit 0755 mode"
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     "explicit-mode.sh",
		Typeflag: tar.TypeReg,
		Mode:     0755,
		Size:     int64(len(content2)),
	}))
	_, err = tw.Write([]byte(content2))
	require.NoError(t, err)

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())

	err = extractTarGz(bytes.NewReader(buf.Bytes()), targetDir)
	require.NoError(t, err)

	// Verify zero-mode file gets default 0644 (Unix only; Windows has different permission model)
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(targetDir, "default-mode.txt"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
	}

	data, err := os.ReadFile(filepath.Join(targetDir, "default-mode.txt"))
	require.NoError(t, err)
	assert.Equal(t, content, string(data))

	// Verify explicit-mode file has permissions capped at 0644 (extractTarGz strips execute bits)
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(targetDir, "explicit-mode.sh"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
	}

	data, err = os.ReadFile(filepath.Join(targetDir, "explicit-mode.sh"))
	require.NoError(t, err)
	assert.Equal(t, content2, string(data))
}

func TestExtractTarGz_InvalidGzipData(t *testing.T) {
	err := extractTarGz(bytes.NewReader([]byte("not gzip data")), t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create gzip reader")
}

func TestExtractTarGz_CorruptTarInsideGzip(t *testing.T) {
	// Create valid gzip wrapping invalid tar data
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err := gw.Write([]byte("this is not valid tar data but is valid gzip content"))
	require.NoError(t, err)
	require.NoError(t, gw.Close())

	err = extractTarGz(bytes.NewReader(buf.Bytes()), t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read tar entry")
}

func TestExtractTarGz_SymlinkSkipped(t *testing.T) {
	// Symlink entries should be silently skipped (not TypeDir, not TypeReg)
	targetDir := t.TempDir()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Add a symlink entry
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     "link.txt",
		Typeflag: tar.TypeSymlink,
		Linkname: "/etc/passwd",
	}))

	// Add a normal file to verify extraction still works
	content := "normal file"
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     "normal.txt",
		Typeflag: tar.TypeReg,
		Mode:     0644,
		Size:     int64(len(content)),
	}))
	_, err := tw.Write([]byte(content))
	require.NoError(t, err)

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())

	err = extractTarGz(bytes.NewReader(buf.Bytes()), targetDir)
	require.NoError(t, err)

	// Symlink should NOT have been created
	_, err = os.Lstat(filepath.Join(targetDir, "link.txt"))
	assert.True(t, os.IsNotExist(err))

	// Normal file should exist
	data, err := os.ReadFile(filepath.Join(targetDir, "normal.txt"))
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}
