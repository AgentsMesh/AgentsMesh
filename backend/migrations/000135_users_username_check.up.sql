-- Phase 2: enforce slugkit contract on users.username for NEW writes only.
-- NOT VALID skips the historical scan; pre-existing rows (containing dots
-- like "kudin.private" from the OAuth-bug era) are not touched here. The
-- backfill program (Phase 3) rewrites them; VALIDATE CONSTRAINT (Phase 4)
-- then promotes the constraint to enforce historical compliance too.
ALTER TABLE users ADD CONSTRAINT users_username_format
  CHECK (username ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND char_length(username) BETWEEN 2 AND 100)
  NOT VALID;
