-- Add credential_profile_id to pods for audit tracking
ALTER TABLE pods ADD COLUMN IF NOT EXISTS credential_profile_id BIGINT;

-- Optional FK constraint (soft reference — profile may be deleted later)
-- No FK constraint intentionally: credential profiles can be deleted independently
