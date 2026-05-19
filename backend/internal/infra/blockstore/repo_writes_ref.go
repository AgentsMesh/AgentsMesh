package blockstoreinfra

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (w *txWriter) InsertRef(ctx context.Context, r *blockstore.BlockRef) (int64, error) {
	if r.WorkspaceID == uuid.Nil {
		r.WorkspaceID = w.workspaceID
	}
	if err := w.tx.WithContext(ctx).Create(r).Error; err != nil {
		return 0, err
	}
	return r.ID, nil
}

func (w *txWriter) DeleteRefByID(ctx context.Context, id int64) error {
	res := w.tx.WithContext(ctx).
		Where("id = ?", id).
		Delete(&blockstore.BlockRef{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return blockstore.ErrRefNotFound
	}
	return nil
}

func (w *txWriter) UpdateRefFields(ctx context.Context, id int64, fields map[string]any) error {
	if _, has := fields["updated_at"]; !has {
		fields["updated_at"] = gorm.Expr("CURRENT_TIMESTAMP")
	}
	res := w.tx.WithContext(ctx).
		Model(&blockstore.BlockRef{}).
		Where("id = ?", id).
		Updates(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return blockstore.ErrRefNotFound
	}
	return nil
}

func (w *txWriter) FindRefByID(ctx context.Context, id int64) (*blockstore.BlockRef, error) {
	var r blockstore.BlockRef
	err := w.tx.WithContext(ctx).Where("id = ?", id).First(&r).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, blockstore.ErrRefNotFound
		}
		return nil, err
	}
	return &r, nil
}

func (w *txWriter) FindNestParent(ctx context.Context, childID uuid.UUID) (*blockstore.BlockRef, error) {
	var r blockstore.BlockRef
	err := w.tx.WithContext(ctx).
		Where("to_id = ? AND rel = ?", childID, blockstore.RelNest).
		First(&r).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

func (w *txWriter) FindAncestors(ctx context.Context, blockID uuid.UUID, maxDepth int) ([]uuid.UUID, error) {
	if maxDepth <= 0 {
		maxDepth = 64
	}
	const q = `
WITH RECURSIVE up AS (
  SELECT from_id AS ancestor, 1 AS depth
    FROM block_refs
   WHERE to_id = ? AND rel = 'nest'
  UNION ALL
  SELECT r.from_id, up.depth + 1
    FROM block_refs r JOIN up ON r.to_id = up.ancestor AND r.rel = 'nest'
   WHERE up.depth < ?
)
SELECT ancestor FROM up;`
	rows, err := w.tx.WithContext(ctx).Raw(q, blockID, maxDepth).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
