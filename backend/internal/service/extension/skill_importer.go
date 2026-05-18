package extension

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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

	// staleSyncTimeout is the threshold beyond which a registry stuck in
	// sync_status="syncing" is considered abandoned and may be reclaimed.
	// Tunable via SetStaleSyncTimeout (default: defaultStaleSyncTimeout).
	staleSyncTimeout time.Duration

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
		repo:             repo,
		storage:          storage,
		staleSyncTimeout: defaultStaleSyncTimeout,
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

// SyncSource syncs a skill registry by cloning/pulling the repo and importing skills.
// The lock is claimed atomically via the repository before any work begins —
// concurrent callers either get the lock or ErrSyncInProgress.
func (imp *SkillImporter) SyncSource(ctx context.Context, sourceID int64) error {
	claimed, wasStale, err := imp.repo.ClaimSyncLock(ctx, sourceID, imp.staleSyncTimeout)
	if err != nil {
		return fmt.Errorf("failed to claim sync lock: %w", err)
	}
	if !claimed {
		return fmt.Errorf("%w: registry %d", ErrSyncInProgress, sourceID)
	}
	if wasStale {
		slog.WarnContext(ctx, "reclaimed stale syncing registry", "registry_id", sourceID)
	}

	source, err := imp.repo.GetSkillRegistry(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("failed to get skill registry: %w", err)
	}

	syncErr := imp.doSync(ctx, source)

	now := time.Now()
	source.LastSyncedAt = &now
	if syncErr != nil {
		source.SyncStatus = extension.SyncStatusFailed
		source.SyncError = syncErr.Error()
		slog.ErrorContext(ctx, "Skill registry sync failed",
			"registry_id", sourceID,
			"url", source.RepositoryURL,
			"error", syncErr)
	} else {
		source.SyncStatus = extension.SyncStatusSuccess
		source.SyncError = ""
	}

	if err := imp.repo.UpdateSkillRegistry(ctx, source); err != nil {
		slog.ErrorContext(ctx, "Failed to update skill registry after sync",
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
		slog.WarnContext(ctx, "Failed to get HEAD commit", "error", err)
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
		slog.InfoContext(ctx, "No skills found in source", "registry_id", source.ID, "url", source.RepositoryURL)
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
			slog.ErrorContext(ctx, "Failed to process skill",
				"slug", skillInfo.Slug,
				"registry_id", source.ID,
				"error", err)
			continue
		}
	}

	// 5. Deactivate skills no longer in the repo. SkillCount is derived at
	// query time by SkillRegistryRepository — no field write needed here.
	if err := imp.repo.DeactivateSkillMarketItemsNotIn(ctx, source.ID, activeSlugs); err != nil {
		slog.ErrorContext(ctx, "Failed to deactivate removed skills", "registry_id", source.ID, "error", err)
	}

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
	return imp.upsertMarketItem(ctx, source, existing, info, contentSha, storageKey, packageData)
}

// upsertMarketItem creates or updates a skill market item in the database.
func (imp *SkillImporter) upsertMarketItem(
	ctx context.Context,
	source *extension.SkillRegistry,
	existing *extension.SkillMarketItem,
	info SkillInfo,
	contentSha, storageKey string,
	packageData []byte,
) error {
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
		existing.AgentFilter = source.CompatibleAgents
		existing.IsActive = true
		return imp.repo.UpdateSkillMarketItem(ctx, existing)
	}

	item := &extension.SkillMarketItem{
		RegistryID:    source.ID,
		Slug:          info.Slug,
		DisplayName:   info.DisplayName,
		Description:   info.Description,
		License:       info.License,
		Compatibility: info.Compatibility,
		AllowedTools:  info.AllowedTools,
		Category:      info.Category,
		ContentSha:    contentSha,
		StorageKey:    storageKey,
		PackageSize:   int64(len(packageData)),
		Version:       1,
		AgentFilter:   source.CompatibleAgents,
		IsActive:      true,
	}
	return imp.repo.CreateSkillMarketItem(ctx, item)
}

// cloneRepo clones the source's repository using auth if configured.
func (imp *SkillImporter) cloneRepo(ctx context.Context, source *extension.SkillRegistry, targetDir string) error {
	if source.HasAuth() && source.AuthCredential != "" {
		// Decrypt credential
		credential := source.AuthCredential
		if imp.credentialDecryptor != nil {
			decrypted, err := imp.credentialDecryptor(credential)
			if err != nil {
				slog.WarnContext(ctx, "Failed to decrypt auth credential, attempting raw value",
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
