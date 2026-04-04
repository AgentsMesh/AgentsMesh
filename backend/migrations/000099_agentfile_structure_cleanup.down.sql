-- Revert is a no-op since migration 098 already has the previous version.
-- To fully revert, run: migrate down 2 (reverts 099 + 098)
-- This will restore the pre-MODE-args AgentFile versions.
SELECT 1;
