package extension

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/internal/infra/storage"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
)

// Standard service-level errors for typed error handling in API layer
var (
	ErrNotFound         = errors.New("resource not found")
	ErrForbidden        = errors.New("access denied")
	ErrInvalidScope     = errors.New("invalid scope")
	ErrInvalidInput     = errors.New("invalid input")
	ErrAlreadyInstalled = errors.New("already installed")
)

// validateScope checks that scope is a valid value ("org" or "user")
func validateScope(scope string) error {
	if scope != extension.ScopeOrg && scope != extension.ScopeUser {
		return fmt.Errorf("%w: %s, must be 'org' or 'user'", ErrInvalidScope, scope)
	}
	return nil
}

// presignedURLExpiry is the duration for presigned download URLs
const presignedURLExpiry = 15 * time.Minute

// Service provides extension management capabilities including
// marketplace queries, skill/MCP installation management, and scope merging.
type Service struct {
	repo     extension.Repository
	storage  storage.Storage
	crypto   *crypto.Encryptor
	packager *SkillPackager
	importer *SkillImporter
}

// NewService creates a new extension service
func NewService(repo extension.Repository, storage storage.Storage, cryptoEncryptor *crypto.Encryptor) *Service {
	return &Service{
		repo:    repo,
		storage: storage,
		crypto:  cryptoEncryptor,
	}
}

// SetSkillPackager sets the SkillPackager dependency.
// This uses a setter to avoid circular initialization issues.
func (s *Service) SetSkillPackager(p *SkillPackager) {
	s.packager = p
}

// SetSkillImporter sets the SkillImporter dependency.
// This uses a setter to avoid circular initialization issues.
func (s *Service) SetSkillImporter(imp *SkillImporter) {
	s.importer = imp
}

// --- Skill Registries ---

// ListSkillRegistries lists skill registries for an organization (includes platform-level if orgID provided)
func (s *Service) ListSkillRegistries(ctx context.Context, orgID int64) ([]*extension.SkillRegistry, error) {
	return s.repo.ListSkillRegistries(ctx, &orgID)
}

// CreateSkillRegistryInput holds the input for creating a skill registry
type CreateSkillRegistryInput struct {
	RepositoryURL    string   `json:"repository_url"`
	Branch           string   `json:"branch"`
	SourceType       string   `json:"source_type"`
	CompatibleAgents []string `json:"compatible_agents"`
	AuthType         string   `json:"auth_type"`
	AuthCredential   string   `json:"auth_credential"`
}

// CreateSkillRegistry creates a new skill registry for an organization
func (s *Service) CreateSkillRegistry(ctx context.Context, orgID int64, input CreateSkillRegistryInput) (*extension.SkillRegistry, error) {
	if input.Branch == "" {
		input.Branch = "main"
	}
	if input.SourceType == "" {
		input.SourceType = "auto"
	}
	if input.AuthType == "" {
		input.AuthType = extension.AuthTypeNone
	}

	// Validate auth_type
	switch input.AuthType {
	case extension.AuthTypeNone, extension.AuthTypeGitHubPAT, extension.AuthTypeGitLabPAT, extension.AuthTypeSSHKey:
		// valid
	default:
		return nil, fmt.Errorf("%w: invalid auth_type %q", ErrInvalidInput, input.AuthType)
	}

	registry := &extension.SkillRegistry{
		OrganizationID: &orgID,
		RepositoryURL:  input.RepositoryURL,
		Branch:         input.Branch,
		SourceType:     input.SourceType,
		AuthType:       input.AuthType,
		SyncStatus:     "pending",
		IsActive:       true,
	}

	// Set compatible_agents as JSON
	if len(input.CompatibleAgents) > 0 {
		agentsJSON, err := marshalJSON(input.CompatibleAgents)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal compatible_agents: %w", err)
		}
		registry.CompatibleAgents = agentsJSON
	}
	// else: use DB default '["claude-code"]'

	// Encrypt credential if provided
	if input.AuthCredential != "" && input.AuthType != extension.AuthTypeNone {
		encrypted, err := s.encryptCredential(input.AuthCredential)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt auth credential: %w", err)
		}
		registry.AuthCredential = encrypted
	}

	if err := s.repo.CreateSkillRegistry(ctx, registry); err != nil {
		return nil, fmt.Errorf("failed to create skill registry: %w", err)
	}

	return registry, nil
}

