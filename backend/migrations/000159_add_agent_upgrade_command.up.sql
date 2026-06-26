-- 000159_add_agent_upgrade_command.up.sql
-- Remote-upgrade command for builtin agents, the SSOT for "how to upgrade
-- this agent" (deliberately NOT in the AgentFile DSL — it's an ops attribute,
-- not pod-launch semantics).
--
-- upgrade_args is a JSON argv array the runner exec()s directly (no shell),
-- e.g. ["npm","i","-g","@anthropic-ai/claude-code@latest"]. NULL = the agent
-- does not support remote upgrade. cursor-cli (curl|bash install) and loopal
-- (self-built release) are intentionally left NULL — see plan
-- sleepy-mapping-charm.md §7 (no shell-pipe install commands).

ALTER TABLE agents ADD COLUMN upgrade_manager varchar(20);
ALTER TABLE agents ADD COLUMN upgrade_args text;

UPDATE agents SET upgrade_manager = 'npm', upgrade_args = '["npm","i","-g","@anthropic-ai/claude-code@latest"]' WHERE slug = 'claude-code';
UPDATE agents SET upgrade_manager = 'npm', upgrade_args = '["npm","i","-g","@openai/codex@latest"]' WHERE slug = 'codex-cli';
UPDATE agents SET upgrade_manager = 'npm', upgrade_args = '["npm","i","-g","@google/gemini-cli@latest"]' WHERE slug = 'gemini-cli';
UPDATE agents SET upgrade_manager = 'pip', upgrade_args = '["python3","-m","pip","install","--upgrade","aider-chat"]' WHERE slug = 'aider';

-- Both columns encode one fact ("how to upgrade"): keep them in lockstep so the
-- backend gate (upgrade_args presence) and the UI gate (upgrade_manager presence)
-- can never diverge into a dead button or a hidden-but-upgradable agent.
ALTER TABLE agents ADD CONSTRAINT agents_upgrade_pair_chk
  CHECK ((upgrade_manager IS NULL) = (upgrade_args IS NULL));
