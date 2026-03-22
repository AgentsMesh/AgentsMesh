ALTER TABLE agent_types ADD COLUMN supported_modes VARCHAR(50) NOT NULL DEFAULT 'pty';
UPDATE agent_types SET supported_modes = 'pty,acp' WHERE slug IN ('claude-code');
