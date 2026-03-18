-- Loop execution policy: auto-terminate agent after idle timeout
ALTER TABLE loops ADD COLUMN IF NOT EXISTS idle_timeout_sec INT NOT NULL DEFAULT 30;

-- Pod runtime state: when the agent entered "waiting" state
ALTER TABLE pods ADD COLUMN IF NOT EXISTS agent_waiting_since TIMESTAMPTZ;

-- Partial index: accelerate scheduler idle-pod scan
CREATE INDEX IF NOT EXISTS idx_pods_agent_waiting
  ON pods (agent_waiting_since)
  WHERE status = 'running' AND agent_status = 'waiting' AND agent_waiting_since IS NOT NULL;
