-- 000161_channel_read_states_manually_unread.up.sql
-- "Mark as unread" — a sticky per-user flag, orthogonal to the monotonic
-- last_read cursor (the cursor is never rewound). Cleared on the next read.
BEGIN;

ALTER TABLE channel_read_states
    ADD COLUMN manually_unread BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN channel_read_states.manually_unread IS
    'User explicitly marked the channel unread; persists until the channel is '
    'next read. Orthogonal to last_read_message_id (never rewinds the cursor).';

COMMIT;
