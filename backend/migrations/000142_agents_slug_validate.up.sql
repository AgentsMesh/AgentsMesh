-- Phase 4: promote the agents and custom_agents slug CHECK to enforced.
-- Builtin agents.slug values are hard-coded compliant in migrations; custom
-- agent writes have been routed through slugkit since PR-1.5 (REST API
-- binding validate). Both should VALIDATE clean with no backfill required.
ALTER TABLE agents VALIDATE CONSTRAINT agents_slug_format;
ALTER TABLE custom_agents VALIDATE CONSTRAINT custom_agents_slug_format;
