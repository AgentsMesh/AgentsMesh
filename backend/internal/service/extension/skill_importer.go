package extension

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/internal/infra/storage"
)

// SkillImporter handles importing Skills from GitHub repositories.
// It clones repos, detects repo type (collection/single), parses SKILL.md,
// packages skills as tar.gz, and uploads to object storage.
type SkillImporter struct {
	repo    extension.Repository
	storage storage.Storage

	// credentialDecryptor decrypts auth credentials stored on SkillRegistry.
	// Set via SetCredentialDecryptor to avoid circular init.
	credentialDecryptor func(encrypted string) (string, error)

	// gitCloneFn can be overridden in tests to avoid real git operations.
	gitCloneFn func(ctx context.Context, url, branch, targetDir string) error
	// gitCloneAuthFn can be overridden in tests — clone with auth credential.
	gitCloneAuthFn func(ctx context.Context, url, branch, targetDir, authType, credential string) error
	// gitHeadFn can be overridden in tests to avoid real git operations.
	gitHeadFn func(ctx context.Context, repoDir string) (string, error)
}

// NewSkillImporter creates a new SkillImporter
func NewSkillImporter(repo extension.Repository, storage storage.Storage) *SkillImporter {
	return &SkillImporter{
		repo:    repo,
		storage: storage,
	}
}

// SetCredentialDecryptor sets the function used to decrypt auth credentials.
func (imp *SkillImporter) SetCredentialDecryptor(fn func(string) (string, error)) {
	imp.credentialDecryptor = fn
}

// SkillInfo holds parsed information about a discovered skill
type SkillInfo struct {
	Slug          string
	DisplayName   string
	Description   string
	License       string
	Compatibility string
	AllowedTools  string
	Category      string
	DirPath       string // absolute path to the skill directory
}

// SyncSource syncs a skill registry by cloning/pulling the repo and importing skills
func (imp *SkillImporter) SyncSource(ctx context.Context, sourceID int64) error {
	source, err := imp.repo.GetSkillRegistry(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("failed to get skill registry: %w", err)
	}

	// Prevent concurrent sync of the same registry
	if source.SyncStatus == "syncing" {
		return fmt.Errorf("registry %d is already syncing", sourceID)
	}

	// Update status
	source.SyncStatus = "syncing"
	source.SyncError = ""
	if err := imp.repo.UpdateSkillRegistry(ctx, source); err != nil {
		return fmt.Errorf("failed to update sync status: %w", err)
	}

	// Perform sync
	syncErr := imp.doSync(ctx, source)

	// Update final status
	now := time.Now()
	source.LastSyncedAt = &now
	if syncErr != nil {
		source.SyncStatus = "failed"
		source.SyncError = syncErr.Error()
		slog.Error("Skill registry sync failed",
			"registry_id", sourceID,
			"url", source.RepositoryURL,
			"error", syncErr)
	} else {
		source.SyncStatus = "success"
		source.SyncError = ""
	}

	if err := imp.repo.UpdateSkillRegistry(ctx, source); err != nil {
		slog.Error("Failed to update skill registry after sync",
			"registry_id", sourceID, "error", err)
	}

	return syncErr
}

