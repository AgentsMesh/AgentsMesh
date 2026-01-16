-- AgentsMesh Database Migration Rollback
-- Migration: 000002_devmesh_ticket_enhancements
-- Description: Rollback Mesh and Ticket enhancements

-- ==========================================
-- 8. Drop Channel Access Table
-- ==========================================
DROP TABLE IF EXISTS channel_access;

-- ==========================================
-- 7. Drop Channel Pods Table
-- ==========================================
DROP TABLE IF EXISTS channel_pods;

-- ==========================================
-- 6. Drop Ticket Relations Table
-- ==========================================
DROP TABLE IF EXISTS ticket_relations;

-- ==========================================
-- 5. Drop Ticket Commits Table
-- ==========================================
DROP TABLE IF EXISTS ticket_commits;

-- ==========================================
-- 4. Remove Ticket Merge Request Enhancements
-- ==========================================
DROP INDEX IF EXISTS idx_ticket_mrs_pipeline;
ALTER TABLE ticket_merge_requests DROP COLUMN IF EXISTS pipeline_status;
ALTER TABLE ticket_merge_requests DROP COLUMN IF EXISTS pipeline_id;
ALTER TABLE ticket_merge_requests DROP COLUMN IF EXISTS pipeline_url;
ALTER TABLE ticket_merge_requests DROP COLUMN IF EXISTS merge_commit_sha;
ALTER TABLE ticket_merge_requests DROP COLUMN IF EXISTS merged_at;
ALTER TABLE ticket_merge_requests DROP COLUMN IF EXISTS merged_by_id;
ALTER TABLE ticket_merge_requests DROP COLUMN IF EXISTS last_synced_at;

-- ==========================================
-- 3. Remove Pod Binding Enhancements
-- ==========================================
ALTER TABLE pod_bindings DROP COLUMN IF EXISTS pending_scopes;
ALTER TABLE pod_bindings DROP COLUMN IF EXISTS requested_at;
ALTER TABLE pod_bindings DROP COLUMN IF EXISTS responded_at;
ALTER TABLE pod_bindings DROP COLUMN IF EXISTS expires_at;
ALTER TABLE pod_bindings DROP COLUMN IF EXISTS rejection_reason;

-- ==========================================
-- 2. Remove Pod Enhancements
-- ==========================================
ALTER TABLE pods DROP COLUMN IF EXISTS model;
ALTER TABLE pods DROP COLUMN IF EXISTS permission_mode;
ALTER TABLE pods DROP COLUMN IF EXISTS think_level;
ALTER TABLE pods DROP COLUMN IF EXISTS agent_pid;

-- ==========================================
-- 1. Remove Ticket Enhancements
-- ==========================================
DROP INDEX IF EXISTS idx_tickets_severity;
ALTER TABLE tickets DROP COLUMN IF EXISTS severity;
ALTER TABLE tickets DROP COLUMN IF EXISTS estimate;
