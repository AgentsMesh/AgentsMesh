-- Rename binding scopes from terminal:read/write to pod:read/write
UPDATE pod_bindings
SET granted_scopes = array_replace(granted_scopes, 'terminal:read', 'pod:read');

UPDATE pod_bindings
SET granted_scopes = array_replace(granted_scopes, 'terminal:write', 'pod:write');

UPDATE pod_bindings
SET pending_scopes = array_replace(pending_scopes, 'terminal:read', 'pod:read');

UPDATE pod_bindings
SET pending_scopes = array_replace(pending_scopes, 'terminal:write', 'pod:write');
