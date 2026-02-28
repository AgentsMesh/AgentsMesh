-- Add max_retained_runs column to loops table
-- 0 = unlimited (keep all runs), > 0 = keep only the most recent N finished runs
ALTER TABLE loops ADD COLUMN max_retained_runs INT NOT NULL DEFAULT 0;
