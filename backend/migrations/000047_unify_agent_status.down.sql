ALTER TABLE pods ALTER COLUMN agent_status SET DEFAULT 'unknown';
UPDATE pods SET agent_status = 'unknown' WHERE agent_status = 'idle';
UPDATE pods SET agent_status = 'working' WHERE agent_status = 'executing';
