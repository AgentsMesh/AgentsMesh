package blockstoreinfra

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/google/uuid"
)

func (r *Repository) ListWorkspaceSubtree(
	ctx context.Context,
	workspaceID, rootID uuid.UUID,
	maxDepth int,
) ([]*blockstore.Block, []*blockstore.BlockRef, error) {
	if maxDepth <= 0 || maxDepth > 64 {
		maxDepth = 64
	}

	const treeQuery = `
WITH RECURSIVE tree AS (
    SELECT ?::uuid AS id, 0 AS depth
  UNION ALL
    SELECT r.to_id, t.depth + 1
      FROM block_refs r
      JOIN tree t ON r.from_id = t.id AND r.rel = 'nest'
     WHERE t.depth < ?
)
SELECT b.*
  FROM tree t
  JOIN blocks b ON b.id = t.id
 WHERE b.workspace_id = ? AND b.deleted_at IS NULL
 ORDER BY t.depth ASC;`

	var blocks []*blockstore.Block
	if err := r.db.WithContext(ctx).Raw(treeQuery, rootID, maxDepth, workspaceID).
		Scan(&blocks).Error; err != nil {
		return nil, nil, err
	}
	if len(blocks) == 0 {
		return nil, nil, nil
	}

	ids := make([]uuid.UUID, 0, len(blocks))
	for _, b := range blocks {
		ids = append(ids, b.ID)
	}

	var refs []*blockstore.BlockRef
	if err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND rel = ? AND (from_id IN ? OR to_id IN ?)",
			workspaceID, blockstore.RelNest, ids, ids).
		Order("from_id, order_key ASC NULLS LAST").
		Find(&refs).Error; err != nil {
		return nil, nil, err
	}
	return blocks, refs, nil
}
