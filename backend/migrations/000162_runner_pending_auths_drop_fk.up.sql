-- 000162_runner_pending_auths_drop_fk.up.sql
-- Follows 000072: this system enforces referential integrity in the service
-- layer, not with FK constraints. runner_pending_auths kept two FKs into
-- runners/organizations that 000029 never should have declared, and both were
-- left at the NO ACTION default, so:
--   runner_id:       DELETE of any runner that finished an interactive
--                    registration raised 23503 -- surfaced as a 500.
--   organization_id: DELETE of any org that ever saw a registration attempt
--                    raised 23503. DeleteOrganization's runner-count guard
--                    does not cover it: ClaimPendingAuth sets organization_id
--                    while runner_id stays NULL.
--
-- No hand-written cleanup replaces them: the table is self-expiring
-- (expires_at NOT NULL, 15 min) and the pending_auth_purge task drains it.
BEGIN;

ALTER TABLE runner_pending_auths
    DROP CONSTRAINT IF EXISTS runner_pending_auths_runner_id_fkey;

ALTER TABLE runner_pending_auths
    DROP CONSTRAINT IF EXISTS runner_pending_auths_organization_id_fkey;

COMMIT;
