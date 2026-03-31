package runner

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helperReadTar extracts the first file from a tar written to buf.
// Returns the archive name and file content.
func helperReadTar(t *testing.T, buf *bytes.Buffer) (string, []byte) {
	t.Helper()
	tr := tar.NewReader(buf)
	hdr, err := tr.Next()
	require.NoError(t, err)
	data, err := io.ReadAll(tr)
	require.NoError(t, err)
	return hdr.Name, data
}

func TestAddFileToTar_Normal(t *testing.T) {
	content := []byte("hello, log collector")
	tmp := filepath.Join(t.TempDir(), "test.log")
	require.NoError(t, os.WriteFile(tmp, content, 0644))

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	written, err := addFileToTar(tw, tmp, "logs/test.log", 1024)
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), written)
	require.NoError(t, tw.Close())

	name, data := helperReadTar(t, &buf)
	assert.Equal(t, "logs/test.log", name)
	assert.Equal(t, content, data)
}

func TestAddFileToTar_LargeFile_Truncated(t *testing.T) {
	content := []byte(strings.Repeat("x", 500))
	tmp := filepath.Join(t.TempDir(), "big.log")
	require.NoError(t, os.WriteFile(tmp, content, 0644))

	var maxBytes int64 = 100
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	written, err := addFileToTar(tw, tmp, "big.log", maxBytes)
	require.NoError(t, err)
	assert.Equal(t, maxBytes, written)
	require.NoError(t, tw.Close())

	name, data := helperReadTar(t, &buf)
	assert.Equal(t, "big.log", name)
	assert.Len(t, data, int(maxBytes))
	assert.Equal(t, content[:maxBytes], data)
}

func TestAddFileToTar_NonExistent(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	_, err := addFileToTar(tw, "/nonexistent/path/file.log", "missing.log", 1024)
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestAddFileToTar_EmptyFile(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "empty.log")
	require.NoError(t, os.WriteFile(tmp, nil, 0644))

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	written, err := addFileToTar(tw, tmp, "empty.log", 1024)
	require.NoError(t, err)
	assert.Equal(t, int64(0), written)
	require.NoError(t, tw.Close())

	tr := tar.NewReader(&buf)
	hdr, err := tr.Next()
	require.NoError(t, err)
	assert.Equal(t, "empty.log", hdr.Name)
	assert.Equal(t, int64(0), hdr.Size)
}

func TestAddFileToTar_ZeroMaxBytes(t *testing.T) {
	content := []byte("should be ignored")
	tmp := filepath.Join(t.TempDir(), "zeroed.log")
	require.NoError(t, os.WriteFile(tmp, content, 0644))

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	written, err := addFileToTar(tw, tmp, "zeroed.log", 0)
	require.NoError(t, err)
	assert.Equal(t, int64(0), written)
	require.NoError(t, tw.Close())

	tr := tar.NewReader(&buf)
	hdr, err := tr.Next()
	require.NoError(t, err)
	assert.Equal(t, int64(0), hdr.Size)
}

func TestCollectLogs_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	tarPath, _, err := CollectLogs(ctx)
	// If the log dir has files, context cancellation returns error.
	// If the log dir is empty/missing, the function succeeds with empty archive.
	// Either outcome is valid — we just verify no panic and consistent state.
	if err != nil {
		assert.ErrorIs(t, err, context.Canceled)
		assert.Empty(t, tarPath)
	} else if tarPath != "" {
		// Clean up the temp archive
		os.Remove(tarPath)
	}
}

// TestAddFileToTar_RoundTrip_Gzip verifies the full gzip→tar pipeline works.
func TestAddFileToTar_RoundTrip_Gzip(t *testing.T) {
	content := []byte("compressed log line\n")
	tmp := filepath.Join(t.TempDir(), "rt.log")
	require.NoError(t, os.WriteFile(tmp, content, 0644))

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	written, err := addFileToTar(tw, tmp, "rt.log", 4096)
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), written)
	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())

	// Decompress and read back
	gr, err := gzip.NewReader(&buf)
	require.NoError(t, err)
	defer gr.Close()

	name, data := helperReadTar(t, func() *bytes.Buffer {
		b, readErr := io.ReadAll(gr)
		require.NoError(t, readErr)
		return bytes.NewBuffer(b)
	}())
	assert.Equal(t, "rt.log", name)
	assert.Equal(t, content, data)
}
