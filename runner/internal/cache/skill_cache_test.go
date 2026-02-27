package cache

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillCacheManager_PutAndGet(t *testing.T) {
	cacheDir := t.TempDir()
	mgr, err := NewSkillCacheManager(cacheDir)
	require.NoError(t, err)

	sha := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	data := []byte("test package content")

	// Initially not cached
	_, ok := mgr.Get(sha)
	assert.False(t, ok)

	// Put into cache
	path, err := mgr.Put(sha, bytes.NewReader(data))
	require.NoError(t, err)
	assert.Contains(t, path, sha)

	// Now should be cached
	cachedPath, ok := mgr.Get(sha)
	assert.True(t, ok)
	assert.Equal(t, path, cachedPath)

	// Verify content
	content, err := os.ReadFile(cachedPath)
	require.NoError(t, err)
	assert.Equal(t, data, content)
}

func TestSkillCacheManager_PutIdempotent(t *testing.T) {
	cacheDir := t.TempDir()
	mgr, err := NewSkillCacheManager(cacheDir)
	require.NoError(t, err)

	sha := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	data1 := []byte("first write")
	data2 := []byte("second write -- should be ignored")

	// First put
	_, err = mgr.Put(sha, bytes.NewReader(data1))
	require.NoError(t, err)

	// Second put should be a no-op (already cached)
	_, err = mgr.Put(sha, bytes.NewReader(data2))
	require.NoError(t, err)

	// Content should still be from first write
	cachedPath, _ := mgr.Get(sha)
	content, err := os.ReadFile(cachedPath)
	require.NoError(t, err)
	assert.Equal(t, data1, content)
}

func TestSkillCacheManager_ExtractTo(t *testing.T) {
	cacheDir := t.TempDir()
	mgr, err := NewSkillCacheManager(cacheDir)
	require.NoError(t, err)

	sha := "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"

	// Create a tar.gz with test files
	tarGzData := createTestTarGz(t, map[string]string{
		"SKILL.md":              "# My Skill\nDescription here",
		"scripts/run.sh":        "#!/bin/bash\necho hello",
		"references/REFERENCE.md": "# Reference",
	})

	_, err = mgr.Put(sha, bytes.NewReader(tarGzData))
	require.NoError(t, err)

	// Extract to target directory
	targetDir := filepath.Join(t.TempDir(), "skill-output")
	err = mgr.ExtractTo(sha, targetDir)
	require.NoError(t, err)

	// Verify extracted files
	content, err := os.ReadFile(filepath.Join(targetDir, "SKILL.md"))
	require.NoError(t, err)
	assert.Equal(t, "# My Skill\nDescription here", string(content))

	content, err = os.ReadFile(filepath.Join(targetDir, "scripts/run.sh"))
	require.NoError(t, err)
	assert.Equal(t, "#!/bin/bash\necho hello", string(content))

	content, err = os.ReadFile(filepath.Join(targetDir, "references/REFERENCE.md"))
	require.NoError(t, err)
	assert.Equal(t, "# Reference", string(content))
}

func TestSkillCacheManager_ExtractTo_CacheMiss(t *testing.T) {
	cacheDir := t.TempDir()
	mgr, err := NewSkillCacheManager(cacheDir)
	require.NoError(t, err)

	err = mgr.ExtractTo("aabbccddee00112233445566778899aabbccddee00112233445566778899aabb", t.TempDir())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache miss")
}

func TestSkillCacheManager_GetEmpty(t *testing.T) {
	cacheDir := t.TempDir()
	mgr, err := NewSkillCacheManager(cacheDir)
	require.NoError(t, err)

	_, ok := mgr.Get("")
	assert.False(t, ok)
}

func TestSkillCacheManager_PutEmptySha(t *testing.T) {
	cacheDir := t.TempDir()
	mgr, err := NewSkillCacheManager(cacheDir)
	require.NoError(t, err)

	_, err = mgr.Put("", bytes.NewReader([]byte("data")))
	assert.Error(t, err)
}

// createTestTarGz creates a tar.gz archive from a map of filename → content.
func createTestTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		err := tw.WriteHeader(hdr)
		require.NoError(t, err)
		_, err = tw.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())

	return buf.Bytes()
}

// ---------------------------------------------------------------------------
// Additional coverage tests
// ---------------------------------------------------------------------------

func TestSkillCacheManager_CacheDir(t *testing.T) {
	cacheDir := t.TempDir()
	mgr, err := NewSkillCacheManager(cacheDir)
	require.NoError(t, err)

	assert.Equal(t, cacheDir, mgr.CacheDir())
}

