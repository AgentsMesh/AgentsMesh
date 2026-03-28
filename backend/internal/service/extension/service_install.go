package extension

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// InstallSkillFromMarket installs a skill from the marketplace into a repository.
func (s *Service) InstallSkillFromMarket(ctx context.Context, orgID, repoID, userID, marketItemID int64, scope string) (*extension.InstalledSkill, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}

	marketItem, err := s.repo.GetSkillMarketItem(ctx, marketItemID)
	if err != nil {
		return nil, fmt.Errorf("%w: market item %d", ErrNotFound, marketItemID)
	}

	// Validate market item accessibility: must be platform-level (nil org) or belong to same org
	if marketItem.Registry != nil && marketItem.Registry.OrganizationID != nil &&
		*marketItem.Registry.OrganizationID != orgID {
		return nil, fmt.Errorf("%w: skill not accessible to this organization", ErrForbidden)
	}

	skill := &extension.InstalledSkill{
		OrganizationID: orgID,
		RepositoryID:   repoID,
		MarketItemID:   &marketItemID,
		Scope:          scope,
		InstalledBy:    &userID,
		Slug:           marketItem.Slug,
		InstallSource:  "market",
		ContentSha:     marketItem.ContentSha,
		StorageKey:     marketItem.StorageKey,
		PackageSize:    marketItem.PackageSize,
		IsEnabled:      true,
	}

	if err := s.repo.CreateInstalledSkill(ctx, skill); err != nil {
		if errors.Is(err, extension.ErrDuplicateInstall) {
			return nil, fmt.Errorf("%w: skill '%s' is already installed in this repository with scope '%s'", ErrAlreadyInstalled, skill.Slug, skill.Scope)
		}
		slog.Error("failed to install skill from market", "slug", skill.Slug, "org_id", orgID, "repo_id", repoID, "error", err)
		return nil, fmt.Errorf("failed to install skill: %w", err)
	}

	slog.Info("skill installed from market", "slug", skill.Slug, "org_id", orgID, "repo_id", repoID, "scope", scope)
	return skill, nil
}

// InstallSkillFromGitHub imports and installs a skill from a GitHub URL.
func (s *Service) InstallSkillFromGitHub(ctx context.Context, orgID, repoID, userID int64, url, branch, path, scope string) (*extension.InstalledSkill, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}

	if s.packager == nil {
		return nil, fmt.Errorf("skill packager not configured")
	}

	return s.packager.CompleteGitHubInstall(ctx, orgID, repoID, userID, url, branch, path, scope)
}

// InstallSkillFromUpload processes an uploaded archive and installs it as a skill.
func (s *Service) InstallSkillFromUpload(ctx context.Context, orgID, repoID, userID int64, reader io.Reader, filename, scope string) (*extension.InstalledSkill, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}

	if s.packager == nil {
		return nil, fmt.Errorf("skill packager not configured")
	}

	return s.packager.CompleteUploadInstall(ctx, orgID, repoID, userID, reader, filename, scope)
}

// UpdateSkill updates an installed skill's settings.
func (s *Service) UpdateSkill(ctx context.Context, orgID, repoID, installID, userID int64, userRole string, enabled *bool, pinnedVersion *int) (*extension.InstalledSkill, error) {
	skill, err := s.repo.GetInstalledSkill(ctx, installID)
	if err != nil {
		return nil, fmt.Errorf("%w: skill %d", ErrNotFound, installID)
	}

	if err := validateSkillAccess(skill, orgID, repoID, userID, userRole); err != nil {
		return nil, err
	}

	if enabled != nil {
		skill.IsEnabled = *enabled
	}
	if pinnedVersion != nil {
		skill.PinnedVersion = pinnedVersion
	}

	if err := s.repo.UpdateInstalledSkill(ctx, skill); err != nil {
		slog.Error("failed to update installed skill", "install_id", installID, "org_id", orgID, "error", err)
		return nil, fmt.Errorf("failed to update skill: %w", err)
	}

	slog.Info("skill updated", "install_id", installID, "org_id", orgID, "repo_id", repoID, "slug", skill.Slug)
	return skill, nil
}

// UninstallSkill removes an installed skill from a repository.
func (s *Service) UninstallSkill(ctx context.Context, orgID, repoID, installID, userID int64, userRole string) error {
	skill, err := s.repo.GetInstalledSkill(ctx, installID)
	if err != nil {
		return fmt.Errorf("%w: skill %d", ErrNotFound, installID)
	}

	if err := validateSkillAccess(skill, orgID, repoID, userID, userRole); err != nil {
		return err
	}

	if err := s.repo.DeleteInstalledSkill(ctx, installID); err != nil {
		slog.Error("failed to uninstall skill", "install_id", installID, "slug", skill.Slug, "org_id", orgID, "error", err)
		return err
	}

	slog.Info("skill uninstalled", "install_id", installID, "slug", skill.Slug, "org_id", orgID, "repo_id", repoID)
	return nil
}

// validateSkillAccess checks org/repo ownership and scope-based permissions.
func validateSkillAccess(skill *extension.InstalledSkill, orgID, repoID, userID int64, userRole string) error {
	if skill.OrganizationID != orgID {
		return fmt.Errorf("%w: skill does not belong to this organization", ErrForbidden)
	}
	if skill.RepositoryID != repoID {
		return fmt.Errorf("%w: skill does not belong to this repository", ErrForbidden)
	}
	if skill.Scope == extension.ScopeOrg && userRole != "admin" && userRole != "owner" {
		return fmt.Errorf("%w: admin permission required to modify org-scoped skill", ErrForbidden)
	}
	if skill.Scope == extension.ScopeUser {
		if skill.InstalledBy == nil || *skill.InstalledBy != userID {
			return fmt.Errorf("%w: can only modify your own user-scoped skill", ErrForbidden)
		}
	}
	return nil
}
