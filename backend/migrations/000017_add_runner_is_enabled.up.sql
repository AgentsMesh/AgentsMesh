-- Add is_enabled column to runners table
-- This column was missing from the original schema but is required by the domain model

ALTER TABLE runners ADD COLUMN IF NOT EXISTS is_enabled BOOLEAN NOT NULL DEFAULT TRUE;

COMMENT ON COLUMN runners.is_enabled IS 'Whether the runner is enabled and can accept new pods';
