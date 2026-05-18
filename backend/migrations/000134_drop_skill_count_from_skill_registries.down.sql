-- Restore skill_count column. Data cannot be recovered on rollback;
-- the column is repopulated by the next SkillImporter.SyncSource cycle,
-- which previously wrote it after each successful sync. Until that
-- cycle runs, every row reports skill_count = 0.
ALTER TABLE skill_registries ADD COLUMN IF NOT EXISTS skill_count INTEGER DEFAULT 0;
