package blockstoreinfra

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (r *Repository) GetBlock(ctx context.Context, id uuid.UUID) (*blockstore.Block, error) {
	var b blockstore.Block
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&b).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, blockstore.ErrBlockNotFound
		}
		return nil, err
	}
	return &b, nil
}

func (r *Repository) ListBlocks(ctx context.Context, f blockstore.BlockFilter) ([]*blockstore.Block, int64, error) {
	q := r.db.WithContext(ctx).Model(&blockstore.Block{}).Where("workspace_id = ?", f.WorkspaceID)
	if !f.IncludeDeleted {
		q = q.Where("deleted_at IS NULL")
	}
	if f.Type != nil {
		q = q.Where("type = ?", *f.Type)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if f.Limit > 0 {
		q = q.Limit(f.Limit)
	}
	if f.Offset > 0 {
		q = q.Offset(f.Offset)
	}
	q = q.Order("updated_at DESC")
	var out []*blockstore.Block
	if err := q.Find(&out).Error; err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *Repository) ListChildren(ctx context.Context, parentID uuid.UUID, rel string) ([]*blockstore.Block, []*blockstore.BlockRef, error) {
	var refs []*blockstore.BlockRef
	err := r.db.WithContext(ctx).
		Where("from_id = ? AND rel = ?", parentID, rel).
		Order("order_key ASC NULLS LAST, id ASC").
		Find(&refs).Error
	if err != nil {
		return nil, nil, err
	}
	if len(refs) == 0 {
		return nil, nil, nil
	}
	ids := make([]uuid.UUID, 0, len(refs))
	for _, ref := range refs {
		ids = append(ids, ref.ToID)
	}
	var blocks []*blockstore.Block
	if err := r.db.WithContext(ctx).
		Where("id IN ? AND deleted_at IS NULL", ids).
		Find(&blocks).Error; err != nil {
		return nil, nil, err
	}
	byID := make(map[uuid.UUID]*blockstore.Block, len(blocks))
	for _, b := range blocks {
		byID[b.ID] = b
	}
	orderedBlocks := make([]*blockstore.Block, 0, len(refs))
	for _, ref := range refs {
		if b, ok := byID[ref.ToID]; ok {
			orderedBlocks = append(orderedBlocks, b)
		}
	}
	return orderedBlocks, refs, nil
}

func (r *Repository) ListBacklinks(ctx context.Context, targetID uuid.UUID, excludeNest bool) ([]*blockstore.BlockRef, error) {
	q := r.db.WithContext(ctx).
		Table("block_refs AS r").
		Joins("JOIN blocks b ON b.id = r.from_id").
		Where("r.to_id = ? AND b.deleted_at IS NULL", targetID)
	if excludeNest {
		q = q.Where("r.rel <> ?", blockstore.RelNest)
	}
	var refs []*blockstore.BlockRef
	if err := q.Select("r.*").Order("r.created_at DESC").Find(&refs).Error; err != nil {
		return nil, err
	}
	return refs, nil
}

func (r *Repository) ListRefs(ctx context.Context, f blockstore.RefFilter) ([]*blockstore.BlockRef, error) {
	q := r.db.WithContext(ctx).Where("workspace_id = ?", f.WorkspaceID)
	if f.FromID != nil {
		q = q.Where("from_id = ?", *f.FromID)
	}
	if f.ToID != nil {
		q = q.Where("to_id = ?", *f.ToID)
	}
	if f.Rel != nil {
		q = q.Where("rel = ?", *f.Rel)
	}
	if f.Limit > 0 {
		q = q.Limit(f.Limit)
	}
	if f.Offset > 0 {
		q = q.Offset(f.Offset)
	}
	q = q.Order("id ASC")
	var refs []*blockstore.BlockRef
	if err := q.Find(&refs).Error; err != nil {
		return nil, err
	}
	return refs, nil
}

func (r *Repository) StreamOps(ctx context.Context, f blockstore.OpStreamFilter) ([]*blockstore.BlockOp, error) {
	q := r.db.WithContext(ctx).
		Where("workspace_id = ? AND id > ?", f.WorkspaceID, f.AfterID).
		Order("id ASC")
	if f.Limit > 0 {
		q = q.Limit(f.Limit)
	}
	var ops []*blockstore.BlockOp
	if err := q.Find(&ops).Error; err != nil {
		return nil, err
	}
	return ops, nil
}

func (r *Repository) GetTypeDefByKey(
	ctx context.Context,
	workspaceID uuid.UUID,
	typeKey string,
) (*blockstore.Block, error) {
	q := r.db.WithContext(ctx).
		Model(&blockstore.Block{}).
		Where("workspace_id = ? AND type = ? AND deleted_at IS NULL",
			workspaceID, blockstore.BlockTypeTypeDef).
		Order("updated_at DESC").
		Limit(1)

	if r.db.Name() == "postgres" {
		q = q.Where("data->>'type_key' = ?", typeKey)
	} else {
		q = q.Where("data LIKE ?", `%"type_key":"`+typeKey+`"%`)
	}

	var b blockstore.Block
	if err := q.First(&b).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &b, nil
}
