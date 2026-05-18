package blockstoreinfra

import (
	"context"
	"encoding/json"
	"errors"
	"math"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListEmbeddings returns every embedding row for the given workspace joined
// with blocks for ACL projection (created_by, meta). Callers rank in memory
// when pgvector is unavailable.
func (r *Repository) ListEmbeddings(
	ctx context.Context,
	workspaceID uuid.UUID,
	model string,
) ([]blockstore.EmbeddingRow, error) {
	type row struct {
		BlockID   string
		Type      string
		Text      *string
		Vector    string
		CreatedBy int64
		Meta      string
	}
	var rows []row
	q := r.db.WithContext(ctx).Raw(`
		SELECT e.block_id  AS block_id,
		       b.type      AS type,
		       b.text      AS text,
		       e.vector    AS vector,
		       b.created_by AS created_by,
		       b.meta      AS meta
		  FROM block_embeddings e
		  JOIN blocks b ON b.id = e.block_id
		 WHERE b.workspace_id = ? AND b.deleted_at IS NULL
		   AND (? = '' OR e.model = ?)
	`, workspaceID, model, model)
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]blockstore.EmbeddingRow, 0, len(rows))
	for _, r := range rows {
		if er, ok := oneEmbeddingRow(r.BlockID, r.Type, r.Text, r.Vector, r.CreatedBy, r.Meta); ok {
			out = append(out, er)
		}
	}
	return out, nil
}

// GetEmbeddingHash looks up the current source_hash for a block, or "" when
// no row exists. Used by the service to short-circuit re-embedding unchanged
// text.
func (r *Repository) GetEmbeddingHash(ctx context.Context, blockID uuid.UUID) (string, error) {
	var hash string
	err := r.db.WithContext(ctx).
		Raw(`SELECT source_hash FROM block_embeddings WHERE block_id = ?`, blockID).
		Row().Scan(&hash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || err.Error() == "sql: no rows in result set" {
			return "", nil
		}
		return "", err
	}
	return hash, nil
}

// SearchEmbeddings uses pgvector + HNSW to rank server-side when available
// (`ORDER BY vec <=> $1 LIMIT topK`). Falls back to ListEmbeddings on SQLite
// / extension-less Postgres; service layer ranks in memory in that case.
func (r *Repository) SearchEmbeddings(
	ctx context.Context,
	workspaceID uuid.UUID,
	model string,
	queryVec []float32,
	topK int,
) ([]blockstore.EmbeddingRow, error) {
	usePgvector, colDims := r.detectPgvector()
	if !usePgvector || colDims != len(queryVec) {
		// Dim mismatch means the column can't answer this query anyway —
		// return the full set so the service ranks in memory.
		return r.ListEmbeddings(ctx, workspaceID, model)
	}
	type row struct {
		BlockID   string
		Type      string
		Text      *string
		Vector    string
		CreatedBy int64
		Meta      string
		Distance  float64
	}
	if topK <= 0 {
		topK = 20
	}
	var rows []row
	vecLit := formatVectorLiteral(queryVec)
	q := r.db.WithContext(ctx).Raw(`
		SELECT e.block_id  AS block_id,
		       b.type      AS type,
		       b.text      AS text,
		       e.vector    AS vector,
		       b.created_by AS created_by,
		       b.meta      AS meta,
		       (e.vec <=> ?::vector) AS distance
		  FROM block_embeddings e
		  JOIN blocks b ON b.id = e.block_id
		 WHERE b.workspace_id = ? AND b.deleted_at IS NULL
		   AND (? = '' OR e.model = ?)
		 ORDER BY distance ASC
		 LIMIT ?
	`, vecLit, workspaceID, model, model, topK)
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]blockstore.EmbeddingRow, 0, len(rows))
	for _, r := range rows {
		er, ok := oneEmbeddingRow(r.BlockID, r.Type, r.Text, r.Vector, r.CreatedBy, r.Meta)
		if !ok {
			continue
		}
		// pgvector returns NaN distance for the zero vector; clamp so the row
		// ranks below any non-zero min_score (json.Marshal can't encode NaN).
		if math.IsNaN(r.Distance) || math.IsInf(r.Distance, 0) {
			er.Score = -1
		} else {
			er.Score = float32(1.0 - r.Distance)
		}
		out = append(out, er)
	}
	return out, nil
}

// oneEmbeddingRow parses a single projection into an EmbeddingRow. Factored
// so both the pgvector and JSONB read paths share decoding rules (skip rows
// whose vector JSON or id is corrupt — log-and-continue semantics).
func oneEmbeddingRow(
	blockID, blockType string,
	text *string,
	vectorJSON string,
	createdBy int64,
	metaJSON string,
) (blockstore.EmbeddingRow, bool) {
	var vec []float32
	if err := json.Unmarshal([]byte(vectorJSON), &vec); err != nil {
		return blockstore.EmbeddingRow{}, false
	}
	bid, err := uuid.Parse(blockID)
	if err != nil {
		return blockstore.EmbeddingRow{}, false
	}
	var meta blockstore.JSONMap
	if metaJSON != "" {
		_ = json.Unmarshal([]byte(metaJSON), &meta)
	}
	return blockstore.EmbeddingRow{
		BlockID:   bid,
		Type:      blockType,
		Text:      text,
		Vector:    vec,
		CreatedBy: createdBy,
		Meta:      meta,
	}, true
}
