ALTER TABLE pods ALTER COLUMN agent_status SET DEFAULT 'idle';
UPDATE pods SET agent_status = 'idle' WHERE agent_status IN ('unknown', 'finished', 'not_running', '');
UPDATE pods SET agent_status = 'executing' WHERE agent_status = 'working';
