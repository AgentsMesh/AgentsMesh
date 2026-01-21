-- Rollback: Remove preparation_script and preparation_timeout from repositories table

ALTER TABLE repositories
DROP COLUMN IF EXISTS preparation_script,
DROP COLUMN IF EXISTS preparation_timeout;
