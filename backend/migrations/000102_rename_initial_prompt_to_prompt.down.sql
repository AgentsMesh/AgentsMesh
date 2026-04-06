-- Revert: prompt → initial_prompt
ALTER TABLE pods RENAME COLUMN prompt TO initial_prompt;
ALTER TABLE IF EXISTS ralph_pods RENAME COLUMN prompt TO initial_prompt;
