-- 000163_runner_auth_timestamps_tz.down.sql
-- Back to tz-naive UTC wall clock.
BEGIN;

ALTER TABLE runner_certificates
    ALTER COLUMN issued_at  TYPE TIMESTAMP USING issued_at  AT TIME ZONE 'UTC',
    ALTER COLUMN expires_at TYPE TIMESTAMP USING expires_at AT TIME ZONE 'UTC',
    ALTER COLUMN revoked_at TYPE TIMESTAMP USING revoked_at AT TIME ZONE 'UTC',
    ALTER COLUMN created_at TYPE TIMESTAMP USING created_at AT TIME ZONE 'UTC';

ALTER TABLE runner_pending_auths
    ALTER COLUMN expires_at TYPE TIMESTAMP USING expires_at AT TIME ZONE 'UTC',
    ALTER COLUMN created_at TYPE TIMESTAMP USING created_at AT TIME ZONE 'UTC';

ALTER TABLE runner_reactivation_tokens
    ALTER COLUMN expires_at TYPE TIMESTAMP USING expires_at AT TIME ZONE 'UTC',
    ALTER COLUMN used_at    TYPE TIMESTAMP USING used_at    AT TIME ZONE 'UTC',
    ALTER COLUMN created_at TYPE TIMESTAMP USING created_at AT TIME ZONE 'UTC';

ALTER TABLE runners
    ALTER COLUMN cert_expires_at TYPE TIMESTAMP USING cert_expires_at AT TIME ZONE 'UTC';

COMMIT;
