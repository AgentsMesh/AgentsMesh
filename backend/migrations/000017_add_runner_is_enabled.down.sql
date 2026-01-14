-- Remove is_enabled column from runners table

ALTER TABLE runners DROP COLUMN IF EXISTS is_enabled;
