-- Phase 4+ architecture cleanup: tighten block_ops invariants.
--
-- 1. CHECK constraint: exactly one of (target_block, target_ref) is set.
--    Previously enforced only at the service layer. Adding it in DB keeps
--    rogue migrations / direct-SQL rescue scripts honest.
-- 2. Audit context: JSONB slot for request metadata (request_id, ip,
--    user_agent, trace_id). Reserved ahead of need so future audit work
--    doesn't require another migration on a table that's already the
--    hottest write path.

ALTER TABLE block_ops
    ADD COLUMN IF NOT EXISTS context JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE block_ops
    ADD CONSTRAINT block_ops_target_exclusive
    CHECK (
        (target_block IS NOT NULL AND target_ref IS NULL)
     OR (target_block IS NULL AND target_ref IS NOT NULL)
    );
