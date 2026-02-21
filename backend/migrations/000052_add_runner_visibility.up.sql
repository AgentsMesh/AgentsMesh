ALTER TABLE runners ADD COLUMN visibility VARCHAR(20) NOT NULL DEFAULT 'organization';
ALTER TABLE runners ADD COLUMN registered_by_user_id BIGINT;
CREATE INDEX idx_runners_visibility ON runners(visibility);
CREATE INDEX idx_runners_registered_by_user_id ON runners(registered_by_user_id);
