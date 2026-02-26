-- Add agent_versions column to runners table
-- Stores detected version info for each installed agent (slug, version, path)
-- Populated during Runner initialization handshake (requires Runner >= 0.4.7)
ALTER TABLE runners ADD COLUMN IF NOT EXISTS agent_versions JSONB DEFAULT '[]'::jsonb;

COMMENT ON COLUMN runners.agent_versions IS 'Detected version info for installed agents [{slug, version, path}], populated during initialization handshake';
