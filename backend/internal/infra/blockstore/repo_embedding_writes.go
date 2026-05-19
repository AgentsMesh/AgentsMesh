package blockstoreinfra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (r *Repository) detectPgvector() (bool, int) {
	r.pgvectorOnce.Do(func() {
		if r.db.Name() != "postgres" {
			return
		}
		var typeName string
		err := r.db.Raw(`
			SELECT format_type(a.atttypid, a.atttypmod)
			  FROM pg_attribute a
			  JOIN pg_class c ON a.attrelid = c.oid
			 WHERE c.relname = 'block_embeddings'
			   AND a.attname = 'vec'
			   AND NOT a.attisdropped
		`).Row().Scan(&typeName)
		if err != nil || typeName == "" {
			return
		}
		if i := strings.Index(typeName, "("); i > 0 {
			rest := typeName[i+1:]
			if j := strings.Index(rest, ")"); j > 0 {
				_, _ = fmt.Sscanf(rest[:j], "%d", &r.pgvectorDims)
			}
		}
		r.pgvectorReady = r.pgvectorDims > 0
	})
	return r.pgvectorReady, r.pgvectorDims
}

func (r *Repository) UpsertEmbedding(
	ctx context.Context,
	blockID uuid.UUID,
	model string,
	dims int,
	vector []float32,
	sourceHash string,
) error {
	raw, err := json.Marshal(vector)
	if err != nil {
		return err
	}
	usePgvector, colDims := r.detectPgvector()
	if usePgvector && colDims == dims {
		vecLit := formatVectorLiteral(vector)
		return r.db.WithContext(ctx).Exec(`
			INSERT INTO block_embeddings (block_id, model, dims, vector, vec, source_hash, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?::vector, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			ON CONFLICT (block_id) DO UPDATE SET
				model = excluded.model,
				dims = excluded.dims,
				vector = excluded.vector,
				vec = excluded.vec,
				source_hash = excluded.source_hash,
				updated_at = CURRENT_TIMESTAMP
		`, blockID, model, dims, string(raw), vecLit, sourceHash).Error
	}
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO block_embeddings (block_id, model, dims, vector, source_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (block_id) DO UPDATE SET
			model = excluded.model,
			dims = excluded.dims,
			vector = excluded.vector,
			source_hash = excluded.source_hash,
			updated_at = CURRENT_TIMESTAMP
	`, blockID, model, dims, string(raw), sourceHash).Error
}

func (r *Repository) DeleteEmbedding(ctx context.Context, blockID uuid.UUID) error {
	res := r.db.WithContext(ctx).Where("block_id = ?", blockID).Delete(&blockstore.BlockEmbedding{})
	if res.Error != nil && !errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return res.Error
	}
	return nil
}

func formatVectorLiteral(v []float32) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		_, _ = fmt.Fprintf(&b, "%g", x)
	}
	b.WriteByte(']')
	return b.String()
}
