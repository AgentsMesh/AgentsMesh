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
			want:        filepath.FromSlash("/home/user/sandbox/skills/my-skill"),
		},
		{
			name:        "replaces_sandbox_work_dir",
			pathTmpl:    "{{.sandbox.work_dir}}/.agents/cache",
			sandboxRoot: "/home/user/sandbox",
			workDir:     "/home/user/sandbox/work",
			want:        filepath.FromSlash("/home/user/sandbox/work/.agents/cache"),
		},
		{
			name:        "replaces_both_templates",
			pathTmpl:    "{{.sandbox.root_path}}/data/sub",
			sandboxRoot: "/root",
			workDir:     "/root/work",
			want:        filepath.FromSlash("/root/data/sub"),
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
