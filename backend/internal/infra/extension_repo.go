package infra

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// isDuplicateKeyError checks whether the given error is a database unique constraint violation.
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "duplicate key value") || // PostgreSQL
		strings.Contains(errStr, "UNIQUE constraint failed") || // SQLite
		strings.Contains(errStr, "Duplicate entry") // MySQL
}

// escapeLike escapes special LIKE/ILIKE characters (%, _, \) in a search string.
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

// attachSkillCounts populates the derived SkillCount field on each registry
// by counting active rows in skill_market_items grouped by registry_id.
// One batched query regardless of slice size; absent rows imply count=0.
func (r *extensionRepo) attachSkillCounts(ctx context.Context, registries []*extension.SkillRegistry) error {
	if len(registries) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(registries))
	for _, reg := range registries {
		ids = append(ids, reg.ID)
	}

	type countRow struct {
		RegistryID int64 `gorm:"column:registry_id"`
		Count      int   `gorm:"column:count"`
	}
	var rows []countRow
	if err := r.db.WithContext(ctx).
		Table("skill_market_items").
		Select("registry_id, COUNT(*) AS count").
		Where("registry_id IN ? AND is_active = ?", ids, true).
		Group("registry_id").
		Scan(&rows).Error; err != nil {
		return err
	}

	counts := make(map[int64]int, len(rows))
	for _, row := range rows {
		counts[row.RegistryID] = row.Count
	}
	for _, reg := range registries {
		reg.SkillCount = counts[reg.ID]
	}
	return nil
}

func (r *extensionRepo) ListSkillRegistries(ctx context.Context, orgID *int64) ([]*extension.SkillRegistry, error) {
	var registries []*extension.SkillRegistry
	query := r.db.WithContext(ctx)
	if orgID == nil {
		query = query.Where("organization_id IS NULL")
	} else {
		query = query.Where("organization_id IS NULL OR organization_id = ?", *orgID)
	}
	if err := query.Order("organization_id ASC NULLS FIRST, created_at DESC").Find(&registries).Error; err != nil {
		return nil, err
	}
	if err := r.attachSkillCounts(ctx, registries); err != nil {
		return nil, err
	}
	return registries, nil
}

func (r *extensionRepo) ListAllActiveSkillRegistries(ctx context.Context) ([]*extension.SkillRegistry, error) {
	var registries []*extension.SkillRegistry
	if err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("organization_id ASC NULLS FIRST, created_at DESC").
		Find(&registries).Error; err != nil {
		return nil, err
	}
	if err := r.attachSkillCounts(ctx, registries); err != nil {
		return nil, err
	}
	return registries, nil
}

func (r *extensionRepo) GetSkillRegistry(ctx context.Context, id int64) (*extension.SkillRegistry, error) {
	var registry extension.SkillRegistry
	if err := r.db.WithContext(ctx).First(&registry, id).Error; err != nil {
		return nil, err
	}
	if err := r.attachSkillCounts(ctx, []*extension.SkillRegistry{&registry}); err != nil {
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
	if err := r.attachSkillCounts(ctx, []*extension.SkillRegistry{&registry}); err != nil {
		return nil, err
	}
	return &registry, nil
}

// ClaimSyncLock implements atomic CAS: an UPDATE that only succeeds if the
// row is not already syncing (or is stale). One SELECT precedes the UPDATE
// to surface the wasStale signal for ops logging — the SELECT result has no
// effect on the CAS itself, which is the authoritative gate.
func (r *extensionRepo) ClaimSyncLock(ctx context.Context, registryID int64, staleAfter time.Duration) (bool, bool, error) {
	var existing extension.SkillRegistry
	if err := r.db.WithContext(ctx).Select("sync_status", "updated_at").
		First(&existing, registryID).Error; err != nil {
		return false, false, err
	}
	wasStale := existing.SyncStatus == extension.SyncStatusSyncing &&
		time.Since(existing.UpdatedAt) >= staleAfter

	staleBefore := time.Now().Add(-staleAfter)
	result := r.db.WithContext(ctx).
		Model(&extension.SkillRegistry{}).
		Where("id = ?", registryID).
		Where("sync_status <> ? OR updated_at < ?", extension.SyncStatusSyncing, staleBefore).
		Updates(map[string]any{
			"sync_status": extension.SyncStatusSyncing,
			"sync_error":  "",
		})
	if result.Error != nil {
		return false, false, result.Error
	}
	return result.RowsAffected == 1, wasStale, nil
}

// --- Skill Market Items ---

func (r *extensionRepo) ListSkillMarketItems(ctx context.Context, orgID *int64, queryStr string, category string) ([]*extension.SkillMarketItem, error) {
	var items []*extension.SkillMarketItem

	query := r.db.WithContext(ctx).
		Joins("JOIN skill_registries ON skill_registries.id = skill_market_items.registry_id").
		Where("skill_market_items.is_active = ?", true).
		Preload("Registry")

	if orgID == nil {
		query = query.Where("skill_registries.organization_id IS NULL")
	} else {
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
