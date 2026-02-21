DROP INDEX IF EXISTS idx_runners_registered_by_user_id;
DROP INDEX IF EXISTS idx_runners_visibility;
ALTER TABLE runners DROP COLUMN IF EXISTS registered_by_user_id;
ALTER TABLE runners DROP COLUMN IF EXISTS visibility;
