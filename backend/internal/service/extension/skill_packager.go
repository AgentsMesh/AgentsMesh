package extension

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
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
		return nil, fmt.Errorf("failed to upload: %w", err)
	}

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

// findSkillDir finds the directory containing SKILL.md in extracted content
func findSkillDir(extractDir string) (string, error) {
	// Check root
	if fileExists(filepath.Join(extractDir, "SKILL.md")) {
		return extractDir, nil
	}

	// Check one level deep
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return "", fmt.Errorf("failed to read extracted dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subDir := filepath.Join(extractDir, entry.Name())
			if fileExists(filepath.Join(subDir, "SKILL.md")) {
				return subDir, nil
			}
		}
	}

	return "", fmt.Errorf("SKILL.md not found in uploaded archive")
}

// maxTotalExtractSize is the maximum total decompressed size allowed for tar.gz extraction (zip bomb protection).
const maxTotalExtractSize = 200 * 1024 * 1024 // 200MB

// extractTarGz extracts a tar.gz archive to the target directory
func extractTarGz(reader io.Reader, targetDir string) error {
	gz, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()

	var totalSize int64

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		// Accumulate total decompressed size and enforce limit
		if header.Size > 0 {
			totalSize += header.Size
			if totalSize > maxTotalExtractSize {
				return fmt.Errorf("archive exceeds maximum total decompressed size of %d bytes", maxTotalExtractSize)
			}
		}

		targetPath := filepath.Join(targetDir, filepath.Clean(header.Name))
		// Prevent directory traversal
		if !strings.HasPrefix(targetPath, filepath.Clean(targetDir)+string(os.PathSeparator)) &&
			targetPath != filepath.Clean(targetDir) {
			slog.Warn("Skipping archive entry with path traversal", "entry", header.Name)
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// Restrict directory permissions to prevent world-writable dirs from tar
			dirMode := os.FileMode(header.Mode) & 0755
			if dirMode == 0 {
				dirMode = 0755
			}
			if err := os.MkdirAll(targetPath, dirMode); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", targetPath, err)
			}
			// Restrict file permissions: strip execute bits, cap at 0644
			mode := os.FileMode(header.Mode) & 0644
			if mode == 0 {
				mode = 0644
			}
			f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(f, io.LimitReader(tr, 50*1024*1024))
			closeErr := f.Close()
			if copyErr != nil {
				return fmt.Errorf("failed to extract file %s: %w", targetPath, copyErr)
			}
			if closeErr != nil {
				return fmt.Errorf("failed to close file %s: %w", targetPath, closeErr)
			}
		case tar.TypeSymlink, tar.TypeLink:
			slog.Warn("Skipping symlink/hardlink in archive to prevent symlink attacks", "entry", header.Name, "type", header.Typeflag)
			continue
		default:
			slog.Debug("Skipping unsupported tar entry type", "entry", header.Name, "type", header.Typeflag)
			continue
		}
	}

	return nil
}

// CompleteGitHubInstall completes the installation of a skill from GitHub
func (p *SkillPackager) CompleteGitHubInstall(ctx context.Context, orgID, repoID, userID int64, url, branch, path, scope string) (*extension.InstalledSkill, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}

	pkg, err := p.PackageFromGitHub(ctx, url, branch, path)
	if err != nil {
		return nil, err
	}

	sourceURL := url
	if branch != "" {
		sourceURL = fmt.Sprintf("%s@%s", url, branch)
	}
	if path != "" {
		sourceURL = fmt.Sprintf("%s#%s", sourceURL, path)
	}

	skill := &extension.InstalledSkill{
		OrganizationID: orgID,
		RepositoryID:   repoID,
		Scope:          scope,
		InstalledBy:    &userID,
		Slug:           pkg.Slug,
		InstallSource:  "github",
		SourceURL:      sourceURL,
		ContentSha:     pkg.ContentSha,
		StorageKey:     pkg.StorageKey,
		PackageSize:    pkg.PackageSize,
		IsEnabled:      true,
	}

	if err := p.repo.CreateInstalledSkill(ctx, skill); err != nil {
		if errors.Is(err, extension.ErrDuplicateInstall) {
			return nil, fmt.Errorf("%w: skill '%s' is already installed in this repository with scope '%s'", ErrAlreadyInstalled, skill.Slug, scope)
		}
		return nil, fmt.Errorf("failed to create installed skill: %w", err)
	}

	slog.Info("Skill installed from GitHub",
		"slug", pkg.Slug, "org_id", orgID, "repo_id", repoID)

	return skill, nil
}

// CompleteUploadInstall completes the installation of a skill from upload
func (p *SkillPackager) CompleteUploadInstall(ctx context.Context, orgID, repoID, userID int64, reader io.Reader, filename, scope string) (*extension.InstalledSkill, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}

	pkg, err := p.PackageFromUpload(ctx, reader, filename)
	if err != nil {
		return nil, err
	}

	skill := &extension.InstalledSkill{
		OrganizationID: orgID,
		RepositoryID:   repoID,
		Scope:          scope,
		InstalledBy:    &userID,
		Slug:           pkg.Slug,
		InstallSource:  "upload",
		ContentSha:     pkg.ContentSha,
		StorageKey:     pkg.StorageKey,
		PackageSize:    pkg.PackageSize,
		IsEnabled:      true,
	}

	if err := p.repo.CreateInstalledSkill(ctx, skill); err != nil {
		if errors.Is(err, extension.ErrDuplicateInstall) {
			return nil, fmt.Errorf("%w: skill '%s' is already installed in this repository with scope '%s'", ErrAlreadyInstalled, skill.Slug, scope)
		}
		return nil, fmt.Errorf("failed to create installed skill: %w", err)
	}

	slog.Info("Skill installed from upload",
		"slug", pkg.Slug, "org_id", orgID, "repo_id", repoID)

	return skill, nil
}