func TestNewSkillCacheManager_InvalidPath(t *testing.T) {
	// /dev/null is a regular file, not a directory — creating a subdirectory under it should fail
	invalidPath := "/dev/null/impossible/cache/dir"
	mgr, err := NewSkillCacheManager(invalidPath)
	assert.Nil(t, mgr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create cache directory")
}

func TestSkillCacheManager_Put_WriteFail(t *testing.T) {
	cacheDir := t.TempDir()
	mgr, err := NewSkillCacheManager(cacheDir)
	require.NoError(t, err)

	// errReader always returns an error on Read
	r := &errReader{err: os.ErrClosed}
	_, err = mgr.Put("aa11bb22cc33dd44ee55ff6600112233aa11bb22cc33dd44ee55ff6600112233", r)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write cache data")
}

// errReader is a reader that always returns an error.
type errReader struct {
	err error
}

func (r *errReader) Read(p []byte) (int, error) {
	return 0, r.err
}

func TestSkillCacheManager_ExtractTo_WithDirectoryEntries(t *testing.T) {
	cacheDir := t.TempDir()
	mgr, err := NewSkillCacheManager(cacheDir)
	require.NoError(t, err)

	sha := "1111222233334444555566667777888899990000aaaabbbbccccddddeeeeffff"

	// Create a tar.gz that explicitly includes TypeDir entries
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Add a directory entry
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     "mydir/",
		Typeflag: tar.TypeDir,
		Mode:     0755,
	}))

	// Add a file inside the directory
	content := "file inside dir"
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     "mydir/file.txt",
		Typeflag: tar.TypeReg,
		Mode:     0644,
		Size:     int64(len(content)),
	}))
	_, err = tw.Write([]byte(content))
	require.NoError(t, err)

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())

	_, err = mgr.Put(sha, bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)

	targetDir := filepath.Join(t.TempDir(), "extracted-dirs")
	err = mgr.ExtractTo(sha, targetDir)
	require.NoError(t, err)

	// Verify directory was created
	info, err := os.Stat(filepath.Join(targetDir, "mydir"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Verify file inside directory
	data, err := os.ReadFile(filepath.Join(targetDir, "mydir", "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestSkillCacheManager_Put_CreateTempFail(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}

	cacheDir := t.TempDir()
	mgr, err := NewSkillCacheManager(cacheDir)
	require.NoError(t, err)

	// Make cache directory read-only so CreateTemp fails
	require.NoError(t, os.Chmod(cacheDir, 0555))
	defer os.Chmod(cacheDir, 0755) // restore for cleanup

	_, err = mgr.Put("2222333344445555666677778888999900001111aaaabbbbccccddddeeeeffff", bytes.NewReader([]byte("some data")))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create temp file")
}

func TestSkillCacheManager_ExtractTo_TargetDirCreationFail(t *testing.T) {
	cacheDir := t.TempDir()
	mgr, err := NewSkillCacheManager(cacheDir)
	require.NoError(t, err)

	sha := "3333444455556666777788889999000011112222aaaabbbbccccddddeeeeffff"
	_, err = mgr.Put(sha, bytes.NewReader(createTestTarGz(t, map[string]string{"a.txt": "a"})))
	require.NoError(t, err)

	// Use a path under /dev/null which can't be created
	err = mgr.ExtractTo(sha, "/dev/null/impossible/target")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create target directory")
}

func TestSkillCacheManager_ExtractTo_OpenCachedFileFail(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}

	cacheDir := t.TempDir()
	mgr, err := NewSkillCacheManager(cacheDir)
	require.NoError(t, err)

	sha := "4444555566667777888899990000111122223333aaaabbbbccccddddeeeeffff"
	// Put data in cache
	_, err = mgr.Put(sha, bytes.NewReader([]byte("some data")))
	require.NoError(t, err)

	// Make the cached file unreadable
	cachePath := filepath.Join(cacheDir, sha+".tar.gz")
	require.NoError(t, os.Chmod(cachePath, 0000))
	defer os.Chmod(cachePath, 0644)

	err = mgr.ExtractTo(sha, t.TempDir())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open cached file")
}

func TestSkillCacheManager_ExtractTo_InvalidGzip(t *testing.T) {
	cacheDir := t.TempDir()
	mgr, err := NewSkillCacheManager(cacheDir)
	require.NoError(t, err)

	sha := "5555666677778888999900001111222233334444aaaabbbbccccddddeeeeffff"
	// Put non-gzip data in cache
	_, err = mgr.Put(sha, bytes.NewReader([]byte("this is not gzip")))
	require.NoError(t, err)

	err = mgr.ExtractTo(sha, filepath.Join(t.TempDir(), "out"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create gzip reader")
}

func TestSkillCacheManager_PutAndVerify_PutFails(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}

	cacheDir := t.TempDir()
	mgr, err := NewSkillCacheManager(cacheDir)
	require.NoError(t, err)

	// Make cache directory read-only so Put's CreateTemp fails
	require.NoError(t, os.Chmod(cacheDir, 0555))
	defer os.Chmod(cacheDir, 0755)

	_, err = mgr.PutAndVerify("6666777788889999000011112222333344445555aaaabbbbccccddddeeeeffff", bytes.NewReader([]byte("data")))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create temp file")
}

func TestSkillCacheManager_ExtractTo_ZeroModeDefaults644(t *testing.T) {
	cacheDir := t.TempDir()
	mgr, err := NewSkillCacheManager(cacheDir)
	require.NoError(t, err)

	sha := "7777888899990000111122223333444455556666aaaabbbbccccddddeeeeffff"

	// Create a tar.gz with a file that has mode 0
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	content := "file with zero mode"
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     "zero-mode.txt",
		Typeflag: tar.TypeReg,
		Mode:     0, // zero mode — should default to 0644
		Size:     int64(len(content)),
	}))
	_, err = tw.Write([]byte(content))
	require.NoError(t, err)

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())

	_, err = mgr.Put(sha, bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)

	targetDir := filepath.Join(t.TempDir(), "extracted-zero-mode")
	err = mgr.ExtractTo(sha, targetDir)
	require.NoError(t, err)

	// Verify file was extracted with correct content
	data, err := os.ReadFile(filepath.Join(targetDir, "zero-mode.txt"))
	require.NoError(t, err)
	assert.Equal(t, content, string(data))

	// Verify file permissions default to 0644
	info, err := os.Stat(filepath.Join(targetDir, "zero-mode.txt"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
}
