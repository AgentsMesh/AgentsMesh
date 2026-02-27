package cache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// resolveResourcePath
// ---------------------------------------------------------------------------

func TestResolveResourcePath(t *testing.T) {
	tests := []struct {
		name        string
		pathTmpl    string
		sandboxRoot string
		workDir     string
		want        string
		wantErr     bool
	}{
		{
			name:        "replaces_sandbox_root_path",
			pathTmpl:    "{{.sandbox.root_path}}/skills/my-skill",
			sandboxRoot: "/home/user/sandbox",
			workDir:     "/home/user/sandbox/work",
			want:        "/home/user/sandbox/skills/my-skill",
		},
		{
			name:        "replaces_sandbox_work_dir",
			pathTmpl:    "{{.sandbox.work_dir}}/.agents/cache",
			sandboxRoot: "/home/user/sandbox",
			workDir:     "/home/user/sandbox/work",
			want:        "/home/user/sandbox/work/.agents/cache",
		},
		{
			name:        "replaces_both_templates",
			pathTmpl:    "{{.sandbox.root_path}}/data/sub",
			sandboxRoot: "/root",
			workDir:     "/root/work",
			want:        "/root/data/sub",
		},
		{
			name:        "path_traversal_rejected",
			pathTmpl:    "{{.sandbox.root_path}}/../../etc/passwd",
			sandboxRoot: "/home/user/sandbox",
			workDir:     "/home/user/sandbox/work",
			wantErr:     true,
		},
		{
			name:        "path_outside_sandbox_rejected",
			pathTmpl:    "/absolute/path/to/resource",
			sandboxRoot: "/sandbox",
			workDir:     "/sandbox/work",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveResourcePath(tt.pathTmpl, tt.sandboxRoot, tt.workDir)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DownloadAndExtract
// ---------------------------------------------------------------------------

func TestDownloader_DownloadAndExtract(t *testing.T) {
	t.Run("nil_resource", func(t *testing.T) {
		cacheDir := t.TempDir()
		mgr, err := NewSkillCacheManager(cacheDir)
		require.NoError(t, err)

		dl := NewDownloader(mgr)
		_, err = dl.DownloadAndExtract(context.Background(), nil, "/sandbox", "/work")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resource is nil")
	})

	t.Run("empty_sha", func(t *testing.T) {
		cacheDir := t.TempDir()
		mgr, err := NewSkillCacheManager(cacheDir)
		require.NoError(t, err)

		dl := NewDownloader(mgr)
		res := &runnerv1.ResourceToDownload{Sha: ""}
		_, err = dl.DownloadAndExtract(context.Background(), res, "/sandbox", "/work")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resource SHA is required")
	})

	t.Run("cache_hit", func(t *testing.T) {
		cacheDir := t.TempDir()
		mgr, err := NewSkillCacheManager(cacheDir)
		require.NoError(t, err)

		sha := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
		tarGzData := createTestTarGz(t, map[string]string{
			"hello.txt": "world",
		})

		// Pre-populate the cache
		_, err = mgr.Put(sha, bytes.NewReader(tarGzData))
		require.NoError(t, err)

		sandboxRoot := t.TempDir()
		targetDir := filepath.Join(sandboxRoot, "output")
		dl := NewDownloader(mgr)
		res := &runnerv1.ResourceToDownload{
			Sha:        sha,
			TargetPath: "{{.sandbox.root_path}}/output",
		}

		result, err := dl.DownloadAndExtract(context.Background(), res, sandboxRoot, "")
		require.NoError(t, err)
		assert.True(t, result.CacheHit)
		assert.Equal(t, int64(0), result.BytesRead)
		assert.Equal(t, sha, result.SHA)

		// Verify extraction still happened
		content, err := os.ReadFile(filepath.Join(targetDir, "hello.txt"))
		require.NoError(t, err)
		assert.Equal(t, "world", string(content))
	})

	t.Run("cache_miss_no_url", func(t *testing.T) {
		cacheDir := t.TempDir()
		mgr, err := NewSkillCacheManager(cacheDir)
		require.NoError(t, err)

		dl := NewDownloader(mgr)
		res := &runnerv1.ResourceToDownload{
			Sha:         "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			DownloadUrl: "",
			TargetPath:  "/some/path",
		}
		_, err = dl.DownloadAndExtract(context.Background(), res, "", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "download URL is required")
	})

	t.Run("cache_miss_success", func(t *testing.T) {
		cacheDir := t.TempDir()
		mgr, err := NewSkillCacheManager(cacheDir)
		require.NoError(t, err)

		tarGzData := createTestTarGz(t, map[string]string{
			"README.md":  "# Skill",
			"config.yml": "name: test",
		})

		// Compute real SHA256 of the tar.gz data
		h := sha256.New()
		h.Write(tarGzData)
		sha := hex.EncodeToString(h.Sum(nil))

		// Serve the tar.gz from a test HTTP server
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/gzip")
			w.WriteHeader(http.StatusOK)
			w.Write(tarGzData)
		}))
		defer srv.Close()

		targetDir := filepath.Join(t.TempDir(), "extracted")
		dl := NewDownloader(mgr)
		res := &runnerv1.ResourceToDownload{
			Sha:         sha,
			DownloadUrl: srv.URL + "/package.tar.gz",
			TargetPath:  "{{.sandbox.root_path}}/skills",
		}

		result, err := dl.DownloadAndExtract(context.Background(), res, targetDir, "")
		require.NoError(t, err)

		assert.False(t, result.CacheHit)
		assert.Equal(t, sha, result.SHA)
		assert.Greater(t, result.BytesRead, int64(0))

		// Verify extracted files
		content, err := os.ReadFile(filepath.Join(targetDir, "skills", "README.md"))
		require.NoError(t, err)
		assert.Equal(t, "# Skill", string(content))

		content, err = os.ReadFile(filepath.Join(targetDir, "skills", "config.yml"))
		require.NoError(t, err)
		assert.Equal(t, "name: test", string(content))
	})

	t.Run("download_http_error", func(t *testing.T) {
		cacheDir := t.TempDir()
		mgr, err := NewSkillCacheManager(cacheDir)
		require.NoError(t, err)

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		dl := NewDownloader(mgr)
		res := &runnerv1.ResourceToDownload{
			Sha:         "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
			DownloadUrl: srv.URL + "/fail",
			TargetPath:  t.TempDir(),
		}

		_, err = dl.DownloadAndExtract(context.Background(), res, "", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})
}

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
