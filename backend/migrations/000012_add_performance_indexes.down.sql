-- Remove performance indexes
DROP INDEX IF EXISTS idx_runners_node_id;
DROP INDEX IF EXISTS idx_pods_ticket_id;
DROP INDEX IF EXISTS idx_pods_runner_status;
DROP INDEX IF EXISTS idx_runners_available;
DROP INDEX IF EXISTS idx_pods_org_status;
DROP INDEX IF EXISTS idx_runners_status_heartbeat;
