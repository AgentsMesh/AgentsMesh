-- Rollback: Remove gRPC + mTLS related tables

-- Drop reactivation tokens table
DROP TABLE IF EXISTS runner_reactivation_tokens;

-- Drop registration tokens table
DROP TABLE IF EXISTS runner_registration_tokens;

-- Drop pending auths table
DROP TABLE IF EXISTS runner_pending_auths;

-- Drop certificates table
DROP TABLE IF EXISTS runner_certificates;

-- Remove certificate columns from runners table
ALTER TABLE runners
    DROP COLUMN IF EXISTS cert_serial_number,
    DROP COLUMN IF EXISTS cert_expires_at;
