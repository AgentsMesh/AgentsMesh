-- Add podfile_source to custom_agent_types
ALTER TABLE custom_agent_types ADD COLUMN IF NOT EXISTS podfile_source TEXT;
