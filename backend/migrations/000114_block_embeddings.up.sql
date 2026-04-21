-- Phase 4 B1: Block embeddings for semantic search + Agent memory.
--
-- Keeping embeddings in a dedicated table (rather than a column on blocks)
-- lets us evolve the vector shape and storage later (pgvector, side-car
-- index, HNSW tuning) without rewriting block rows.
--
-- NOTE: Phase 4 MVP stores the vector as JSONB of float32. When production
-- traffic demands it, swap the column to `vector(D)` from the pgvector
-- extension and add HNSW / IVFFLAT indexes.

CREATE TABLE block_embeddings (
    block_id    UUID PRIMARY KEY REFERENCES blocks(id) ON DELETE CASCADE,
    model       TEXT NOT NULL,
    dims        INT  NOT NULL,
    vector      JSONB NOT NULL,
    source_hash TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Used by the workspace-scoped semantic search query to avoid scanning the
-- whole table. block_embeddings rows inherit ACL by join-checking blocks.
CREATE INDEX idx_block_embeddings_model ON block_embeddings(model);
