-- Rename registration token table to match code's TableName()
-- Migration 000029 created 'runner_registration_tokens' but GRPCRegistrationToken.TableName() returns 'runner_grpc_registration_tokens'
-- Also need to migrate from old schema (000001) to new schema (gRPC/mTLS)

-- Step 1: Rename table
ALTER TABLE IF EXISTS runner_registration_tokens
RENAME TO runner_grpc_registration_tokens;

-- Step 2: Rename indexes
ALTER INDEX IF EXISTS idx_runner_reg_tokens_org RENAME TO idx_grpc_reg_tokens_org;
ALTER INDEX IF EXISTS idx_runner_reg_tokens_expires RENAME TO idx_grpc_reg_tokens_expires;

-- Step 3: Add new columns needed by GRPCRegistrationToken
-- Add 'name' column (maps from old 'description')
ALTER TABLE runner_grpc_registration_tokens
ADD COLUMN IF NOT EXISTS name VARCHAR(255);

-- Add 'labels' column for JSON labels
ALTER TABLE runner_grpc_registration_tokens
ADD COLUMN IF NOT EXISTS labels JSONB;

-- Add 'single_use' column (maps from old 'is_active')
ALTER TABLE runner_grpc_registration_tokens
ADD COLUMN IF NOT EXISTS single_use BOOLEAN DEFAULT TRUE;

-- Add 'created_by' column (maps from old 'created_by_id')
ALTER TABLE runner_grpc_registration_tokens
ADD COLUMN IF NOT EXISTS created_by BIGINT REFERENCES users(id);

-- Step 4: Migrate data from old columns to new columns
UPDATE runner_grpc_registration_tokens
SET name = description,
    single_use = is_active,
    created_by = created_by_id
WHERE name IS NULL AND description IS NOT NULL;

-- Step 5: Set defaults for required columns
UPDATE runner_grpc_registration_tokens
SET single_use = TRUE
WHERE single_use IS NULL;

UPDATE runner_grpc_registration_tokens
SET max_uses = 1
WHERE max_uses IS NULL;

UPDATE runner_grpc_registration_tokens
SET used_count = 0
WHERE used_count IS NULL;

-- Step 6: Make expires_at NOT NULL with a default for existing rows
UPDATE runner_grpc_registration_tokens
SET expires_at = created_at + INTERVAL '7 days'
WHERE expires_at IS NULL;

-- Step 7: Remove NOT NULL constraint from old columns (they are deprecated)
-- created_by_id is replaced by created_by
ALTER TABLE runner_grpc_registration_tokens
ALTER COLUMN created_by_id DROP NOT NULL;