// doSync performs the actual sync operation
func (imp *SkillImporter) doSync(ctx context.Context, source *extension.SkillRegistry) error {
	// 1. Clone repo to temp directory
	tmpDir, err := os.MkdirTemp("", "skill-sync-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	repoDir := filepath.Join(tmpDir, "repo")

	// Clone with or without auth
	if err := imp.cloneRepo(ctx, source, repoDir); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Get current commit SHA
	headFn := gitHead
	if imp.gitHeadFn != nil {
		headFn = imp.gitHeadFn
	}
	commitSha, err := headFn(ctx, repoDir)
	if err != nil {
		slog.Warn("Failed to get HEAD commit", "error", err)
	} else {
		source.LastCommitSha = commitSha
	}

	// 2. Detect repo type
	detectedType := detectRepoType(repoDir)
	source.DetectedType = detectedType

	// 3. Scan for skills
	var skills []SkillInfo
	switch detectedType {
	case "single":
		info, err := parseSkillDir(repoDir)
		if err != nil {
			return fmt.Errorf("failed to parse single skill: %w", err)
		}
		skills = append(skills, *info)
	case "collection":
		skills, err = scanCollectionSkills(repoDir)
		if err != nil {
			return fmt.Errorf("failed to scan collection: %w", err)
		}
	}

	if len(skills) == 0 {
		slog.Info("No skills found in source", "registry_id", source.ID, "url", source.RepositoryURL)
		source.SkillCount = 0
		return nil
	}

	// 4. Process each skill
	activeSlugs := make([]string, 0, len(skills))
	for _, skillInfo := range skills {
		// Always keep the slug active — even if processSkill fails, the skill
		// exists in the repo and may have a previous version in the DB.
		// Deactivating on transient errors (e.g. storage failure) would break
		// existing installations that depend on the prior version.
		activeSlugs = append(activeSlugs, skillInfo.Slug)
		if err := imp.processSkill(ctx, source, skillInfo); err != nil {
			slog.Error("Failed to process skill",
				"slug", skillInfo.Slug,
				"registry_id", source.ID,
				"error", err)
			continue
		}
	}

	// 5. Deactivate skills no longer in the repo
	if err := imp.repo.DeactivateSkillMarketItemsNotIn(ctx, source.ID, activeSlugs); err != nil {
		slog.Error("Failed to deactivate removed skills", "registry_id", source.ID, "error", err)
	}

	source.SkillCount = len(activeSlugs)
	return nil
}

// processSkill handles a single discovered skill: compute SHA, package, upload, upsert DB record
func (imp *SkillImporter) processSkill(ctx context.Context, source *extension.SkillRegistry, info SkillInfo) error {
	// Compute content SHA
	contentSha, err := computeDirSHA(info.DirPath)
	if err != nil {
		return fmt.Errorf("failed to compute SHA: %w", err)
	}

	// Check if already exists with same SHA
	existing, _ := imp.repo.FindSkillMarketItemBySlug(ctx, source.ID, info.Slug)
	if existing != nil && existing.ContentSha == contentSha {
		// SHA unchanged, ensure active
		if !existing.IsActive {
			existing.IsActive = true
			return imp.repo.UpdateSkillMarketItem(ctx, existing)
		}
		return nil
	}

	// Package as tar.gz
	packageData, err := packageSkillDir(info.DirPath)
	if err != nil {
		return fmt.Errorf("failed to package skill: %w", err)
	}

	// Upload to storage
	storageKey := fmt.Sprintf("skills/%d/%s/%s.tar.gz", source.ID, info.Slug, contentSha)
	_, err = imp.storage.Upload(ctx, storageKey, bytes.NewReader(packageData), int64(len(packageData)), "application/gzip")
	if err != nil {
		return fmt.Errorf("failed to upload skill package: %w", err)
	}

	// Upsert market item
	if existing != nil {
		existing.ContentSha = contentSha
		existing.StorageKey = storageKey
		existing.PackageSize = int64(len(packageData))
		existing.Version++
		existing.DisplayName = info.DisplayName
		existing.Description = info.Description
		existing.License = info.License
		existing.Compatibility = info.Compatibility
		existing.AllowedTools = info.AllowedTools
		existing.Category = info.Category
		existing.AgentTypeFilter = source.CompatibleAgents // propagate from source
		existing.IsActive = true
		return imp.repo.UpdateSkillMarketItem(ctx, existing)
	}

	item := &extension.SkillMarketItem{
		RegistryID:      source.ID,
		Slug:            info.Slug,
		DisplayName:     info.DisplayName,
		Description:     info.Description,
		License:         info.License,
		Compatibility:   info.Compatibility,
		AllowedTools:    info.AllowedTools,
		Category:        info.Category,
		ContentSha:      contentSha,
		StorageKey:      storageKey,
		PackageSize:     int64(len(packageData)),
		Version:         1,
		AgentTypeFilter: source.CompatibleAgents, // propagate from source
		IsActive:        true,
	}
	return imp.repo.CreateSkillMarketItem(ctx, item)
}

// --- Repo type detection ---

// detectRepoType determines if a repo is a single-skill or collection
func detectRepoType(repoDir string) string {
	// Check root for SKILL.md → single
	if fileExists(filepath.Join(repoDir, "SKILL.md")) {
		return "single"
	}
	return "collection"
}

// --- Skill scanning ---

// scanCollectionSkills scans a collection repo for skill directories
func scanCollectionSkills(repoDir string) ([]SkillInfo, error) {
	var skills []SkillInfo

	// Priority 1: Check skills/ subdirectory
	skillsDir := filepath.Join(repoDir, "skills")
	if dirExists(skillsDir) {
		entries, err := os.ReadDir(skillsDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() || shouldIgnoreDir(entry.Name()) {
					continue
				}
				dirPath := filepath.Join(skillsDir, entry.Name())
				if fileExists(filepath.Join(dirPath, "SKILL.md")) {
					info, err := parseSkillDir(dirPath)
					if err != nil {
						slog.Warn("Failed to parse skill", "dir", dirPath, "error", err)
						continue
					}
					skills = append(skills, *info)
				}
			}
		}
		if len(skills) > 0 {
			return skills, nil
		}
	}

	// Priority 2: Scan root-level subdirectories
	entries, err := os.ReadDir(repoDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read repo dir: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() || shouldIgnoreDir(entry.Name()) {
			continue
		}
		dirPath := filepath.Join(repoDir, entry.Name())
		if fileExists(filepath.Join(dirPath, "SKILL.md")) {
			info, err := parseSkillDir(dirPath)
			if err != nil {
				slog.Warn("Failed to parse skill", "dir", dirPath, "error", err)
				continue
			}
			skills = append(skills, *info)
		}
	}

	return skills, nil
}

