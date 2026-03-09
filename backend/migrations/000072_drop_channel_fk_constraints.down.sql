-- Re-add FK constraints (use with caution on large datasets — will do full table scan to validate)

-- channels table
ALTER TABLE channels ADD CONSTRAINT channels_organization_id_fkey
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE channels ADD CONSTRAINT channels_team_id_fkey
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE SET NULL;
ALTER TABLE channels ADD CONSTRAINT channels_repository_id_fkey
    FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE SET NULL;
ALTER TABLE channels ADD CONSTRAINT channels_created_by_user_id_fkey
    FOREIGN KEY (created_by_user_id) REFERENCES users(id);

-- channel_messages
ALTER TABLE channel_messages ADD CONSTRAINT channel_messages_channel_id_fkey
    FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE;
ALTER TABLE channel_messages ADD CONSTRAINT channel_messages_sender_user_id_fkey
    FOREIGN KEY (sender_user_id) REFERENCES users(id);

-- channel_members
ALTER TABLE channel_members ADD CONSTRAINT channel_members_channel_id_fkey
    FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE;
ALTER TABLE channel_members ADD CONSTRAINT channel_members_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- channel_read_states
ALTER TABLE channel_read_states ADD CONSTRAINT channel_read_states_channel_id_fkey
    FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE;
ALTER TABLE channel_read_states ADD CONSTRAINT channel_read_states_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- channel_pods
ALTER TABLE channel_pods ADD CONSTRAINT channel_pods_channel_id_fkey
    FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE;

-- channel_access
ALTER TABLE channel_access ADD CONSTRAINT channel_access_channel_id_fkey
    FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE;
ALTER TABLE channel_access ADD CONSTRAINT channel_access_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- notification_preferences
ALTER TABLE notification_preferences ADD CONSTRAINT notification_preferences_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
