-- Performance indexes for 100K runner support
-- These indexes optimize high-frequency queries in the runner/pod lifecycle

-- Runner authentication: node_id lookup during registration and heartbeat
CREATE INDEX IF NOT EXISTS idx_runners_node_id ON runners(node_id);

-- Pod ticket association: lookup pods by ticket_id
CREATE INDEX IF NOT EXISTS idx_pods_ticket_id ON pods(ticket_id);

-- Pod reconciliation: query pods by runner and status during heartbeat
CREATE INDEX IF NOT EXISTS idx_pods_runner_status ON pods(runner_id, status);

-- Runner selection: find available runners for pod scheduling
-- Covers: WHERE org_id=? AND status=? AND is_enabled=? AND current_pods < max_concurrent_pods
CREATE INDEX IF NOT EXISTS idx_runners_available
  ON runners(organization_id, status, is_enabled, current_pods);

-- Pod listing: query pods by organization and status
CREATE INDEX IF NOT EXISTS idx_pods_org_status ON pods(organization_id, status);

-- Offline runner marking: periodic job to mark stale runners as offline
CREATE INDEX IF NOT EXISTS idx_runners_status_heartbeat ON runners(status, last_heartbeat);
