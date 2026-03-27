package infra

import (
	"context"
	"time"

	"gorm.io/gorm/clause"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

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

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

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
	if item.RegistryName != "" {
		var existing extension.McpMarketItem
		err := r.db.WithContext(ctx).Where("registry_name = ?", item.RegistryName).First(&existing).Error
		if err == nil {
			item.ID = existing.ID
			item.CreatedAt = existing.CreatedAt
			return r.db.WithContext(ctx).Save(item).Error
		}
	}
	var existingBySlug extension.McpMarketItem
	err := r.db.WithContext(ctx).Where("slug = ?", item.Slug).First(&existingBySlug).Error
	if err == nil {
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

	for _, item := range items {
		item.UpdatedAt = now
		if item.CreatedAt.IsZero() {
			item.CreatedAt = now
		}
	}

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
