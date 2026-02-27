package infra

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// isDuplicateKeyError checks whether the given error is a database unique constraint violation.
// Supports PostgreSQL, SQLite, and MySQL error messages.
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "duplicate key value") || // PostgreSQL
		strings.Contains(errStr, "UNIQUE constraint failed") || // SQLite
		strings.Contains(errStr, "Duplicate entry") // MySQL
}

// escapeLike escapes special LIKE/ILIKE characters (%, _, \) in a search string
// so they are treated as literal characters in PostgreSQL.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// extensionRepo implements extension.Repository using GORM
type extensionRepo struct {
	db *gorm.DB
}

// NewExtensionRepository creates a new extension repository
func NewExtensionRepository(db *gorm.DB) extension.Repository {
	return &extensionRepo{db: db}
}

// --- Skill Registries ---

func (r *extensionRepo) ListSkillRegistries(ctx context.Context, orgID *int64) ([]*extension.SkillRegistry, error) {
	var registries []*extension.SkillRegistry
	query := r.db.WithContext(ctx)
	if orgID == nil {
		// Admin: platform-level only
		query = query.Where("organization_id IS NULL")
	} else {
		// Org user: merge platform-level + org-specific (same as ListSkillMarketItems)
		query = query.Where("organization_id IS NULL OR organization_id = ?", *orgID)
	}
	if err := query.Order("organization_id ASC NULLS FIRST, created_at DESC").Find(&registries).Error; err != nil {
		return nil, err
	}
	return registries, nil
}

func (r *extensionRepo) GetSkillRegistry(ctx context.Context, id int64) (*extension.SkillRegistry, error) {
	var registry extension.SkillRegistry
	if err := r.db.WithContext(ctx).First(&registry, id).Error; err != nil {
		return nil, err
	}
	return &registry, nil
}

func (r *extensionRepo) CreateSkillRegistry(ctx context.Context, registry *extension.SkillRegistry) error {
	if err := r.db.WithContext(ctx).Create(registry).Error; err != nil {
		if isDuplicateKeyError(err) {
			return fmt.Errorf("%w: skill registry with URL '%s'", extension.ErrDuplicateInstall, registry.RepositoryURL)
		}
		return err
	}
	return nil
}

func (r *extensionRepo) UpdateSkillRegistry(ctx context.Context, registry *extension.SkillRegistry) error {
	return r.db.WithContext(ctx).Save(registry).Error
}

func (r *extensionRepo) DeleteSkillRegistry(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&extension.SkillRegistry{}, id).Error
}

func (r *extensionRepo) FindSkillRegistryByURL(ctx context.Context, orgID *int64, repoURL string) (*extension.SkillRegistry, error) {
	var registry extension.SkillRegistry
	query := r.db.WithContext(ctx).Where("repository_url = ?", repoURL)
	if orgID == nil {
		query = query.Where("organization_id IS NULL")
	} else {
		query = query.Where("organization_id = ?", *orgID)
	}
	if err := query.First(&registry).Error; err != nil {
		return nil, err
	}
	return &registry, nil
}

// --- Skill Market Items ---

