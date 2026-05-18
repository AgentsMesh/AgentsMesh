-- Irreversible: the deleted rows were poison (zero vectors → NaN cosine
-- distance), so there is nothing to restore. Embeddings repopulate on the
-- next write to each affected block via embedBlock's hash-mismatch path.

SELECT 1;
