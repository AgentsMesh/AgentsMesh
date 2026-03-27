package infra

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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

func (r *extensionRepo) BatchUpsertMcpMarketItems(ctx context.Context, items []*extension.McpMarketItem) error {
	if len(items) == 0 {
		return nil
	}

	const batchSize = 100
	now := time.Now()

	// Pre-process: handle slug conflicts with seed data and set timestamps
	for _, item := range items {
		item.UpdatedAt = now
		if item.CreatedAt.IsZero() {
			item.CreatedAt = now
		}
	}

	// Use GORM's OnConflict clause targeting the partial unique index on registry_name.
	// On conflict, update all mutable fields while preserving created_at.
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "registry_name"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"name", "description", "icon", "transport_type", "command",
				"default_args", "default_http_url", "default_http_headers",
				"env_var_schema", "agent_filter", "category", "is_active",
				"version", "repository_url", "registry_meta", "last_synced_at",
				"updated_at",
			}),
		}).
		CreateInBatches(items, batchSize).Error
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