func (r *extensionRepo) ListSkillMarketItems(ctx context.Context, orgID *int64, queryStr string, category string) ([]*extension.SkillMarketItem, error) {
	var items []*extension.SkillMarketItem

	query := r.db.WithContext(ctx).
		Joins("JOIN skill_registries ON skill_registries.id = skill_market_items.registry_id").
		Where("skill_market_items.is_active = ?", true).
		Preload("Registry")

	if orgID == nil {
		// Platform-level only
		query = query.Where("skill_registries.organization_id IS NULL")
	} else {
		// Merge platform-level + org-specific, excluding disabled platform sources
		query = query.
			Joins("LEFT JOIN skill_registry_overrides sso ON sso.registry_id = skill_registries.id AND sso.organization_id = ?", *orgID).
			Where("(skill_registries.organization_id IS NULL AND (sso.id IS NULL OR sso.is_disabled = false)) OR skill_registries.organization_id = ?", *orgID)
	}

	if queryStr != "" {
		search := "%" + escapeLike(queryStr) + "%"
		query = query.Where(
			"skill_market_items.slug ILIKE ? OR skill_market_items.display_name ILIKE ? OR skill_market_items.description ILIKE ?",
			search, search, search,
		)
	}

	if category != "" {
		query = query.Where("skill_market_items.category = ?", category)
	}

	if err := query.Order("skill_market_items.display_name ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *extensionRepo) GetSkillMarketItem(ctx context.Context, id int64) (*extension.SkillMarketItem, error) {
	var item extension.SkillMarketItem
	if err := r.db.WithContext(ctx).Preload("Registry").First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *extensionRepo) FindSkillMarketItemBySlug(ctx context.Context, registryID int64, slug string) (*extension.SkillMarketItem, error) {
	var item extension.SkillMarketItem
	if err := r.db.WithContext(ctx).
		Where("registry_id = ? AND slug = ?", registryID, slug).
		First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *extensionRepo) CreateSkillMarketItem(ctx context.Context, item *extension.SkillMarketItem) error {
	if err := r.db.WithContext(ctx).Create(item).Error; err != nil {
		if isDuplicateKeyError(err) {
			return fmt.Errorf("%w: skill market item '%s'", extension.ErrDuplicateInstall, item.Slug)
		}
		return err
	}
	return nil
}

func (r *extensionRepo) UpdateSkillMarketItem(ctx context.Context, item *extension.SkillMarketItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *extensionRepo) DeactivateSkillMarketItemsNotIn(ctx context.Context, registryID int64, slugs []string) error {
	query := r.db.WithContext(ctx).
		Model(&extension.SkillMarketItem{}).
		Where("registry_id = ?", registryID)

	if len(slugs) > 0 {
		query = query.Where("slug NOT IN ?", slugs)
	}

	return query.Update("is_active", false).Error
}

// --- MCP Market Items ---

func (r *extensionRepo) ListMcpMarketItems(ctx context.Context, queryStr string, category string, limit, offset int) ([]*extension.McpMarketItem, int64, error) {
	var items []*extension.McpMarketItem
	var total int64

	query := r.db.WithContext(ctx).Model(&extension.McpMarketItem{}).Where("is_active = ?", true)

	if queryStr != "" {
		search := "%" + escapeLike(queryStr) + "%"
		query = query.Where(
			"slug ILIKE ? OR name ILIKE ? OR description ILIKE ?",
			search, search, search,
		)
	}

	if category != "" {
		query = query.Where("category = ?", category)
	}

	// Count total matching items
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination defaults
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	if err := query.Order("name ASC").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *extensionRepo) GetMcpMarketItem(ctx context.Context, id int64) (*extension.McpMarketItem, error) {
	var item extension.McpMarketItem
	if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *extensionRepo) FindMcpMarketItemByRegistryName(ctx context.Context, registryName string) (*extension.McpMarketItem, error) {
	var item extension.McpMarketItem
	if err := r.db.WithContext(ctx).Where("registry_name = ?", registryName).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *extensionRepo) UpsertMcpMarketItem(ctx context.Context, item *extension.McpMarketItem) error {
	// Try to find existing record by registry_name
	if item.RegistryName != "" {
		var existing extension.McpMarketItem
		err := r.db.WithContext(ctx).Where("registry_name = ?", item.RegistryName).First(&existing).Error
		if err == nil {
			// Update existing: preserve ID and creation time
			item.ID = existing.ID
			item.CreatedAt = existing.CreatedAt
			return r.db.WithContext(ctx).Save(item).Error
		}
	}
	// Also check for slug conflict (from seed data)
	var existingBySlug extension.McpMarketItem
	err := r.db.WithContext(ctx).Where("slug = ?", item.Slug).First(&existingBySlug).Error
	if err == nil {
		// Slug already taken (probably by seed data), append a suffix
		item.Slug = item.Slug + "-registry"
	}
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *extensionRepo) DeactivateMcpMarketItemsNotIn(ctx context.Context, sourceType string, registryNames []string) (int64, error) {
	query := r.db.WithContext(ctx).
		Model(&extension.McpMarketItem{}).
		Where("source = ? AND is_active = ?", sourceType, true)
	if len(registryNames) > 0 {
		query = query.Where("registry_name NOT IN ?", registryNames)
	}
	result := query.Update("is_active", false)
	return result.RowsAffected, result.Error
}

// --- Installed MCP Servers ---

func (r *extensionRepo) ListInstalledMcpServers(ctx context.Context, orgID, repoID, userID int64, scope string) ([]*extension.InstalledMcpServer, error) {
	var servers []*extension.InstalledMcpServer
	query := r.db.WithContext(ctx).
		Where("organization_id = ? AND repository_id = ?", orgID, repoID).
		Preload("MarketItem")

	if scope != "" {
		query = query.Where("scope = ?", scope)
	}

	// Filter user-scope items to only the current user's installations
	// to prevent data leakage across users
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

	// Fetch all enabled servers for this org+repo that are either:
	// - org-scope, OR
	// - user-scope installed by this user
	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND repository_id = ? AND is_enabled = ?", orgID, repoID, true).
		Where("(scope = ? OR (scope = ? AND installed_by = ?))", extension.ScopeOrg, extension.ScopeUser, userID).
		Preload("MarketItem").
		Order("scope ASC, created_at ASC"). // org first, then user
		Find(&servers).Error; err != nil {
		return nil, err
	}

	// Deduplicate by slug: user-scope overrides org-scope
	seen := make(map[string]*extension.InstalledMcpServer, len(servers))
	for _, s := range servers {
		existing, exists := seen[s.Slug]
		if !exists {
			seen[s.Slug] = s
			continue
		}
		// User scope overrides org scope
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
	return result, nil
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

	// Filter user-scope items to only the current user's installations
	// to prevent data leakage across users
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

	// Fetch all enabled skills for this org+repo that are either:
	// - org-scope, OR
	// - user-scope installed by this user
	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND repository_id = ? AND is_enabled = ?", orgID, repoID, true).
		Where("(scope = ? OR (scope = ? AND installed_by = ?))", extension.ScopeOrg, extension.ScopeUser, userID).
		Preload("MarketItem").
		Preload("MarketItem.Registry").
		Order("scope ASC, created_at ASC"). // org first, then user
		Find(&skills).Error; err != nil {
		return nil, err
	}

	// Deduplicate by slug: user-scope overrides org-scope
	seen := make(map[string]*extension.InstalledSkill, len(skills))
	for _, s := range skills {
		existing, exists := seen[s.Slug]
		if !exists {
			seen[s.Slug] = s
			continue
		}
		// User scope overrides org scope
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
	return result, nil
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

// Compile-time interface compliance check
var _ extension.Repository = (*extensionRepo)(nil)
