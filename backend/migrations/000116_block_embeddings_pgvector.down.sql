DROP INDEX IF EXISTS idx_block_embeddings_hnsw;
ALTER TABLE block_embeddings DROP COLUMN IF EXISTS vec;
-- Intentionally do NOT DROP EXTENSION vector — other modules may rely on it.
