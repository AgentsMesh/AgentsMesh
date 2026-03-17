-- Revert binding scopes from pod:read/write back to terminal:read/write
UPDATE pod_bindings
SET granted_scopes = array_replace(granted_scopes, 'pod:read', 'terminal:read');

UPDATE pod_bindings
SET granted_scopes = array_replace(granted_scopes, 'pod:write', 'terminal:write');

UPDATE pod_bindings
SET pending_scopes = array_replace(pending_scopes, 'pod:read', 'terminal:read');

UPDATE pod_bindings
SET pending_scopes = array_replace(pending_scopes, 'pod:write', 'terminal:write');
