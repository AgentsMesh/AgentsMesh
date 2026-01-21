-- Add dangerously_skip_permissions boolean field to Claude Code config
-- and remove bypassPermissions from permission_mode options

-- Step 1: Update config_schema - add dangerously_skip_permissions field and remove bypassPermissions option
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
                    {"value": "plan"}
                ]
            },
            {
                "name": "dangerously_skip_permissions",
                "type": "boolean",
                "default": false
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

-- Step 2: Update command_template - change condition from permission_mode=bypassPermissions to dangerously_skip_permissions=true
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
                "condition": {"field": "dangerously_skip_permissions", "operator": "eq", "value": true},
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

-- Step 3: Migrate existing user configurations
-- Convert permission_mode=bypassPermissions to dangerously_skip_permissions=true
UPDATE user_agent_configs
SET config = config
    - 'permission_mode'
    || jsonb_build_object('permission_mode', 'default')
    || jsonb_build_object('dangerously_skip_permissions', true)
WHERE agent_type_id = (SELECT id FROM agent_types WHERE slug = 'claude-code')
  AND config->>'permission_mode' = 'bypassPermissions';
