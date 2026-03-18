DROP INDEX IF EXISTS idx_pods_agent_waiting;
ALTER TABLE pods DROP COLUMN IF EXISTS agent_waiting_since;
ALTER TABLE loops DROP COLUMN IF EXISTS idle_timeout_sec;
