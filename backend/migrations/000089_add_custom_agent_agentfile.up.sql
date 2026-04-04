-- Add agentfile_source to custom_agent_types
ALTER TABLE custom_agent_types ADD COLUMN IF NOT EXISTS agentfile_source TEXT;
