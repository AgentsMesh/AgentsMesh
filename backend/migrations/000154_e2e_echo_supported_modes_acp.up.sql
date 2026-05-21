-- 000154_e2e_echo_supported_modes_acp.up.sql
-- The e2e-echo agent's `supported_modes` column has been 'pty' since 000127.
-- Migration 000151 added MODE acp to the AgentFile, but did not update the
-- separate `supported_modes` DB column the backend's CreatePod handler
-- validates against (pod_create.go:117 UNSUPPORTED_INTERACTION_MODE). Set
-- it to 'pty,acp' so ACP-mode pod creation through the e2e-mock-agent is
-- accepted by the orchestrator.

BEGIN;

UPDATE agents
SET supported_modes = 'pty,acp'
WHERE slug = 'e2e-echo';

COMMIT;
