package extension

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

func (s *Service) ListMarketSkills(ctx context.Context, orgID int64, query, category string) ([]*extension.SkillMarketItem, error) {
	return s.repo.ListSkillMarketItems(ctx, &orgID, query, category)
}

func (s *Service) ListMarketMcpServers(ctx context.Context, query, category string, limit, offset int) ([]*extension.McpMarketItem, int64, error) {
	return s.repo.ListMcpMarketItems(ctx, query, category, limit, offset)
}

func (s *Service) ListRepoSkills(ctx context.Context, orgID, repoID, userID int64, scope string) ([]*extension.InstalledSkill, error) {
	if scope == "all" {
		scope = ""
	}
	return s.repo.ListInstalledSkills(ctx, orgID, repoID, userID, scope)
}

func (s *Service) ListRepoMcpServers(ctx context.Context, orgID, repoID, userID int64, scope string) ([]*extension.InstalledMcpServer, error) {
	if scope == "all" {
		scope = ""
	}
	return s.repo.ListInstalledMcpServers(ctx, orgID, repoID, userID, scope)
}

type ResolvedSkill struct {
	Slug        string
	ContentSha  string
	DownloadURL string // presigned URL
	PackageSize int64
	TargetDir   string // relative path in plugin directory
}

func (s *Service) GetEffectiveMcpServers(ctx context.Context, orgID, userID, repoID int64, agentSlug string) ([]*extension.InstalledMcpServer, error) {
	servers, err := s.repo.GetEffectiveMcpServers(ctx, orgID, userID, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get effective MCP servers: %w", err)
	}

	servers = filterMcpServersByAgent(servers, agentSlug)

	for _, srv := range servers {
		if err := s.decryptServerEnvVars(srv); err != nil {
			slog.WarnContext(ctx, "Failed to decrypt env vars for MCP server",
				"slug", srv.Slug, "error", err)
		}
	}

	return servers, nil
}

func (s *Service) GetEffectiveSkills(ctx context.Context, orgID, userID, repoID int64, agentSlug string) ([]*ResolvedSkill, error) {
	skills, err := s.repo.GetEffectiveSkills(ctx, orgID, userID, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get effective skills: %w", err)
	}

	skills = filterSkillsByAgent(skills, agentSlug)

	resolved := make([]*ResolvedSkill, 0, len(skills))
	for _, skill := range skills {
		sha := skill.GetEffectiveSha()
		storageKey := skill.GetEffectiveStorageKey()
		packageSize := skill.GetEffectivePackageSize()

		if sha == "" || storageKey == "" {
			slog.WarnContext(ctx, "Skill missing SHA or storage key, skipping",
				"slug", skill.Slug, "install_source", skill.InstallSource)
			continue
		}

		downloadURL, err := s.storage.GetInternalURL(ctx, storageKey, presignedURLExpiry)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to generate presigned URL for skill",
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

func filterMcpServersByAgent(servers []*extension.InstalledMcpServer, agentSlug string) []*extension.InstalledMcpServer {
	if agentSlug == "" {
		return servers
	}
	result := make([]*extension.InstalledMcpServer, 0, len(servers))
	for _, srv := range servers {
		if srv.MarketItem == nil {
			result = append(result, srv)
			continue
		}
		filter := srv.MarketItem.GetAgentFilter()
		if len(filter) == 0 {
			result = append(result, srv)
			continue
		}
		for _, allowed := range filter {
			if agentSlugMatches(allowed, agentSlug) {
				result = append(result, srv)
				break
			}
		}
	}
	return result
}

func filterSkillsByAgent(skills []*extension.InstalledSkill, agentSlug string) []*extension.InstalledSkill {
	if agentSlug == "" {
		return skills
	}
	result := make([]*extension.InstalledSkill, 0, len(skills))
	for _, skill := range skills {
		if skill.MarketItem == nil {
			result = append(result, skill)
			continue
		}
		filter := skill.MarketItem.GetAgentFilter()
		if len(filter) == 0 {
			result = append(result, skill)
			continue
		}
		for _, allowed := range filter {
			if agentSlugMatches(allowed, agentSlug) {
				result = append(result, skill)
				break
			}
		}
	}
	return result
}

func agentSlugMatches(filterValue, actual string) bool {
	if filterValue == actual {
		return true
	}
	aliases := map[string][]string{
		"codex-cli":   {"codex"},
		"codex":       {"codex-cli"},
		"claude-code": {"claude"},
		"claude":      {"claude-code"},
		"gemini-cli":  {"gemini"},
		"gemini":      {"gemini-cli"},
	}
	for _, alias := range aliases[actual] {
		if filterValue == alias {
			return true
		}
	}
	return false
}
