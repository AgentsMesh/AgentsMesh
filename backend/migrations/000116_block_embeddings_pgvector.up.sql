-- Phase 4+ pgvector upgrade: swap JSONB vector storage for a native vector
-- column with an HNSW index. Makes semantic search scale from linear-scan
-- (thousands) to index-backed ANN (millions).
--
-- The pgvector extension is optional. `deploy/*/docker-compose.yml` uses
-- `pgvector/pgvector:pg16` which bundles it. On vanilla `postgres:16-alpine`
-- the extension is unavailable; the service layer detects this and falls
-- back to JSONB-only mode at runtime. We wrap the DDL in a DO block so the
-- migration succeeds in both environments — the column/index are only
-- created when the extension is present.

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_available_extensions WHERE name = 'vector') THEN
        EXECUTE 'CREATE EXTENSION IF NOT EXISTS vector';
        EXECUTE 'ALTER TABLE block_embeddings ADD COLUMN IF NOT EXISTS vec vector(256)';
        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_block_embeddings_hnsw ON block_embeddings USING hnsw (vec vector_cosine_ops)';
    END IF;
END$$;
