-- Phase 4: promote the username CHECK from NOT VALID to enforced. Deploy ONLY
-- after the Phase 3 backfill program has run with --apply and a fresh
-- --dry-run reports zero violations. VALIDATE scans the entire users table
-- under ACCESS EXCLUSIVE briefly; for very large tables, prefer running this
-- during a maintenance window.
ALTER TABLE users VALIDATE CONSTRAINT users_username_format;
