package extension

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// InstallMcpFromMarket installs an MCP server from the marketplace.
func (s *Service) InstallMcpFromMarket(ctx context.Context, orgID, repoID, userID, marketItemID int64, envVars map[string]string, scope string) (*extension.InstalledMcpServer, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}

	marketItem, err := s.repo.GetMcpMarketItem(ctx, marketItemID)
	if err != nil {
		return nil, fmt.Errorf("%w: MCP market item %d", ErrNotFound, marketItemID)
	}

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
		slog.Error("failed to install MCP server from market", "slug", server.Slug, "org_id", orgID, "repo_id", repoID, "error", err)
		return nil, fmt.Errorf("failed to install MCP server: %w", err)
	}

	slog.Info("MCP server installed from market", "slug", server.Slug, "org_id", orgID, "repo_id", repoID, "scope", scope)
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
		slog.Error("failed to install custom MCP server", "slug", server.Slug, "org_id", orgID, "repo_id", repoID, "error", err)
		return nil, fmt.Errorf("failed to install custom MCP server: %w", err)
	}

	slog.Info("custom MCP server installed", "slug", server.Slug, "org_id", orgID, "repo_id", repoID, "scope", server.Scope)
	return server, nil
}

// UpdateMcpServer updates an installed MCP server's settings.
func (s *Service) UpdateMcpServer(ctx context.Context, orgID, repoID, installID, userID int64, userRole string, enabled *bool, envVars map[string]string) (*extension.InstalledMcpServer, error) {
	server, err := s.repo.GetInstalledMcpServer(ctx, installID)
	if err != nil {
		return nil, fmt.Errorf("%w: MCP server %d", ErrNotFound, installID)
	}

	if err := validateMcpServerAccess(server, orgID, repoID, userID, userRole); err != nil {
		return nil, err
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
		slog.Error("failed to update MCP server", "install_id", installID, "org_id", orgID, "error", err)
		return nil, fmt.Errorf("failed to update MCP server: %w", err)
	}

	slog.Info("MCP server updated", "install_id", installID, "slug", server.Slug, "org_id", orgID, "repo_id", repoID)
	return server, nil
}

// UninstallMcpServer removes an installed MCP server from a repository.
func (s *Service) UninstallMcpServer(ctx context.Context, orgID, repoID, installID, userID int64, userRole string) error {
	server, err := s.repo.GetInstalledMcpServer(ctx, installID)
	if err != nil {
		return fmt.Errorf("%w: MCP server %d", ErrNotFound, installID)
	}

	if err := validateMcpServerAccess(server, orgID, repoID, userID, userRole); err != nil {
		return err
	}

	if err := s.repo.DeleteInstalledMcpServer(ctx, installID); err != nil {
		slog.Error("failed to uninstall MCP server", "install_id", installID, "slug", server.Slug, "org_id", orgID, "error", err)
		return err
	}

	slog.Info("MCP server uninstalled", "install_id", installID, "slug", server.Slug, "org_id", orgID, "repo_id", repoID)
	return nil
}

// validateMcpServerAccess checks org/repo ownership and scope-based permissions.
func validateMcpServerAccess(server *extension.InstalledMcpServer, orgID, repoID, userID int64, userRole string) error {
	if server.OrganizationID != orgID {
		return fmt.Errorf("%w: MCP server does not belong to this organization", ErrForbidden)
	}
	if server.RepositoryID != repoID {
		return fmt.Errorf("%w: MCP server does not belong to this repository", ErrForbidden)
	}
	if server.Scope == extension.ScopeOrg && userRole != "admin" && userRole != "owner" {
		return fmt.Errorf("%w: admin permission required for org-scoped MCP server", ErrForbidden)
	}
	if server.Scope == extension.ScopeUser {
		if server.InstalledBy == nil || *server.InstalledBy != userID {
			return fmt.Errorf("%w: can only modify your own user-scoped MCP server", ErrForbidden)
		}
	}
	return nil
}
