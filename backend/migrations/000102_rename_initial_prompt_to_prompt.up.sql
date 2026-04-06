-- Rename initial_prompt → prompt for unified naming
ALTER TABLE pods RENAME COLUMN initial_prompt TO prompt;
ALTER TABLE IF EXISTS ralph_pods RENAME COLUMN initial_prompt TO prompt;
