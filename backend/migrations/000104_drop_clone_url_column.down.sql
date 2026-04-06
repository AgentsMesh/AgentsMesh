-- Restore legacy clone_url column and populate from http_clone_url.
ALTER TABLE repositories ADD COLUMN IF NOT EXISTS clone_url VARCHAR(500);
UPDATE repositories SET clone_url = http_clone_url WHERE clone_url IS NULL;
