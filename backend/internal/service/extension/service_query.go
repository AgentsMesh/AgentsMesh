package extension

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// --- Marketplace Queries ---

// ListMarketSkills returns marketplace skills, merging platform-level + org-specific items.
func (s *Service) ListMarketSkills(ctx context.Context, orgID int64, query, category string) ([]*extension.SkillMarketItem, error) {
	return s.repo.ListSkillMarketItems(ctx, &orgID, query, category)
}

func (s *Service) ListMarketMcpServers(ctx context.Context, query, category string, limit, offset int) ([]*extension.McpMarketItem, int64, error) {
	return s.repo.ListMcpMarketItems(ctx, query, category, limit, offset)
}

// --- Repo Skills / MCP listing ---

// ListRepoSkills lists installed skills for a repository, filtering user-scope items to current user.
func (s *Service) ListRepoSkills(ctx context.Context, orgID, repoID, userID int64, scope string) ([]*extension.InstalledSkill, error) {
	if scope == "all" {
		scope = ""
	}
	return s.repo.ListInstalledSkills(ctx, orgID, repoID, userID, scope)
}

// ListRepoMcpServers lists installed MCP servers for a repository, filtering user-scope to current user.
func (s *Service) ListRepoMcpServers(ctx context.Context, orgID, repoID, userID int64, scope string) ([]*extension.InstalledMcpServer, error) {
	if scope == "all" {
		scope = ""
	}
	return s.repo.ListInstalledMcpServers(ctx, orgID, repoID, userID, scope)
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
// Filters out servers whose MarketItem has an agent_filter that does not include agentSlug.
func (s *Service) GetEffectiveMcpServers(ctx context.Context, orgID, userID, repoID int64, agentSlug string) ([]*extension.InstalledMcpServer, error) {
	servers, err := s.repo.GetEffectiveMcpServers(ctx, orgID, userID, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get effective MCP servers: %w", err)
	}

	// Filter by agent compatibility
	servers = filterMcpServersByAgent(servers, agentSlug)

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
// Filters out skills whose MarketItem has an agent_filter that does not include agentSlug.
func (s *Service) GetEffectiveSkills(ctx context.Context, orgID, userID, repoID int64, agentSlug string) ([]*ResolvedSkill, error) {
	skills, err := s.repo.GetEffectiveSkills(ctx, orgID, userID, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get effective skills: %w", err)
	}

	// Filter by agent compatibility
	skills = filterSkillsByAgent(skills, agentSlug)

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

// --- Agent filtering helpers ---

// filterMcpServersByAgent removes MCP servers whose MarketItem has an
// agent_filter that does not include the given agentSlug.
// Servers without a MarketItem (custom installs) always pass through.
// An empty agentSlug disables filtering (all servers pass through).
// A MarketItem with an empty/null agent_filter means "all agents allowed".
func filterMcpServersByAgent(servers []*extension.InstalledMcpServer, agentSlug string) []*extension.InstalledMcpServer {
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
		filter := srv.MarketItem.GetAgentFilter()
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

// filterSkillsByAgent removes skills whose MarketItem has an
// agent_filter that does not include the given agentSlug.
// Skills without a MarketItem (github/upload installs) always pass through.
// An empty agentSlug disables filtering (all skills pass through).
// A MarketItem with an empty/null agent_filter means "all agents allowed".
func filterSkillsByAgent(skills []*extension.InstalledSkill, agentSlug string) []*extension.InstalledSkill {
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
		filter := skill.MarketItem.GetAgentFilter()
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
