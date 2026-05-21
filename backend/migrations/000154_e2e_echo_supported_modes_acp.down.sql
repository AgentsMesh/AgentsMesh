-- 000154_e2e_echo_supported_modes_acp.down.sql

BEGIN;

UPDATE agents
SET supported_modes = 'pty'
WHERE slug = 'e2e-echo';

COMMIT;
