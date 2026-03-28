package extension

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/internal/infra/storage"
)

// SkillPackager handles packaging skills from GitHub URLs and file uploads
type SkillPackager struct {
	repo    extension.Repository
	storage storage.Storage

	// gitCloneFn can be overridden in tests to avoid real git operations.
	gitCloneFn func(ctx context.Context, url, branch, targetDir string) error
}

// NewSkillPackager creates a new SkillPackager
func NewSkillPackager(repo extension.Repository, storage storage.Storage) *SkillPackager {
	return &SkillPackager{
		repo:    repo,
		storage: storage,
	}
}

// PackageFromGitHub clones a GitHub repo (optionally specific path) and packages the skill
func (p *SkillPackager) PackageFromGitHub(ctx context.Context, url, branch, path string) (*PackagedSkill, error) {
	// Clone to temp dir
	tmpDir, err := os.MkdirTemp("", "skill-github-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	repoDir := filepath.Join(tmpDir, "repo")
	cloneFn := gitClone
	if p.gitCloneFn != nil {
		cloneFn = p.gitCloneFn
	}
	if err := cloneFn(ctx, url, branch, repoDir); err != nil {
		return nil, fmt.Errorf("failed to clone: %w", err)
	}

	// Determine skill directory
	skillDir := repoDir
	if path != "" {
		skillDir = filepath.Join(repoDir, filepath.Clean(path))
		// Prevent path traversal outside the cloned repository
		if !strings.HasPrefix(skillDir, filepath.Clean(repoDir)+string(os.PathSeparator)) {
			return nil, fmt.Errorf("invalid path: escapes repository directory")
		}
		if !dirExists(skillDir) {
			return nil, fmt.Errorf("path %q not found in repository", path)
		}
	}

	// Verify SKILL.md exists
	if !fileExists(filepath.Join(skillDir, "SKILL.md")) {
		return nil, fmt.Errorf("SKILL.md not found in %s", path)
	}

	return p.packageDir(ctx, skillDir)
}

// PackageFromUpload processes an uploaded tar.gz/zip file
func (p *SkillPackager) PackageFromUpload(ctx context.Context, reader io.Reader, filename string) (*PackagedSkill, error) {
	// Save to temp file
	tmpDir, err := os.MkdirTemp("", "skill-upload-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Read upload content with size limit
	const maxUploadSize = 50 * 1024 * 1024 // 50MB
	limitedReader := io.LimitReader(reader, maxUploadSize+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read upload: %w", err)
	}
	if int64(len(data)) > maxUploadSize {
		return nil, fmt.Errorf("upload exceeds maximum size of %d bytes", maxUploadSize)
	}

	// Extract based on file type
	extractDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create extract dir: %w", err)
	}

	if strings.HasSuffix(filename, ".tar.gz") || strings.HasSuffix(filename, ".tgz") {
		if err := extractTarGz(bytes.NewReader(data), extractDir); err != nil {
			return nil, fmt.Errorf("failed to extract tar.gz: %w", err)
		}
	} else {
		return nil, fmt.Errorf("unsupported file format: %s (only .tar.gz supported)", filename)
	}

	// Find SKILL.md in extracted content
	skillDir, err := findSkillDir(extractDir)
	if err != nil {
		return nil, err
	}

	return p.packageDir(ctx, skillDir)
}

// packageDir packages a skill directory and uploads to storage
func (p *SkillPackager) packageDir(ctx context.Context, dirPath string) (*PackagedSkill, error) {
	// Parse SKILL.md
	info, err := parseSkillDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse skill: %w", err)
	}

	// Compute SHA
	sha, err := computeDirSHA(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to compute SHA: %w", err)
	}

	// Package
	packageData, err := packageSkillDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to package: %w", err)
	}

	// Upload to storage
	storageKey := fmt.Sprintf("skills/direct/%s/%s.tar.gz", info.Slug, sha)
	_, err = p.storage.Upload(ctx, storageKey, bytes.NewReader(packageData), int64(len(packageData)), "application/gzip")
	if err != nil {
		slog.Error("failed to upload skill package", "slug", info.Slug, "storage_key", storageKey, "error", err)
		return nil, fmt.Errorf("failed to upload: %w", err)
	}

	slog.Info("skill packaged and uploaded", "slug", info.Slug, "content_sha", sha, "package_size", len(packageData))

	return &PackagedSkill{
		Slug:        info.Slug,
		DisplayName: info.DisplayName,
		Description: info.Description,
		ContentSha:  sha,
		StorageKey:  storageKey,
		PackageSize: int64(len(packageData)),
	}, nil
}

// PackagedSkill represents the result of packaging a skill
type PackagedSkill struct {
	Slug        string
	DisplayName string
	Description string
	ContentSha  string
	StorageKey  string
	PackageSize int64
}
