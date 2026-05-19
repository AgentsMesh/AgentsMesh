package blockstoreinfra

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// txWriter is the per-transaction write surface passed to ApplyOps callers.
// It shares the *gorm.DB of the enclosing transaction and is short-lived.
// Block / ref / op writers live in sibling files under this same type; this
// file owns the op-log write surface plus helpers that resolve state within
// the enclosing transaction (idempotency lookup, type-def listing).
type txWriter struct {
	tx          *gorm.DB
	workspaceID uuid.UUID
}

func (w *txWriter) InsertOp(ctx context.Context, o *blockstore.BlockOp) (int64, error) {
	if o.WorkspaceID == uuid.Nil {
		o.WorkspaceID = w.workspaceID
	}
	if err := w.tx.WithContext(ctx).Create(o).Error; err != nil {
		return 0, err
	}
	return o.ID, nil
}

func (w *txWriter) FindOpByIdempotencyKey(ctx context.Context, key string) (*blockstore.BlockOp, error) {
	if key == "" {
		return nil, nil
	}
	var o blockstore.BlockOp
	err := w.tx.WithContext(ctx).
		Where("idempotency_key = ?", key).
		First(&o).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}

func (w *txWriter) ListOpsByParent(ctx context.Context, parentOpID int64) ([]*blockstore.BlockOp, error) {
	var ops []*blockstore.BlockOp
	err := w.tx.WithContext(ctx).
		Where("parent_op_id = ?", parentOpID).
		Order("id ASC").
		Find(&ops).Error
	if err != nil {
		return nil, err
	}
	return ops, nil
}

func (w *txWriter) ListTypeDefs(ctx context.Context) ([]*blockstore.Block, error) {
	var out []*blockstore.Block
	err := w.tx.WithContext(ctx).
		Where("workspace_id = ? AND type = ? AND deleted_at IS NULL",
			w.workspaceID, blockstore.BlockTypeTypeDef).
		Order("updated_at DESC").
		Find(&out).Error
	return out, err
}