// ignoredDirs is the set of directory names to skip during skill scanning.
// Extracted as a package-level variable to avoid re-creating the map on every call.
var ignoredDirs = map[string]bool{
	".git": true, ".github": true, ".vscode": true,
	"spec": true, "template": true, "templates": true, ".claude-plugin": true,
	"node_modules": true, "__pycache__": true, "vendor": true,
}

// shouldIgnoreDir returns true for directories to skip during scanning
func shouldIgnoreDir(name string) bool {
	return strings.HasPrefix(name, ".") || ignoredDirs[name]
}

// parseSkillDir parses a skill directory's SKILL.md frontmatter
func parseSkillDir(dirPath string) (*SkillInfo, error) {
	skillMdPath := filepath.Join(dirPath, "SKILL.md")
	content, err := os.ReadFile(skillMdPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SKILL.md: %w", err)
	}

	fm := parseFrontmatter(string(content))

	slug := fm["name"]
	if slug == "" {
		// Fallback to directory name
		slug = filepath.Base(dirPath)
	}

	return &SkillInfo{
		Slug:          slug,
		DisplayName:   fm["name"],
		Description:   fm["description"],
		License:       fm["license"],
		Compatibility: fm["compatibility"],
		AllowedTools:  fm["allowed-tools"],
		Category:      fm["category"],
		DirPath:       dirPath,
	}, nil
}

// parseFrontmatter extracts YAML-like frontmatter from a markdown file
// Supports simple key: value pairs between --- delimiters
func parseFrontmatter(content string) map[string]string {
	fm := make(map[string]string)

	lines := strings.Split(content, "\n")
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "---" {
		return fm
	}

	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "---" {
			break
		}
		// Simple key: value parsing
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Remove surrounding quotes
			value = strings.Trim(value, `"'`)
			fm[key] = value
		}
	}

	return fm
}

// --- Packaging ---

