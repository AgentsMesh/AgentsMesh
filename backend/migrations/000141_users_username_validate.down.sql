-- Postgres has no "un-VALIDATE" command; downgrade re-creates the CHECK as
-- NOT VALID. The constraint name stays the same so application code is
-- unaffected.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_format;
ALTER TABLE users ADD CONSTRAINT users_username_format
  CHECK (username ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND char_length(username) BETWEEN 2 AND 100)
  NOT VALID;
