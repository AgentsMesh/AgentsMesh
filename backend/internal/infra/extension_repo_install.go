package infra

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// --- Installed MCP Servers ---

func (r *extensionRepo) ListInstalledMcpServers(ctx context.Context, orgID, repoID, userID int64, scope string) ([]*extension.InstalledMcpServer, error) {
	var servers []*extension.InstalledMcpServer
	query := r.db.WithContext(ctx).
		Where("organization_id = ? AND repository_id = ?", orgID, repoID).
		Preload("MarketItem")

	if scope != "" {
		query = query.Where("scope = ?", scope)
	}

	if scope == "" || scope == extension.ScopeUser {
		query = query.Where("(scope = ? OR (scope = ? AND installed_by = ?))",
			extension.ScopeOrg, extension.ScopeUser, userID)
	}

	if err := query.Order("created_at DESC").Find(&servers).Error; err != nil {
		return nil, err
	}
	return servers, nil
}

func (r *extensionRepo) GetInstalledMcpServer(ctx context.Context, id int64) (*extension.InstalledMcpServer, error) {
	var server extension.InstalledMcpServer
	if err := r.db.WithContext(ctx).Preload("MarketItem").First(&server, id).Error; err != nil {
		return nil, err
	}
	return &server, nil
}

func (r *extensionRepo) CreateInstalledMcpServer(ctx context.Context, server *extension.InstalledMcpServer) error {
	if err := r.db.WithContext(ctx).Create(server).Error; err != nil {
		if isDuplicateKeyError(err) {
			return fmt.Errorf("%w: MCP server '%s'", extension.ErrDuplicateInstall, server.Slug)
		}
		return err
	}
	return nil
}

func (r *extensionRepo) UpdateInstalledMcpServer(ctx context.Context, server *extension.InstalledMcpServer) error {
	return r.db.WithContext(ctx).Save(server).Error
}

func (r *extensionRepo) DeleteInstalledMcpServer(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&extension.InstalledMcpServer{}, id).Error
}

func (r *extensionRepo) GetEffectiveMcpServers(ctx context.Context, orgID, userID, repoID int64) ([]*extension.InstalledMcpServer, error) {
	var servers []*extension.InstalledMcpServer

	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND repository_id = ? AND is_enabled = ?", orgID, repoID, true).
		Where("(scope = ? OR (scope = ? AND installed_by = ?))", extension.ScopeOrg, extension.ScopeUser, userID).
		Preload("MarketItem").
		Order("scope ASC, created_at ASC").
		Find(&servers).Error; err != nil {
		return nil, err
	}

	return deduplicateMcpServers(servers), nil
}

func deduplicateMcpServers(servers []*extension.InstalledMcpServer) []*extension.InstalledMcpServer {
	seen := make(map[string]*extension.InstalledMcpServer, len(servers))
	for _, s := range servers {
		existing, exists := seen[s.Slug]
		if !exists {
			seen[s.Slug] = s
			continue
		}
		if s.Scope == extension.ScopeUser && existing.Scope == extension.ScopeOrg {
			seen[s.Slug] = s
		}
	}
	result := make([]*extension.InstalledMcpServer, 0, len(seen))
	for _, s := range seen {
		result = append(result, s)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Slug < result[j].Slug
	})
	return result
}

// --- Installed Skills ---

func (r *extensionRepo) ListInstalledSkills(ctx context.Context, orgID, repoID, userID int64, scope string) ([]*extension.InstalledSkill, error) {
	var skills []*extension.InstalledSkill
	query := r.db.WithContext(ctx).
		Where("organization_id = ? AND repository_id = ?", orgID, repoID).
		Preload("MarketItem").
		Preload("MarketItem.Registry")

	if scope != "" {
		query = query.Where("scope = ?", scope)
	}

	if scope == "" || scope == extension.ScopeUser {
		query = query.Where("(scope = ? OR (scope = ? AND installed_by = ?))",
			extension.ScopeOrg, extension.ScopeUser, userID)
	}

	if err := query.Order("created_at DESC").Find(&skills).Error; err != nil {
		return nil, err
	}
	return skills, nil
}

func (r *extensionRepo) GetInstalledSkill(ctx context.Context, id int64) (*extension.InstalledSkill, error) {
	var skill extension.InstalledSkill
	if err := r.db.WithContext(ctx).Preload("MarketItem").Preload("MarketItem.Registry").First(&skill, id).Error; err != nil {
		return nil, err
	}
	return &skill, nil
}

func (r *extensionRepo) CreateInstalledSkill(ctx context.Context, skill *extension.InstalledSkill) error {
	if err := r.db.WithContext(ctx).Create(skill).Error; err != nil {
		if isDuplicateKeyError(err) {
			return fmt.Errorf("%w: skill '%s'", extension.ErrDuplicateInstall, skill.Slug)
		}
		return err
	}
	return nil
}

func (r *extensionRepo) UpdateInstalledSkill(ctx context.Context, skill *extension.InstalledSkill) error {
	return r.db.WithContext(ctx).Save(skill).Error
}

func (r *extensionRepo) DeleteInstalledSkill(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&extension.InstalledSkill{}, id).Error
}

func (r *extensionRepo) GetEffectiveSkills(ctx context.Context, orgID, userID, repoID int64) ([]*extension.InstalledSkill, error) {
	var skills []*extension.InstalledSkill

	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND repository_id = ? AND is_enabled = ?", orgID, repoID, true).
		Where("(scope = ? OR (scope = ? AND installed_by = ?))", extension.ScopeOrg, extension.ScopeUser, userID).
		Preload("MarketItem").
		Preload("MarketItem.Registry").
		Order("scope ASC, created_at ASC").
		Find(&skills).Error; err != nil {
		return nil, err
	}

	return deduplicateSkills(skills), nil
}

func deduplicateSkills(skills []*extension.InstalledSkill) []*extension.InstalledSkill {
	seen := make(map[string]*extension.InstalledSkill, len(skills))
	for _, s := range skills {
		existing, exists := seen[s.Slug]
		if !exists {
			seen[s.Slug] = s
			continue
		}
		if s.Scope == extension.ScopeUser && existing.Scope == extension.ScopeOrg {
			seen[s.Slug] = s
		}
	}
	result := make([]*extension.InstalledSkill, 0, len(seen))
	for _, s := range seen {
		result = append(result, s)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Slug < result[j].Slug
	})
	return result
}

// --- Skill Registry Overrides ---

func (r *extensionRepo) SetSkillRegistryOverride(ctx context.Context, orgID int64, registryID int64, isDisabled bool) error {
	override := &extension.SkillRegistryOverride{
		OrganizationID: orgID,
		RegistryID:     registryID,
		IsDisabled:     isDisabled,
	}
	return r.db.WithContext(ctx).
		Where("organization_id = ? AND registry_id = ?", orgID, registryID).
		Assign(map[string]interface{}{"is_disabled": isDisabled, "updated_at": time.Now()}).
		FirstOrCreate(override).Error
}

func (r *extensionRepo) ListSkillRegistryOverrides(ctx context.Context, orgID int64) ([]*extension.SkillRegistryOverride, error) {
	var overrides []*extension.SkillRegistryOverride
	err := r.db.WithContext(ctx).Where("organization_id = ?", orgID).Find(&overrides).Error
	return overrides, err
}

// Compile-time interface compliance check.
var _ extension.Repository = (*extensionRepo)(nil)