// SyncSkillRegistry triggers a sync for a skill registry
func (s *Service) SyncSkillRegistry(ctx context.Context, orgID, sourceID int64) (*extension.SkillRegistry, error) {
	registry, err := s.repo.GetSkillRegistry(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("%w: skill registry %d", ErrNotFound, sourceID)
	}

	// Platform-level registries (OrganizationID == nil) cannot be synced by org users
	if registry.OrganizationID == nil {
		return nil, fmt.Errorf("%w: cannot sync platform-level registry", ErrForbidden)
	}

	// Validate org ownership
	if *registry.OrganizationID != orgID {
		return nil, fmt.Errorf("%w: skill registry does not belong to this organization", ErrForbidden)
	}

	if s.importer == nil {
		return nil, fmt.Errorf("skill importer not available")
	}

	// Trigger sync — runs synchronously so caller gets final status
	if err := s.importer.SyncSource(ctx, sourceID); err != nil {
		slog.Error("Skill registry sync failed", "registry_id", sourceID, "error", err)
		// Reload to get the error status set by importer
		registry, _ = s.repo.GetSkillRegistry(ctx, sourceID)
		if registry != nil {
			return registry, nil
		}
		return nil, fmt.Errorf("sync failed: %w", err)
	}

	// Reload to get updated status
	registry, err = s.repo.GetSkillRegistry(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to reload registry after sync: %w", err)
	}

	return registry, nil
}

// DeleteSkillRegistry deletes a skill registry
func (s *Service) DeleteSkillRegistry(ctx context.Context, orgID, sourceID int64) error {
	registry, err := s.repo.GetSkillRegistry(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("%w: skill registry %d", ErrNotFound, sourceID)
	}

	// Platform-level registries (OrganizationID == nil) cannot be deleted by org users
	if registry.OrganizationID == nil {
		return fmt.Errorf("%w: cannot delete platform-level registry", ErrForbidden)
	}

	// Validate org ownership
	if *registry.OrganizationID != orgID {
		return fmt.Errorf("%w: skill registry does not belong to this organization", ErrForbidden)
	}

	return s.repo.DeleteSkillRegistry(ctx, sourceID)
}

// --- Skill Registry Overrides ---

// TogglePlatformRegistry enables or disables a platform-level skill registry for an organization
func (s *Service) TogglePlatformRegistry(ctx context.Context, orgID int64, sourceID int64, disabled bool) error {
	registry, err := s.repo.GetSkillRegistry(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("%w: registry not found", ErrNotFound)
	}
	if !registry.IsPlatformLevel() {
		return fmt.Errorf("%w: can only toggle platform-level registries", ErrInvalidInput)
	}
	return s.repo.SetSkillRegistryOverride(ctx, orgID, sourceID, disabled)
}

// ListSkillRegistryOverrides returns all skill registry overrides for an organization
func (s *Service) ListSkillRegistryOverrides(ctx context.Context, orgID int64) ([]*extension.SkillRegistryOverride, error) {
	return s.repo.ListSkillRegistryOverrides(ctx, orgID)
}

// --- Marketplace Queries ---

// ListMarketSkills returns skills available in the marketplace
// Merges platform-level (org_id=NULL) + org-specific skill market items
func (s *Service) ListMarketSkills(ctx context.Context, orgID int64, query, category string) ([]*extension.SkillMarketItem, error) {
	return s.repo.ListSkillMarketItems(ctx, &orgID, query, category)
}

// ListMarketMcpServers returns MCP server templates from the marketplace with pagination
func (s *Service) ListMarketMcpServers(ctx context.Context, query, category string, limit, offset int) ([]*extension.McpMarketItem, int64, error) {
	return s.repo.ListMcpMarketItems(ctx, query, category, limit, offset)
}

// --- Repo Skills Installation ---

// ListRepoSkills lists installed skills for a repository.
// userID is used to filter user-scope items to only the current user's installations.
func (s *Service) ListRepoSkills(ctx context.Context, orgID, repoID, userID int64, scope string) ([]*extension.InstalledSkill, error) {
	if scope == "all" {
		scope = ""
	}
	return s.repo.ListInstalledSkills(ctx, orgID, repoID, userID, scope)
}

// InstallSkillFromMarket installs a skill from the marketplace
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
// It delegates to SkillPackager for cloning, packaging, and uploading,
// then persists the installed skill record.
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
// It delegates to SkillPackager for extraction, packaging, and uploading,
// then persists the installed skill record.
func (s *Service) InstallSkillFromUpload(ctx context.Context, orgID, repoID, userID int64, reader io.Reader, filename, scope string) (*extension.InstalledSkill, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}

	if s.packager == nil {
		return nil, fmt.Errorf("skill packager not configured")
	}

	return s.packager.CompleteUploadInstall(ctx, orgID, repoID, userID, reader, filename, scope)
}

// UpdateSkill updates an installed skill (enable/disable, pin version)
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

	// Org-scoped skills require admin/owner role to modify
	if skill.Scope == extension.ScopeOrg && userRole != "admin" && userRole != "owner" {
		return nil, fmt.Errorf("%w: admin permission required to modify org-scoped skill", ErrForbidden)
	}

	// User-scoped skills can only be modified by the user who installed them
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

