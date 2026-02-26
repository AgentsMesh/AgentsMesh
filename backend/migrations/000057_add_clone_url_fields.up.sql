-- Add separate HTTP and SSH clone URL fields to repositories
ALTER TABLE repositories ADD COLUMN http_clone_url VARCHAR(500);
ALTER TABLE repositories ADD COLUMN ssh_clone_url VARCHAR(500);

-- Migrate data from existing clone_url
UPDATE repositories SET http_clone_url = clone_url WHERE clone_url LIKE 'https://%' OR clone_url LIKE 'http://%';
UPDATE repositories SET ssh_clone_url = clone_url WHERE clone_url LIKE 'git@%' OR clone_url LIKE 'ssh://%';
