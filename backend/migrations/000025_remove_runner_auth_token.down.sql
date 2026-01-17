-- Rollback: Re-add auth_token_hash column to runners table
-- Note: This will add the column back but existing runners will have NULL values
-- They will need to re-register or use certificate-based authentication

ALTER TABLE runners ADD COLUMN auth_token_hash VARCHAR(255);
