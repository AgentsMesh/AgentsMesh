package cache

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// PutAndVerify
// ---------------------------------------------------------------------------

func TestSkillCacheManager_PutAndVerify(t *testing.T) {
	t.Run("empty_sha", func(t *testing.T) {
		cacheDir := t.TempDir()
		mgr, err := NewSkillCacheManager(cacheDir)
		require.NoError(t, err)

		_, err = mgr.PutAndVerify("", bytes.NewReader([]byte("data")))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected SHA is required")
	})

	t.Run("sha_matches", func(t *testing.T) {
		cacheDir := t.TempDir()
		mgr, err := NewSkillCacheManager(cacheDir)
		require.NoError(t, err)

		data := []byte("test content for sha verification")
		h := sha256.New()
		h.Write(data)
		expectedSha := hex.EncodeToString(h.Sum(nil))

		path, err := mgr.PutAndVerify(expectedSha, bytes.NewReader(data))
		require.NoError(t, err)
		assert.NotEmpty(t, path)

		// Verify the file exists in cache
		cachedPath, ok := mgr.Get(expectedSha)
		assert.True(t, ok)
		assert.Equal(t, path, cachedPath)

		// Verify content
		content, err := os.ReadFile(cachedPath)
		require.NoError(t, err)
		assert.Equal(t, data, content)
	})

	t.Run("sha_mismatch", func(t *testing.T) {
		cacheDir := t.TempDir()
		mgr, err := NewSkillCacheManager(cacheDir)
		require.NoError(t, err)

		data := []byte("this data does not match the expected sha")
		wrongSha := "0000000000000000000000000000000000000000000000000000000000000000"

		_, err = mgr.PutAndVerify(wrongSha, bytes.NewReader(data))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SHA mismatch")

		// Verify file was removed from cache
		_, ok := mgr.Get(wrongSha)
		assert.False(t, ok)
	})

	t.Run("cache_hit_skips_verification", func(t *testing.T) {
		cacheDir := t.TempDir()
		mgr, err := NewSkillCacheManager(cacheDir)
		require.NoError(t, err)

		data := []byte("pre-existing cached content")
		h := sha256.New()
		h.Write(data)
		sha := hex.EncodeToString(h.Sum(nil))

		// Pre-populate the cache via Put
		_, err = mgr.Put(sha, bytes.NewReader(data))
		require.NoError(t, err)

		// PutAndVerify with different reader data should still succeed because
		// the file already exists and Put returns early (teeReader not consumed).
		differentData := []byte("completely different data that would fail SHA check")
		path, err := mgr.PutAndVerify(sha, bytes.NewReader(differentData))
		require.NoError(t, err)
		assert.NotEmpty(t, path)

		// The original content should be preserved
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, data, content)
	})
}
