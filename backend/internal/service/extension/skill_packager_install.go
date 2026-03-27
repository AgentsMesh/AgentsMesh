package extension

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

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
