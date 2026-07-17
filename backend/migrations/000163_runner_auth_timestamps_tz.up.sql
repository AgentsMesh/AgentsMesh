-- 000163_runner_auth_timestamps_tz.up.sql
-- The 000029 runner-auth tables were declared with `TIMESTAMP` (no tz), the
-- minority form -- the rest of the schema is `TIMESTAMPTZ`. The Go code writes
-- and compares these with time.Now(), so on a non-UTC host the read path
-- mislabels the wall clock (pgx decodes a tz-naive column as UTC), and a purge
-- or claim would drift by the host offset. TIMESTAMPTZ stores an absolute
-- instant, tz-independent. Prod runs UTC, so the stored wall clock IS UTC and
-- `AT TIME ZONE 'UTC'` preserves every value.
BEGIN;

ALTER TABLE runner_certificates
    ALTER COLUMN issued_at  TYPE TIMESTAMPTZ USING issued_at  AT TIME ZONE 'UTC',
    ALTER COLUMN expires_at TYPE TIMESTAMPTZ USING expires_at AT TIME ZONE 'UTC',
    ALTER COLUMN revoked_at TYPE TIMESTAMPTZ USING revoked_at AT TIME ZONE 'UTC',
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC';

ALTER TABLE runner_pending_auths
    ALTER COLUMN expires_at TYPE TIMESTAMPTZ USING expires_at AT TIME ZONE 'UTC',
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC';

ALTER TABLE runner_reactivation_tokens
    ALTER COLUMN expires_at TYPE TIMESTAMPTZ USING expires_at AT TIME ZONE 'UTC',
    ALTER COLUMN used_at    TYPE TIMESTAMPTZ USING used_at    AT TIME ZONE 'UTC',
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC';

ALTER TABLE runners
    ALTER COLUMN cert_expires_at TYPE TIMESTAMPTZ USING cert_expires_at AT TIME ZONE 'UTC';

COMMIT;