// UninstallSkill removes an installed skill
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

// ListRepoMcpServers lists installed MCP servers for a repository.
// userID is used to filter user-scope items to only the current user's installations.
func (s *Service) ListRepoMcpServers(ctx context.Context, orgID, repoID, userID int64, scope string) ([]*extension.InstalledMcpServer, error) {
	if scope == "all" {
		scope = ""
	}
	return s.repo.ListInstalledMcpServers(ctx, orgID, repoID, userID, scope)
}

// InstallMcpFromMarket installs an MCP server from a marketplace template
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

// InstallCustomMcpServer installs a custom MCP server
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

// UpdateMcpServer updates an installed MCP server
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

	// Org-scoped MCP servers require admin/owner role to modify
	if server.Scope == extension.ScopeOrg && userRole != "admin" && userRole != "owner" {
		return nil, fmt.Errorf("%w: admin permission required to modify org-scoped MCP server", ErrForbidden)
	}

	// User-scoped MCP servers can only be modified by the user who installed them
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

// UninstallMcpServer removes an installed MCP server
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

	// Org-scoped MCP servers require admin/owner role to uninstall
	if server.Scope == extension.ScopeOrg && userRole != "admin" && userRole != "owner" {
		return fmt.Errorf("%w: admin permission required to uninstall org-scoped MCP server", ErrForbidden)
	}

	// User-scoped MCP servers can only be uninstalled by the user who installed them
	if server.Scope == extension.ScopeUser {
		if server.InstalledBy == nil || *server.InstalledBy != userID {
			return fmt.Errorf("%w: can only uninstall your own user-scoped MCP server", ErrForbidden)
		}
	}

	return s.repo.DeleteInstalledMcpServer(ctx, installID)
}

// --- ExtensionProvider interface (for ConfigBuilder) ---

// ResolvedSkill contains skill metadata + download info for Pod creation
type ResolvedSkill struct {
	Slug        string
	ContentSha  string
	DownloadURL string // presigned URL
	PackageSize int64
	TargetDir   string // relative path in plugin directory
}

// GetEffectiveMcpServers returns the merged MCP server list for a repo
// (org scope + current user's user scope, slug dedup with user priority)
// Filters out servers whose MarketItem has an agent_type_filter that does not include agentSlug.
func (s *Service) GetEffectiveMcpServers(ctx context.Context, orgID, userID, repoID int64, agentSlug string) ([]*extension.InstalledMcpServer, error) {
	servers, err := s.repo.GetEffectiveMcpServers(ctx, orgID, userID, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get effective MCP servers: %w", err)
	}

	// Filter by agent type compatibility
	servers = filterMcpServersByAgentType(servers, agentSlug)

	// Decrypt env vars for each server
	for _, srv := range servers {
		if err := s.decryptServerEnvVars(srv); err != nil {
			slog.Warn("Failed to decrypt env vars for MCP server",
				"slug", srv.Slug, "error", err)
		}
	}

	return servers, nil
}

// GetEffectiveSkills returns the merged skill list with presigned download URLs.
// Filters out skills whose MarketItem has an agent_type_filter that does not include agentSlug.
func (s *Service) GetEffectiveSkills(ctx context.Context, orgID, userID, repoID int64, agentSlug string) ([]*ResolvedSkill, error) {
	skills, err := s.repo.GetEffectiveSkills(ctx, orgID, userID, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get effective skills: %w", err)
	}

	// Filter by agent type compatibility
	skills = filterSkillsByAgentType(skills, agentSlug)

	resolved := make([]*ResolvedSkill, 0, len(skills))
	for _, skill := range skills {
		sha := skill.GetEffectiveSha()
		storageKey := skill.GetEffectiveStorageKey()
		packageSize := skill.GetEffectivePackageSize()

		if sha == "" || storageKey == "" {
			slog.Warn("Skill missing SHA or storage key, skipping",
				"slug", skill.Slug, "install_source", skill.InstallSource)
			continue
		}

		// Generate presigned download URL using internal endpoint.
		// Runner downloads happen within the Docker/internal network,
		// so we use GetInternalURL instead of GetURL (which uses public endpoint).
		downloadURL, err := s.storage.GetInternalURL(ctx, storageKey, presignedURLExpiry)
		if err != nil {
			slog.Error("Failed to generate presigned URL for skill",
				"slug", skill.Slug, "storage_key", storageKey, "error", err)
			continue
		}

		resolved = append(resolved, &ResolvedSkill{
			Slug:        skill.Slug,
			ContentSha:  sha,
			DownloadURL: downloadURL,
			PackageSize: packageSize,
			TargetDir:   fmt.Sprintf("skills/%s", skill.Slug),
		})
	}

	return resolved, nil
}

