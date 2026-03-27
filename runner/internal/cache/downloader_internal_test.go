package cache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// download (lower-level)
// ---------------------------------------------------------------------------

func TestDownloader_download(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cacheDir := t.TempDir()
		mgr, err := NewSkillCacheManager(cacheDir)
		require.NoError(t, err)

		tarGzData := createTestTarGz(t, map[string]string{
			"file.txt": "content",
		})

		// Compute real SHA256
		h := sha256.New()
		h.Write(tarGzData)
		sha := hex.EncodeToString(h.Sum(nil))

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write(tarGzData)
		}))
		defer srv.Close()

		dl := NewDownloader(mgr)
		bytesRead, err := dl.download(context.Background(), sha, srv.URL+"/pkg.tar.gz")
		require.NoError(t, err)
		assert.Equal(t, int64(len(tarGzData)), bytesRead)

		// Verify it was stored in cache
		_, ok := mgr.Get(sha)
		assert.True(t, ok)
	})

	t.Run("non_200", func(t *testing.T) {
		cacheDir := t.TempDir()
		mgr, err := NewSkillCacheManager(cacheDir)
		require.NoError(t, err)

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		dl := NewDownloader(mgr)
		_, err = dl.download(context.Background(), "aabbccddee00112233445566778899aabbccddee00112233445566778899aabb", srv.URL+"/missing")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected HTTP status: 404")
	})

	t.Run("invalid_url_request_creation_error", func(t *testing.T) {
		cacheDir := t.TempDir()
		mgr, err := NewSkillCacheManager(cacheDir)
		require.NoError(t, err)

		dl := NewDownloader(mgr)
		// A URL containing a control character makes http.NewRequestWithContext fail
		_, err = dl.download(context.Background(), "aa11bb22cc33dd44ee55ff6600112233aa11bb22cc33dd44ee55ff6600112233", "http://invalid\x7f.example.com/path")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create HTTP request")
	})
}

// ---------------------------------------------------------------------------
// Additional coverage tests
// ---------------------------------------------------------------------------

func TestNewDownloader(t *testing.T) {
	cacheDir := t.TempDir()
	mgr, err := NewSkillCacheManager(cacheDir)
	require.NoError(t, err)

	dl := NewDownloader(mgr)
	assert.NotNil(t, dl)
	assert.NotNil(t, dl.client)
	assert.Equal(t, mgr, dl.cache)
}

func TestDownloader_DownloadAndExtract_ExtractError(t *testing.T) {
	cacheDir := t.TempDir()
	mgr, err := NewSkillCacheManager(cacheDir)
	require.NoError(t, err)

	sha := "1111222233334444555566667777888899990000aaaabbbbccccddddeeeeffff"

	// Put invalid data (not a valid tar.gz) in the cache so extraction fails
	_, err = mgr.Put(sha, bytes.NewReader([]byte("this is not a tar.gz file")))
	require.NoError(t, err)

	sandboxRoot := t.TempDir()
	dl := NewDownloader(mgr)
	res := &runnerv1.ResourceToDownload{
		Sha:        sha,
		TargetPath: "{{.sandbox.root_path}}/output",
	}

	_, err = dl.DownloadAndExtract(context.Background(), res, sandboxRoot, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to extract resource")
}

func TestDownloader_download_ClientDoError(t *testing.T) {
	cacheDir := t.TempDir()
	mgr, err := NewSkillCacheManager(cacheDir)
	require.NoError(t, err)

	dl := NewDownloader(mgr)

	// Use a cancelled context so that client.Do fails
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = dl.download(ctx, "2222333344445555666677778888999900001111aaaabbbbccccddddeeeeffff", "http://127.0.0.1:1/unreachable")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP request failed")
}

func TestDownloader_download_CachesWithAnySHA(t *testing.T) {
	cacheDir := t.TempDir()
	mgr, err := NewSkillCacheManager(cacheDir)
	require.NoError(t, err)

	// SHA is used as a cache key only (not for content verification).
	// The content_sha from backend is a hash of directory contents,
	// not the tar.gz package hash.
	tarGzData := createTestTarGz(t, map[string]string{"x.txt": "x"})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(tarGzData)
	}))
	defer srv.Close()

	dl := NewDownloader(mgr)
	_, err = dl.download(context.Background(), "3333444455556666777788889999000011112222aaaabbbbccccddddeeeeffff", srv.URL+"/pkg.tar.gz")
	require.NoError(t, err)

	// Verify cached with the SHA key
	_, ok := mgr.Get("3333444455556666777788889999000011112222aaaabbbbccccddddeeeeffff")
	assert.True(t, ok)
}

func TestDownloader_DownloadAndExtract_DownloadFails(t *testing.T) {
	cacheDir := t.TempDir()
	mgr, err := NewSkillCacheManager(cacheDir)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	dl := NewDownloader(mgr)
	res := &runnerv1.ResourceToDownload{
		Sha:         "4444555566667777888899990000111122223333aaaabbbbccccddddeeeeffff",
		DownloadUrl: srv.URL + "/forbidden",
		TargetPath:  t.TempDir(),
	}

	_, err = dl.DownloadAndExtract(context.Background(), res, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to download resource")
}

func TestCountingReader(t *testing.T) {
	data := []byte("hello, counting reader!")
	cr := &countingReader{r: bytes.NewReader(data)}

	buf := make([]byte, 5)
	n, err := cr.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, int64(5), cr.n)

	n, err = cr.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, int64(10), cr.n)
}
