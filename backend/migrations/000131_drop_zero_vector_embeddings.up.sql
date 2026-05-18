-- Drop zero-vector embeddings produced by the legacy ASCII-only tokenizer
-- (any non-ASCII text → empty tokens → zero vector → pgvector NaN distance
-- → MCP 500). Affected blocks re-embed on their next write via embedBlock's
-- hash-mismatch path.

DELETE FROM block_embeddings WHERE vector_norm(vec) = 0;
