package cache

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// maxDownloadSize is the maximum allowed download size (200MB).
const maxDownloadSize = 200 * 1024 * 1024

// maxRedirects is the maximum number of HTTP redirects to follow.
const maxRedirects = 3

// Downloader handles downloading resources and integrating with the cache.
type Downloader struct {
	cache  *SkillCacheManager
	client *http.Client
}

// NewDownloader creates a new Downloader with the given cache manager.
func NewDownloader(cache *SkillCacheManager) *Downloader {
	return &Downloader{
		cache: cache,
		client: &http.Client{
			Timeout: 5 * time.Minute,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= maxRedirects {
					return fmt.Errorf("too many redirects (max %d)", maxRedirects)
				}
				// Only allow HTTP/HTTPS protocols on redirects
				if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
					return fmt.Errorf("redirect to disallowed protocol: %s", req.URL.Scheme)
				}
				return nil
			},
		},
	}
}

// DownloadResult contains the result of a download operation.
type DownloadResult struct {
	SHA       string
	CacheHit  bool
	BytesRead int64
}

// DownloadAndExtract downloads a resource (if not cached) and extracts it to the target path.
// Steps:
// 1. Check cache by SHA — if hit, skip download
// 2. HTTP GET the presigned URL — download to cache
// 3. Extract from cache to target_path
func (d *Downloader) DownloadAndExtract(ctx context.Context, res *runnerv1.ResourceToDownload, sandboxRoot, workDir string) (*DownloadResult, error) {
	if res == nil {
		return nil, fmt.Errorf("resource is nil")
	}
	if res.Sha == "" {
		return nil, fmt.Errorf("resource SHA is required")
	}

	result := &DownloadResult{SHA: res.Sha}

	// 1. Check cache
	if _, ok := d.cache.Get(res.Sha); ok {
		result.CacheHit = true
	} else {
		// 2. Download from presigned URL
		if res.DownloadUrl == "" {
			return nil, fmt.Errorf("download URL is required for SHA %s", res.Sha)
		}

		bytesRead, err := d.download(ctx, res.Sha, res.DownloadUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to download resource (SHA: %s): %w", res.Sha, err)
		}
		result.BytesRead = bytesRead
	}

	// 3. Resolve target path and extract
	targetPath, err := resolveResourcePath(res.TargetPath, sandboxRoot, workDir)
	if err != nil {
		return nil, fmt.Errorf("invalid resource target path: %w", err)
	}
	if err := d.cache.ExtractTo(res.Sha, targetPath); err != nil {
		return nil, fmt.Errorf("failed to extract resource (SHA: %s) to %s: %w", res.Sha, targetPath, err)
	}

	return result, nil
}

// download fetches the resource from the URL and stores it in cache.
func (d *Downloader) download(ctx context.Context, sha, url string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	// Enforce download size limit to prevent resource exhaustion
	limitedBody := io.LimitReader(resp.Body, maxDownloadSize+1)

	// Wrap body in a counting reader
	cr := &countingReader{r: limitedBody}

	// Use Put (not PutAndVerify) because the SHA is a content-level identifier
	// (hash of directory contents), not a hash of the tar.gz package itself.
	// The SHA serves as a cache version key for deduplication.
	if _, err := d.cache.Put(sha, cr); err != nil {
		return 0, fmt.Errorf("failed to cache downloaded data: %w", err)
	}

	if cr.n >= maxDownloadSize {
		// Remove the oversized cached file
		d.cache.mu.Lock()
		os.Remove(d.cache.cachePath(sha))
		d.cache.mu.Unlock()
		return 0, fmt.Errorf("download exceeded maximum size of %d bytes", maxDownloadSize)
	}

	return cr.n, nil
}

// resolveResourcePath resolves template variables in resource target paths.
// Returns an error if the resolved path escapes the sandbox root.
func resolveResourcePath(pathTemplate, sandboxRoot, workDir string) (string, error) {
	path := pathTemplate
	path = strings.ReplaceAll(path, "{{.sandbox.root_path}}", sandboxRoot)
	path = strings.ReplaceAll(path, "{{.sandbox.work_dir}}", workDir)
	path = filepath.Clean(path)

	// Validate the resolved path stays within the sandbox
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}
	absSandbox, err := filepath.Abs(sandboxRoot)
	if err != nil {
		return "", fmt.Errorf("failed to resolve sandbox root: %w", err)
	}
	absSandbox = filepath.Clean(absSandbox)

	if absPath != absSandbox && !strings.HasPrefix(absPath, absSandbox+string(os.PathSeparator)) {
		return "", fmt.Errorf("path %q escapes sandbox root %q", path, sandboxRoot)
	}

	return path, nil
}

// countingReader wraps a reader to count bytes read.
type countingReader struct {
	r io.Reader
	n int64
}

func (cr *countingReader) Read(p []byte) (int, error) {
	n, err := cr.r.Read(p)
	cr.n += int64(n)
	return n, err
}