// packageSkillDir creates a tar.gz archive of a skill directory
func packageSkillDir(dirPath string) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	baseDir := dirPath
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}

		// Skip root directory itself
		if relPath == "." {
			return nil
		}

		// Skip .git and other ignored directories
		if info.IsDir() && shouldIgnoreDir(info.Name()) {
			return filepath.SkipDir
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(tw, f)
			f.Close()
			if copyErr != nil {
				return copyErr
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// --- SHA computation ---

// computeDirSHA computes a deterministic SHA256 of a directory's contents
func computeDirSHA(dirPath string) (string, error) {
	h := sha256.New()

	// Collect all file paths
	var files []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if shouldIgnoreDir(info.Name()) && path != dirPath {
				return filepath.SkipDir
			}
			return nil
		}
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}
		files = append(files, relPath)
		return nil
	})
	if err != nil {
		return "", err
	}

	// Sort for deterministic ordering
	sort.Strings(files)

	// Hash each file's path + content
	for _, relPath := range files {
		absPath := filepath.Join(dirPath, relPath)
		content, err := os.ReadFile(absPath)
		if err != nil {
			return "", err
		}
		// Write path and content to hasher
		h.Write([]byte(relPath))
		h.Write([]byte{0}) // separator
		h.Write(content)
		h.Write([]byte{0}) // separator
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// --- Clone helpers ---

// cloneRepo clones the source's repository using auth if configured.
func (imp *SkillImporter) cloneRepo(ctx context.Context, source *extension.SkillRegistry, targetDir string) error {
	if source.HasAuth() && source.AuthCredential != "" {
		// Decrypt credential
		credential := source.AuthCredential
		if imp.credentialDecryptor != nil {
			decrypted, err := imp.credentialDecryptor(credential)
			if err != nil {
				slog.Warn("Failed to decrypt auth credential, attempting raw value",
					"registry_id", source.ID, "error", err)
			} else {
				credential = decrypted
			}
		}

		cloneAuthFn := gitCloneWithAuth
		if imp.gitCloneAuthFn != nil {
			cloneAuthFn = imp.gitCloneAuthFn
		}
		return cloneAuthFn(ctx, source.RepositoryURL, source.Branch, targetDir, source.AuthType, credential)
	}

	// No auth — public repo clone
	cloneFn := gitClone
	if imp.gitCloneFn != nil {
		cloneFn = imp.gitCloneFn
	}
	return cloneFn(ctx, source.RepositoryURL, source.Branch, targetDir)
}

// --- Git helpers ---

// validateGitBranch validates that a branch name contains only safe characters
func validateGitBranch(branch string) error {
	for _, c := range branch {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '/') {
			return fmt.Errorf("invalid branch name character: %c", c)
		}
	}
	return nil
}

// gitCloneWithAuth clones a repo using the specified auth method.
// For PAT-based auth, the token is injected into the URL so git receives it via HTTPS.
// For SSH key auth, a temporary identity file is created and passed via GIT_SSH_COMMAND.
func gitCloneWithAuth(ctx context.Context, repoURL, branch, targetDir, authType, credential string) error {
	switch authType {
	case extension.AuthTypeGitHubPAT:
		// GitHub PAT: inject as https://<token>@github.com/owner/repo.git
		authedURL, err := injectPATIntoURL(repoURL, credential)
		if err != nil {
			return fmt.Errorf("failed to build authenticated URL: %w", err)
		}
		return gitClone(ctx, authedURL, branch, targetDir)

	case extension.AuthTypeGitLabPAT:
		// GitLab PAT: inject as https://oauth2:<token>@gitlab.com/owner/repo.git
		authedURL, err := injectGitLabPATIntoURL(repoURL, credential)
		if err != nil {
			return fmt.Errorf("failed to build authenticated URL: %w", err)
		}
		return gitClone(ctx, authedURL, branch, targetDir)

	case extension.AuthTypeSSHKey:
		return gitCloneWithSSHKey(ctx, repoURL, branch, targetDir, credential)

	default:
		// Fall back to unauthenticated clone
		return gitClone(ctx, repoURL, branch, targetDir)
	}
}

// injectPATIntoURL inserts a GitHub PAT into an HTTPS URL.
// https://github.com/owner/repo → https://<token>@github.com/owner/repo
func injectPATIntoURL(repoURL, token string) (string, error) {
	if !strings.HasPrefix(repoURL, "https://") {
		return "", fmt.Errorf("PAT auth requires https:// URL, got: %s", repoURL)
	}
	// Strip "https://" and prepend token
	rest := strings.TrimPrefix(repoURL, "https://")
	return fmt.Sprintf("https://%s@%s", token, rest), nil
}

