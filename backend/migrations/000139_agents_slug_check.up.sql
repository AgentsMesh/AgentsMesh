-- Phase 2: agents and custom_agents already have a slug column from
-- migration 000091 (rename). Builtin agents.slug values are hard-coded and
-- compliant; custom_agents.slug becomes safe after PR-1.5 added REST validate.
-- Both get CHECK NOT VALID; Phase 4 promotes them to VALIDATE.
ALTER TABLE agents ADD CONSTRAINT agents_slug_format
  CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND char_length(slug) BETWEEN 2 AND 100)
  NOT VALID;

ALTER TABLE custom_agents ADD CONSTRAINT custom_agents_slug_format
  CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND char_length(slug) BETWEEN 2 AND 100)
  NOT VALID;
