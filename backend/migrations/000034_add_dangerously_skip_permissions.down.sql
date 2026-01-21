-- Rollback: Remove dangerously_skip_permissions field and restore bypassPermissions option

-- Step 1: Migrate user configurations back
-- Convert dangerously_skip_permissions=true back to permission_mode=bypassPermissions
UPDATE user_agent_configs
SET config = config
    - 'dangerously_skip_permissions'
    || jsonb_build_object('permission_mode', 'bypassPermissions')
WHERE agent_type_id = (SELECT id FROM agent_types WHERE slug = 'claude-code')
  AND (config->>'dangerously_skip_permissions')::boolean = true;

-- Step 2: Remove dangerously_skip_permissions from configs that have it but it's false
UPDATE user_agent_configs
SET config = config - 'dangerously_skip_permissions'
WHERE agent_type_id = (SELECT id FROM agent_types WHERE slug = 'claude-code')
  AND config ? 'dangerously_skip_permissions';

-- Step 3: Restore original config_schema with bypassPermissions option
UPDATE agent_types SET
    config_schema = '{
        "fields": [
            {
                "name": "mcp_enabled",
                "type": "boolean",
                "default": true
            },
            {
                "name": "model",
                "type": "select",
                "default": "sonnet",
                "options": [
                    {"value": "opus"},
                    {"value": "sonnet"}
                ]
            },
            {
                "name": "permission_mode",
                "type": "select",
                "default": "default",
                "options": [
                    {"value": "default"},
                    {"value": "plan"},
                    {"value": "bypassPermissions"}
                ]
            },
            {
                "name": "think_level",
                "type": "select",
                "default": "",
                "options": [
                    {"value": ""},
                    {"value": "think"},
                    {"value": "ultrathink"}
                ]
            }
        ]
    }'::jsonb
WHERE slug = 'claude-code';

-- Step 4: Restore original command_template with permission_mode=bypassPermissions condition
UPDATE agent_types SET
    command_template = '{
        "args": [
            {
                "condition": {"field": "model", "operator": "not_empty"},
                "args": ["--model", "{{.config.model}}"]
            },
            {
                "condition": {"field": "permission_mode", "operator": "eq", "value": "plan"},
                "args": ["--permission-mode", "plan"]
            },
            {
                "condition": {"field": "permission_mode", "operator": "eq", "value": "bypassPermissions"},
                "args": ["--dangerously-skip-permissions"]
            },
            {
                "condition": {"field": "think_level", "operator": "not_empty"},
                "args": ["--{{.config.think_level}}"]
            },
            {
                "condition": {"field": "mcp_enabled", "operator": "eq", "value": true},
                "args": ["--mcp-config", "{{.sandbox.root_path}}/mcp-config.json"]
            }
        ]
    }'::jsonb
WHERE slug = 'claude-code';