// --- Agent type filtering helpers ---

// filterMcpServersByAgentType removes MCP servers whose MarketItem has an
// agent_type_filter that does not include the given agentSlug.
// Servers without a MarketItem (custom installs) always pass through.
// An empty agentSlug disables filtering (all servers pass through).
// A MarketItem with an empty/null agent_type_filter means "all agents allowed".
func filterMcpServersByAgentType(servers []*extension.InstalledMcpServer, agentSlug string) []*extension.InstalledMcpServer {
	if agentSlug == "" {
		return servers
	}
	result := make([]*extension.InstalledMcpServer, 0, len(servers))
	for _, srv := range servers {
		if srv.MarketItem == nil {
			// Custom/non-market installs: no filter, always include
			result = append(result, srv)
			continue
		}
		filter := srv.MarketItem.GetAgentTypeFilter()
		if len(filter) == 0 {
			// No filter on market item: all agents allowed
			result = append(result, srv)
			continue
		}
		for _, allowed := range filter {
			if allowed == agentSlug {
				result = append(result, srv)
				break
			}
		}
	}
	return result
}

// filterSkillsByAgentType removes skills whose MarketItem has an
// agent_type_filter that does not include the given agentSlug.
// Skills without a MarketItem (github/upload installs) always pass through.
// An empty agentSlug disables filtering (all skills pass through).
// A MarketItem with an empty/null agent_type_filter means "all agents allowed".
func filterSkillsByAgentType(skills []*extension.InstalledSkill, agentSlug string) []*extension.InstalledSkill {
	if agentSlug == "" {
		return skills
	}
	result := make([]*extension.InstalledSkill, 0, len(skills))
	for _, skill := range skills {
		if skill.MarketItem == nil {
			// Non-market installs: no filter, always include
			result = append(result, skill)
			continue
		}
		filter := skill.MarketItem.GetAgentTypeFilter()
		if len(filter) == 0 {
			// No filter on market item: all agents allowed
			result = append(result, skill)
			continue
		}
		for _, allowed := range filter {
			if allowed == agentSlug {
				result = append(result, skill)
				break
			}
		}
	}
	return result
}

// --- Encryption helpers ---

// DecryptCredential decrypts a single credential string.
// Exported so it can be passed to SkillImporter.SetCredentialDecryptor.
func (s *Service) DecryptCredential(encrypted string) (string, error) {
	return s.decryptCredential(encrypted)
}

// encryptCredential encrypts a single credential string
func (s *Service) encryptCredential(credential string) (string, error) {
	if s.crypto == nil {
		// No encryption configured (development mode), store as-is
		return credential, nil
	}
	return s.crypto.Encrypt(credential)
}

// decryptCredential decrypts a single credential string
func (s *Service) decryptCredential(encrypted string) (string, error) {
	if s.crypto == nil || encrypted == "" {
		return encrypted, nil
	}
	decrypted, err := s.crypto.Decrypt(encrypted)
	if err != nil {
		// Log warning — silently returning ciphertext could leak encrypted values
		slog.Warn("Failed to decrypt credential, value may be unencrypted or corrupted", "error", err)
		return encrypted, nil
	}
	return decrypted, nil
}

func (s *Service) encryptEnvVars(vars map[string]string) ([]byte, error) {
	if s.crypto == nil {
		// No encryption configured, store as-is (development mode)
		return marshalJSON(vars)
	}
	// Encrypt each sensitive value
	encrypted := make(map[string]string, len(vars))
	for k, v := range vars {
		enc, err := s.crypto.Encrypt(v)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt env var %s: %w", k, err)
		}
		encrypted[k] = enc
	}
	return marshalJSON(encrypted)
}

func (s *Service) decryptServerEnvVars(server *extension.InstalledMcpServer) error {
	if s.crypto == nil || len(server.EnvVars) == 0 {
		return nil
	}

	var encrypted map[string]string
	if err := unmarshalJSON(server.EnvVars, &encrypted); err != nil {
		return err
	}

	decrypted := make(map[string]string, len(encrypted))
	for k, v := range encrypted {
		dec, err := s.crypto.Decrypt(v)
		if err != nil {
			// Log warning — silently keeping ciphertext could leak encrypted values
			slog.Warn("Failed to decrypt env var, value may be unencrypted or corrupted",
				"key", k, "error", err)
			decrypted[k] = v
			continue
		}
		decrypted[k] = dec
	}

	data, err := marshalJSON(decrypted)
	if err != nil {
		return fmt.Errorf("failed to marshal decrypted env vars: %w", err)
	}
	server.EnvVars = data
	return nil
}
