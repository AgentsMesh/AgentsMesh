-- Rollback: Restore old table name and remove new columns

-- Step 1: Migrate data back to old columns
UPDATE runner_grpc_registration_tokens
SET description = name,
    is_active = single_use,
    created_by_id = created_by
WHERE description IS NULL AND name IS NOT NULL;

-- Step 2: Drop new columns
ALTER TABLE runner_grpc_registration_tokens
DROP COLUMN IF EXISTS name;

ALTER TABLE runner_grpc_registration_tokens
DROP COLUMN IF EXISTS labels;

ALTER TABLE runner_grpc_registration_tokens
DROP COLUMN IF EXISTS single_use;

ALTER TABLE runner_grpc_registration_tokens
DROP COLUMN IF EXISTS created_by;

-- Step 3: Rename indexes back
ALTER INDEX IF EXISTS idx_grpc_reg_tokens_org RENAME TO idx_runner_reg_tokens_org;
ALTER INDEX IF EXISTS idx_grpc_reg_tokens_expires RENAME TO idx_runner_reg_tokens_expires;

-- Step 4: Rename table back
ALTER TABLE IF EXISTS runner_grpc_registration_tokens
RENAME TO runner_registration_tokens;
