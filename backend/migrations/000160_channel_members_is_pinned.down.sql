-- 000160_channel_members_is_pinned.down.sql
BEGIN;

ALTER TABLE channel_members DROP COLUMN IF EXISTS is_pinned;

COMMIT;
