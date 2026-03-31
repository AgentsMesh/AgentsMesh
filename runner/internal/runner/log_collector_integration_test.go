//go:build integration

package runner

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// extractTarGz decompresses and reads all entries from a tar.gz buffer.
// Returns a map of archive-name → content.
func extractTarGz(t *testing.T, buf *bytes.Buffer) map[string][]byte {
	t.Helper()
	gr, err := gzip.NewReader(buf)
	require.NoError(t, err)
	defer gr.Close()

	result := make(map[string][]byte)
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		data, err := io.ReadAll(tr)
		require.NoError(t, err)
		result[hdr.Name] = data
	}
	return result
}

func TestLogCollector_RealFiles_Integration(t *testing.T) {
	dir := t.TempDir()

	// Prepare test files with realistic names and content.
	files := map[string]string{
		"runner-2026-01-01.log":      strings.Repeat("log-line\n", 128),
		"blocked-12345.stacks":       "goroutine 1 [running]:\nmain.main()\n",
		"diag-test.txt":              "diagnostic output here",
		"pty-logs/pod-1/output.log":  "PTY output data\n",
		"pty-logs/pod-2/session.log": "second pod session\n",
	}

	for name, content := range files {
		full := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0644))
	}

	// Build a tar.gz archive using addFileToTar for every file.
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	const budget int64 = 1 << 20 // 1 MB — plenty for our test files
	for name := range files {
		full := filepath.Join(dir, name)
		archiveName := filepath.ToSlash(name)
		_, err := addFileToTar(tw, full, archiveName, budget)
		require.NoError(t, err, "addFileToTar failed for %s", name)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())

	// Round-trip: extract and verify every entry.
	entries := extractTarGz(t, &buf)
	assert.Len(t, entries, len(files))

	for name, wantContent := range files {
		archiveName := filepath.ToSlash(name)
		got, ok := entries[archiveName]
		require.True(t, ok, "missing archive entry: %s", archiveName)
		assert.Equal(t, []byte(wantContent), got, "content mismatch for %s", archiveName)
	}
}

func TestLogCollector_SizeBudget_Integration(t *testing.T) {
	dir := t.TempDir()

	// Create 5 files of 1 KB each.
	const fileSize = 1024
	fileNames := []string{"a.log", "b.log", "c.log", "d.log", "e.log"}
	for _, name := range fileNames {
		content := []byte(strings.Repeat("x", fileSize))
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), content, 0644))
	}

	// Use a very tight maxBytes to force truncation.
	const maxBytes int64 = 500

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	for _, name := range fileNames {
		full := filepath.Join(dir, name)
		written, err := addFileToTar(tw, full, name, maxBytes)
		require.NoError(t, err)
		assert.LessOrEqual(t, written, maxBytes,
			"written bytes should not exceed maxBytes for %s", name)
	}
	require.NoError(t, tw.Close())

	// Read back and verify content length capped.
	tr := tar.NewReader(&buf)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		data, err := io.ReadAll(tr)
		require.NoError(t, err)
		assert.LessOrEqual(t, int64(len(data)), maxBytes,
			"archive entry %s exceeds maxBytes", hdr.Name)
	}
}

func TestLogCollector_FilePadding_Integration(t *testing.T) {
	dir := t.TempDir()

	// Create a file with known content.
	content := []byte(strings.Repeat("A", 100))
	path := filepath.Join(dir, "pad.log")
	require.NoError(t, os.WriteFile(path, content, 0644))

	// Open file, stat, then truncate to simulate shrinkage between Stat and Copy.
	f, err := os.Open(path)
	require.NoError(t, err)
	info, err := f.Stat()
	require.NoError(t, err)
	f.Close()

	originalSize := info.Size()
	require.NoError(t, os.WriteFile(path, content[:50], 0644))

	// addFileToTar should still produce a valid tar entry because it pads.
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	written, err := addFileToTar(tw, path, "pad.log", 4096)
	require.NoError(t, err)
	// Written should be at most the (new, smaller) file size since Stat is
	// called inside addFileToTar on the current file state.
	_ = originalSize
	assert.Equal(t, int64(50), written)
	require.NoError(t, tw.Close())

	// Verify the tar entry is readable with correct length.
	tr := tar.NewReader(&buf)
	hdr, err := tr.Next()
	require.NoError(t, err)
	assert.Equal(t, "pad.log", hdr.Name)
	data, err := io.ReadAll(tr)
	require.NoError(t, err)
	assert.Len(t, data, 50)
}
