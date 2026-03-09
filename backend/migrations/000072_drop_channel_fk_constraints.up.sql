-- Drop ALL FK constraints on channel-related tables.
-- Rationale: FK validation on every INSERT adds overhead on high-write tables.
-- CASCADE DELETE on large tables causes long-running transactions + table locks.
-- Application-layer integrity is enforced in the service layer; indexes are retained for queries.

-- channels table itself
ALTER TABLE channels DROP CONSTRAINT IF EXISTS channels_organization_id_fkey;
ALTER TABLE channels DROP CONSTRAINT IF EXISTS channels_team_id_fkey;
ALTER TABLE channels DROP CONSTRAINT IF EXISTS channels_repository_id_fkey;
ALTER TABLE channels DROP CONSTRAINT IF EXISTS channels_created_by_user_id_fkey;

-- channel_messages
ALTER TABLE channel_messages DROP CONSTRAINT IF EXISTS channel_messages_channel_id_fkey;
ALTER TABLE channel_messages DROP CONSTRAINT IF EXISTS channel_messages_sender_user_id_fkey;

-- channel_members
ALTER TABLE channel_members DROP CONSTRAINT IF EXISTS channel_members_channel_id_fkey;
ALTER TABLE channel_members DROP CONSTRAINT IF EXISTS channel_members_user_id_fkey;

-- channel_read_states
ALTER TABLE channel_read_states DROP CONSTRAINT IF EXISTS channel_read_states_channel_id_fkey;
ALTER TABLE channel_read_states DROP CONSTRAINT IF EXISTS channel_read_states_user_id_fkey;

-- channel_pods
ALTER TABLE channel_pods DROP CONSTRAINT IF EXISTS channel_pods_channel_id_fkey;

-- channel_access
ALTER TABLE channel_access DROP CONSTRAINT IF EXISTS channel_access_channel_id_fkey;
ALTER TABLE channel_access DROP CONSTRAINT IF EXISTS channel_access_user_id_fkey;

-- notification_preferences
ALTER TABLE notification_preferences DROP CONSTRAINT IF EXISTS notification_preferences_user_id_fkey;
