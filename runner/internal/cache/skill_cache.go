package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// SkillCacheManager manages local caching of Skill packages by content SHA.
// Cache directory structure: {cacheDir}/{sha}.tar.gz
type SkillCacheManager struct {
	cacheDir string
	mu       sync.RWMutex
}

// NewSkillCacheManager creates a new SkillCacheManager with the specified cache directory.
// The directory is created if it doesn't exist.
func NewSkillCacheManager(cacheDir string) (*SkillCacheManager, error) {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		slog.Error("Failed to create skill cache directory", "path", cacheDir, "error", err)
		return nil, fmt.Errorf("failed to create cache directory %s: %w", cacheDir, err)
	}
	return &SkillCacheManager{cacheDir: cacheDir}, nil
}

// Get checks if a cached package with the given SHA exists.
// Returns the file path and true if found, empty string and false otherwise.
func (m *SkillCacheManager) Get(sha string) (string, bool) {
	if sha == "" || !isValidSHA(sha) {
		return "", false
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	path := m.cachePath(sha)
	if _, err := os.Stat(path); err == nil {
		return path, true
	}
	return "", false
}

// Put stores a tar.gz package in the cache indexed by SHA.
// Writes to a temporary file first (without lock to avoid blocking during network IO),
// then acquires lock only for the atomic rename to prevent partial reads.
func (m *SkillCacheManager) Put(sha string, data io.Reader) (string, error) {
	if sha == "" {
		return "", fmt.Errorf("sha is required")
	}
	if !isValidSHA(sha) {
		return "", fmt.Errorf("invalid SHA format: %s", sha)
	}

	targetPath := m.cachePath(sha)

	// Quick check without write lock — race is harmless (worst case: redundant download)
	m.mu.RLock()
	_, err := os.Stat(targetPath)
	m.mu.RUnlock()
	if err == nil {
		return targetPath, nil
	}

	// Write to temp file first — outside lock to avoid blocking during network IO
	tmpFile, err := os.CreateTemp(m.cacheDir, "download-*.tmp")
	if err != nil {
		slog.Error("Failed to create temp file for cache put", "dir", m.cacheDir, "error", err)
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		// Clean up temp file on failure
		if tmpPath != "" {
			os.Remove(tmpPath)
		}
	}()

	if _, err := io.Copy(tmpFile, data); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("failed to write cache data: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	// Acquire write lock only for the atomic rename
	m.mu.Lock()
	// Double-check: another goroutine may have written this SHA while we were downloading
	if _, err := os.Stat(targetPath); err == nil {
		m.mu.Unlock()
		tmpPath = "" // Don't clean up — let defer handle it (it's idempotent since os.Remove is no-op)
		os.Remove(tmpFile.Name())
		return targetPath, nil
	}
	renameErr := os.Rename(tmpPath, targetPath)
	m.mu.Unlock()

	if renameErr != nil {
		slog.Error("Failed to rename temp file to cache", "sha", sha, "error", renameErr)
		return "", fmt.Errorf("failed to rename temp file to cache: %w", renameErr)
	}
	tmpPath = "" // Prevent cleanup

	return targetPath, nil
}

// PutAndVerify stores data in the cache and verifies the SHA256 of the written content
// matches the expected SHA. If mismatch, the cached file is removed and an error is returned.
func (m *SkillCacheManager) PutAndVerify(expectedSha string, data io.Reader) (string, error) {
	if expectedSha == "" {
		return "", fmt.Errorf("expected SHA is required")
	}

	// Wrap the reader with a tee that also feeds a SHA256 hash
	hasher := sha256.New()
	teeReader := io.TeeReader(data, hasher)

	path, err := m.Put(expectedSha, teeReader)
	if err != nil {
		return "", err
	}

	// If the file was already cached (skip path), we trust the existing cache entry
	// because Put returns early if the file exists. In that case the teeReader
	// was not fully consumed, so we cannot verify. This is acceptable since
	// the file was previously verified or written under this SHA key.
	computedSha := hex.EncodeToString(hasher.Sum(nil))
	if computedSha == "" || computedSha == "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		// Empty hash means nothing was read (cache hit), skip verification
		return path, nil
	}

	if computedSha != expectedSha {
		// SHA mismatch - remove the corrupted cached file
		m.mu.Lock()
		os.Remove(m.cachePath(expectedSha))
		m.mu.Unlock()
		slog.Error("Skill cache SHA mismatch", "expected", expectedSha, "got", computedSha)
		return "", fmt.Errorf("SHA mismatch: expected %s, got %s", expectedSha, computedSha)
	}

	return path, nil
}

// ExtractTo extracts a cached tar.gz package to the target directory.
// The target directory is created if it doesn't exist.
// Returns an error if the SHA is not found in cache.
func (m *SkillCacheManager) ExtractTo(sha string, targetDir string) error {
	m.mu.RLock()
	cachePath := m.cachePath(sha)
	m.mu.RUnlock()

	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		slog.Warn("Skill cache miss during extract", "sha", sha)
		return fmt.Errorf("cache miss for SHA %s", sha)
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	f, err := os.Open(cachePath)
	if err != nil {
		return fmt.Errorf("failed to open cached file: %w", err)
	}
	defer f.Close()

	return extractTarGz(f, targetDir)
}

// CacheDir returns the cache directory path.
func (m *SkillCacheManager) CacheDir() string {
	return m.cacheDir
}

// cachePath returns the file path for a given SHA in the cache.
func (m *SkillCacheManager) cachePath(sha string) string {
	return filepath.Join(m.cacheDir, sha+".tar.gz")
}

// isValidSHA checks if a string is a valid hex-encoded SHA256 (exactly 64 lowercase hex characters).
// This prevents path traversal via crafted SHA strings (e.g. "../../etc/passwd").
func isValidSHA(sha string) bool {
	if len(sha) != 64 {
		return false
	}
	for _, c := range sha {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			continue
		}
		return false
	}
	return true
}
