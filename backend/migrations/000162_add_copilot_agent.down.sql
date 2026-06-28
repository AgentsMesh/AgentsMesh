-- 000162_add_copilot_agent.down.sql
-- Rollback path: clear FK-referencing rows before removing the agents row,
-- wrapped in a transaction for atomicity (matches the BEGIN/COMMIT convention
-- in 000157/000156/000149/000130 down migrations).
--
-- organization_agents and organization_agent_configs carry NO ACTION foreign
-- keys onto agents(slug) (migration 000093:23,31), so deleting copilot-cli from
-- agents while any org enabled it would fail - they MUST be cleared first.
-- user_agent_configs carries agent_slug with an index but no FK (000090), so
-- it's cleared only to avoid orphan slug references, not for FK safety.
BEGIN;

DELETE FROM organization_agent_configs WHERE agent_slug = 'copilot-cli';
DELETE FROM organization_agents WHERE agent_slug = 'copilot-cli';
DELETE FROM user_agent_configs WHERE agent_slug = 'copilot-cli';
DELETE FROM agents WHERE slug = 'copilot-cli';

COMMIT;
