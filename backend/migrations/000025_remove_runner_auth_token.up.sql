-- Migration: Remove auth_token_hash from runners table
-- Reason: Runner authentication now uses mTLS certificates instead of auth_token
-- The auth_token_hash column is no longer used after gRPC/mTLS migration

ALTER TABLE runners DROP COLUMN IF EXISTS auth_token_hash;
