package extension

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// --- Repo Skills Installation ---

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
		return nil, fmt.Errorf("failed to install skill: %w", err)
	}

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

	// Validate org ownership
	if skill.OrganizationID != orgID {
		return nil, fmt.Errorf("%w: skill does not belong to this organization", ErrForbidden)
	}

	// Validate repository ownership
	if skill.RepositoryID != repoID {
		return nil, fmt.Errorf("%w: skill does not belong to this repository", ErrForbidden)
	}

	if skill.Scope == extension.ScopeOrg && userRole != "admin" && userRole != "owner" {
		return nil, fmt.Errorf("%w: admin permission required to modify org-scoped skill", ErrForbidden)
	}

	if skill.Scope == extension.ScopeUser {
		if skill.InstalledBy == nil || *skill.InstalledBy != userID {
			return nil, fmt.Errorf("%w: can only modify your own user-scoped skill", ErrForbidden)
		}
	}

	if enabled != nil {
		skill.IsEnabled = *enabled
	}
	if pinnedVersion != nil {
		skill.PinnedVersion = pinnedVersion
	}

	if err := s.repo.UpdateInstalledSkill(ctx, skill); err != nil {
		return nil, fmt.Errorf("failed to update skill: %w", err)
	}

	return skill, nil
}

// UninstallSkill removes an installed skill from a repository.
func (s *Service) UninstallSkill(ctx context.Context, orgID, repoID, installID, userID int64, userRole string) error {
	skill, err := s.repo.GetInstalledSkill(ctx, installID)
	if err != nil {
		return fmt.Errorf("%w: skill %d", ErrNotFound, installID)
	}

	// Validate org ownership
	if skill.OrganizationID != orgID {
		return fmt.Errorf("%w: skill does not belong to this organization", ErrForbidden)
	}

	// Validate repository ownership
	if skill.RepositoryID != repoID {
		return fmt.Errorf("%w: skill does not belong to this repository", ErrForbidden)
	}

	// Org-scoped skills require admin/owner role to uninstall
	if skill.Scope == extension.ScopeOrg && userRole != "admin" && userRole != "owner" {
		return fmt.Errorf("%w: admin permission required to uninstall org-scoped skill", ErrForbidden)
	}

	// User-scoped skills can only be uninstalled by the user who installed them
	if skill.Scope == extension.ScopeUser {
		if skill.InstalledBy == nil || *skill.InstalledBy != userID {
			return fmt.Errorf("%w: can only uninstall your own user-scoped skill", ErrForbidden)
		}
	}

	return s.repo.DeleteInstalledSkill(ctx, installID)
}

// --- Repo MCP Server Installation ---

// InstallMcpFromMarket installs an MCP server from the marketplace.
func (s *Service) InstallMcpFromMarket(ctx context.Context, orgID, repoID, userID, marketItemID int64, envVars map[string]string, scope string) (*extension.InstalledMcpServer, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}

	marketItem, err := s.repo.GetMcpMarketItem(ctx, marketItemID)
	if err != nil {
		return nil, fmt.Errorf("%w: MCP market item %d", ErrNotFound, marketItemID)
	}

	// Verify the market item is active
	if !marketItem.IsActive {
		return nil, fmt.Errorf("%w: MCP market item %d is not active", ErrNotFound, marketItemID)
	}

	server := &extension.InstalledMcpServer{
		OrganizationID: orgID,
		RepositoryID:   repoID,
		MarketItemID:   &marketItemID,
		Scope:          scope,
		InstalledBy:    &userID,
		Name:           marketItem.Name,
		Slug:           marketItem.Slug,
		TransportType:  marketItem.TransportType,
		Command:        marketItem.Command,
		Args:           marketItem.DefaultArgs,
		IsEnabled:      true,
	}

	// Encrypt env vars
	if len(envVars) > 0 {
		encrypted, err := s.encryptEnvVars(envVars)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt env vars: %w", err)
		}
		server.EnvVars = encrypted
	}

	if err := s.repo.CreateInstalledMcpServer(ctx, server); err != nil {
		if errors.Is(err, extension.ErrDuplicateInstall) {
			return nil, fmt.Errorf("%w: MCP server '%s' is already installed in this repository with scope '%s'", ErrAlreadyInstalled, server.Slug, server.Scope)
		}
		return nil, fmt.Errorf("failed to install MCP server: %w", err)
	}

	return server, nil
}

