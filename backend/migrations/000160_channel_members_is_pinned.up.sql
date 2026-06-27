-- 000160_channel_members_is_pinned.up.sql
-- Per-user sidebar pin. Pinned channels render in a dedicated top group in the
-- sidebar regardless of the activity grouping (Active/Linked/Quiet).
BEGIN;

ALTER TABLE channel_members
    ADD COLUMN is_pinned BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN channel_members.is_pinned IS
    'Per-user sidebar pin. Pinned channels sort above all others.';

COMMIT;
