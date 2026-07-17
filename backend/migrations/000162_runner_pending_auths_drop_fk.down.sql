-- 000162_runner_pending_auths_drop_fk.down.sql
-- Restores 000029's constraints. Rows orphaned while the FKs were absent are
-- purged first, or ADD CONSTRAINT's validation scan would reject them.
BEGIN;

DELETE FROM runner_pending_auths
WHERE (runner_id IS NOT NULL AND runner_id NOT IN (SELECT id FROM runners))
   OR (organization_id IS NOT NULL AND organization_id NOT IN (SELECT id FROM organizations));

ALTER TABLE runner_pending_auths
    ADD CONSTRAINT runner_pending_auths_runner_id_fkey
    FOREIGN KEY (runner_id) REFERENCES runners(id);

ALTER TABLE runner_pending_auths
    ADD CONSTRAINT runner_pending_auths_organization_id_fkey
    FOREIGN KEY (organization_id) REFERENCES organizations(id);

COMMIT;