// InstallCustomMcpServer installs a custom MCP server configuration.
func (s *Service) InstallCustomMcpServer(ctx context.Context, orgID, repoID, userID int64, server *extension.InstalledMcpServer, envVars map[string]string) (*extension.InstalledMcpServer, error) {
	if err := validateScope(server.Scope); err != nil {
		return nil, err
	}

	server.OrganizationID = orgID
	server.RepositoryID = repoID
	server.InstalledBy = &userID
	server.IsEnabled = true

	// Encrypt env vars
	if len(envVars) > 0 {
		encrypted, err := s.encryptEnvVars(envVars)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt env vars: %w", err)
		}
		server.EnvVars = encrypted
	}

	if err := s.repo.CreateInstalledMcpServer(ctx, server); err != nil {
		if errors.Is(err, extension.ErrDuplicateInstall) {
			return nil, fmt.Errorf("%w: MCP server '%s' is already installed in this repository with scope '%s'", ErrAlreadyInstalled, server.Slug, server.Scope)
		}
		return nil, fmt.Errorf("failed to install custom MCP server: %w", err)
	}

	return server, nil
}

// UpdateMcpServer updates an installed MCP server's settings.
func (s *Service) UpdateMcpServer(ctx context.Context, orgID, repoID, installID, userID int64, userRole string, enabled *bool, envVars map[string]string) (*extension.InstalledMcpServer, error) {
	server, err := s.repo.GetInstalledMcpServer(ctx, installID)
	if err != nil {
		return nil, fmt.Errorf("%w: MCP server %d", ErrNotFound, installID)
	}

	// Validate org ownership
	if server.OrganizationID != orgID {
		return nil, fmt.Errorf("%w: MCP server does not belong to this organization", ErrForbidden)
	}

	// Validate repository ownership
	if server.RepositoryID != repoID {
		return nil, fmt.Errorf("%w: MCP server does not belong to this repository", ErrForbidden)
	}

	// Org-scoped MCP servers require admin/owner role
	if server.Scope == extension.ScopeOrg && userRole != "admin" && userRole != "owner" {
		return nil, fmt.Errorf("%w: admin permission required to modify org-scoped MCP server", ErrForbidden)
	}

	if server.Scope == extension.ScopeUser {
		if server.InstalledBy == nil || *server.InstalledBy != userID {
			return nil, fmt.Errorf("%w: can only modify your own user-scoped MCP server", ErrForbidden)
		}
	}

	if enabled != nil {
		server.IsEnabled = *enabled
	}
	if envVars != nil {
		encrypted, err := s.encryptEnvVars(envVars)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt env vars: %w", err)
		}
		server.EnvVars = encrypted
	}

	if err := s.repo.UpdateInstalledMcpServer(ctx, server); err != nil {
		return nil, fmt.Errorf("failed to update MCP server: %w", err)
	}

	return server, nil
}

// UninstallMcpServer removes an installed MCP server from a repository.
func (s *Service) UninstallMcpServer(ctx context.Context, orgID, repoID, installID, userID int64, userRole string) error {
	server, err := s.repo.GetInstalledMcpServer(ctx, installID)
	if err != nil {
		return fmt.Errorf("%w: MCP server %d", ErrNotFound, installID)
	}

	// Validate org ownership
	if server.OrganizationID != orgID {
		return fmt.Errorf("%w: MCP server does not belong to this organization", ErrForbidden)
	}

	// Validate repository ownership
	if server.RepositoryID != repoID {
		return fmt.Errorf("%w: MCP server does not belong to this repository", ErrForbidden)
	}

	// Org-scoped MCP servers require admin/owner role
	if server.Scope == extension.ScopeOrg && userRole != "admin" && userRole != "owner" {
		return fmt.Errorf("%w: admin permission required to uninstall org-scoped MCP server", ErrForbidden)
	}

	if server.Scope == extension.ScopeUser {
		if server.InstalledBy == nil || *server.InstalledBy != userID {
			return fmt.Errorf("%w: can only uninstall your own user-scoped MCP server", ErrForbidden)
		}
	}

	return s.repo.DeleteInstalledMcpServer(ctx, installID)
}
