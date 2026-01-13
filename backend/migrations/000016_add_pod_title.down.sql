-- Remove title column from pods table
ALTER TABLE pods DROP COLUMN IF EXISTS title;
