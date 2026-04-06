-- Drop legacy clone_url column from repositories table.
-- This column has been replaced by http_clone_url and ssh_clone_url (migration 000057).
ALTER TABLE repositories DROP COLUMN IF EXISTS clone_url;
