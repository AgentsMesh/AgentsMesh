ALTER TABLE block_ops DROP CONSTRAINT IF EXISTS block_ops_target_exclusive;
ALTER TABLE block_ops DROP COLUMN IF EXISTS context;
