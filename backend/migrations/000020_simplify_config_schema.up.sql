-- Simplify config_schema structure: remove i18n keys (label_key, desc_key)
-- Frontend is now responsible for i18n using: agent.{slug}.fields.{field.name}.label

-- Claude Code
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

-- Gemini CLI
UPDATE agent_types SET
    config_schema = '{
        "fields": [
            {
                "name": "mcp_enabled",
                "type": "boolean",
                "default": true
            },
            {
                "name": "sandbox_mode",
                "type": "boolean",
                "default": false
            }
        ]
    }'::jsonb
WHERE slug = 'gemini-cli';

-- Codex CLI
UPDATE agent_types SET
    config_schema = '{
        "fields": [
            {
                "name": "mcp_enabled",
                "type": "boolean",
                "default": true
            },
            {
                "name": "approval_mode",
                "type": "select",
                "default": "suggest",
                "options": [
                    {"value": "suggest"},
                    {"value": "auto-edit"},
                    {"value": "full-auto"}
                ]
            }
        ]
    }'::jsonb
WHERE slug = 'codex-cli';

-- Aider
UPDATE agent_types SET
    config_schema = '{
        "fields": [
            {
                "name": "model",
                "type": "string",
                "default": ""
            },
            {
                "name": "edit_format",
                "type": "select",
                "default": "",
                "options": [
                    {"value": ""},
                    {"value": "whole"},
                    {"value": "diff"},
                    {"value": "udiff"}
                ]
            }
        ]
    }'::jsonb
WHERE slug = 'aider';

-- OpenCode
UPDATE agent_types SET
    config_schema = '{
        "fields": [
            {
                "name": "mcp_enabled",
                "type": "boolean",
                "default": true
            }
        ]
    }'::jsonb
WHERE slug = 'opencode';