// injectGitLabPATIntoURL inserts a GitLab PAT using oauth2 username.
// https://gitlab.com/owner/repo → https://oauth2:<token>@gitlab.com/owner/repo
func injectGitLabPATIntoURL(repoURL, token string) (string, error) {
	if !strings.HasPrefix(repoURL, "https://") {
		return "", fmt.Errorf("PAT auth requires https:// URL, got: %s", repoURL)
	}
	rest := strings.TrimPrefix(repoURL, "https://")
	return fmt.Sprintf("https://oauth2:%s@%s", token, rest), nil
}

// gitCloneWithSSHKey clones using a temporary SSH identity file.
// Only git@ SSH URLs and local paths are allowed to prevent SSRF via arbitrary protocols.
func gitCloneWithSSHKey(ctx context.Context, repoURL, branch, targetDir, sshKey string) error {
	// Validate URL format — allow git@ SSH URLs and local filesystem paths,
	// but reject http/https/ftp protocols to prevent SSRF.
	isGitSSH := strings.HasPrefix(repoURL, "git@")
	isLocalPath := strings.HasPrefix(repoURL, "/") || strings.HasPrefix(repoURL, ".")
	if !isGitSSH && !isLocalPath {
		return fmt.Errorf("SSH key auth requires git@ URL, got: %s", repoURL)
	}

	// Write SSH key to temporary file
	tmpKeyFile, err := os.CreateTemp("", "skill-ssh-key-*")
	if err != nil {
		return fmt.Errorf("failed to create temp SSH key file: %w", err)
	}
	defer os.Remove(tmpKeyFile.Name())

	if _, err := tmpKeyFile.WriteString(sshKey); err != nil {
		tmpKeyFile.Close()
		return fmt.Errorf("failed to write SSH key: %w", err)
	}
	tmpKeyFile.Close()

	// Set proper permissions (600)
	if err := os.Chmod(tmpKeyFile.Name(), 0600); err != nil {
		return fmt.Errorf("failed to set SSH key permissions: %w", err)
	}

	// Validate branch name if provided
	if branch != "" {
		if err := validateGitBranch(branch); err != nil {
			return fmt.Errorf("invalid branch: %w", err)
		}
	}

	args := []string{"clone", "--depth", "1"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, "--", repoURL, targetDir)

	cmd := exec.CommandContext(ctx, "git", args...)
	sshCommand := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null", tmpKeyFile.Name())
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_SSH_COMMAND="+sshCommand,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		sanitized := sanitizeGitOutput(string(output))
		return fmt.Errorf("git clone with SSH key failed: %s: %w", sanitized, err)
	}
	return nil
}

func gitClone(ctx context.Context, url, branch, targetDir string) error {
	// Validate URL scheme - only allow https://
	if !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("only https:// URLs are allowed for git clone, got: %s", url)
	}

	// Validate branch name if provided
	if branch != "" {
		if err := validateGitBranch(branch); err != nil {
			return fmt.Errorf("invalid branch: %w", err)
		}
	}

	args := []string{"clone", "--depth", "1"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	// Use -- separator to prevent argument injection
	args = append(args, "--", url, targetDir)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Sanitize output to prevent PAT tokens from leaking into logs/errors
		sanitized := sanitizeGitOutput(string(output))
		return fmt.Errorf("git clone failed: %s: %w", sanitized, err)
	}
	return nil
}

// sanitizeGitOutput removes potential credentials from git command output.
// PAT tokens embedded in HTTPS URLs (e.g. https://<token>@github.com) are redacted.
func sanitizeGitOutput(output string) string {
	// Redact HTTPS URLs that contain embedded credentials
	// Pattern: https://<anything>@<host>
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "https://"); idx >= 0 {
			if atIdx := strings.Index(line[idx:], "@"); atIdx > 8 {
				// Redact the credential portion
				lines[i] = line[:idx] + "https://[REDACTED]" + line[idx+atIdx:]
			}
		}
	}
	return strings.Join(lines, "\n")
}

func gitHead(ctx context.Context, repoDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = repoDir

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// --- File helpers ---

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
