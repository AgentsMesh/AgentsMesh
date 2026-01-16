-- Add new columns to agent_types for config-driven agent setup
ALTER TABLE agent_types
    ADD COLUMN IF NOT EXISTS executable VARCHAR(100),
    ADD COLUMN IF NOT EXISTS config_schema JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS command_template JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS files_template JSONB;

-- Update existing agent types with their configurations
-- Claude Code
UPDATE agent_types SET
    executable = 'claude',
    config_schema = '{
        "fields": [
            {
                "name": "mcp_enabled",
                "type": "boolean",
                "default": true,
                "label_key": "agent.claude_code.fields.mcp_enabled.label",
                "desc_key": "agent.claude_code.fields.mcp_enabled.desc"
            },
            {
                "name": "model",
                "type": "select",
                "default": "sonnet",
                "label_key": "agent.claude_code.fields.model.label",
                "desc_key": "agent.claude_code.fields.model.desc",
                "options": [
                    {"value": "opus", "label_key": "agent.claude_code.fields.model.options.opus"},
                    {"value": "sonnet", "label_key": "agent.claude_code.fields.model.options.sonnet"}
                ]
            },
            {
                "name": "permission_mode",
                "type": "select",
                "default": "default",
                "label_key": "agent.claude_code.fields.permission_mode.label",
                "desc_key": "agent.claude_code.fields.permission_mode.desc",
                "options": [
                    {"value": "default", "label_key": "agent.claude_code.fields.permission_mode.options.default"},
                    {"value": "plan", "label_key": "agent.claude_code.fields.permission_mode.options.plan"},
                    {"value": "bypassPermissions", "label_key": "agent.claude_code.fields.permission_mode.options.bypass"}
                ]
            },
            {
                "name": "think_level",
                "type": "select",
                "default": "",
                "label_key": "agent.claude_code.fields.think_level.label",
                "desc_key": "agent.claude_code.fields.think_level.desc",
                "options": [
                    {"value": "", "label_key": "agent.claude_code.fields.think_level.options.default"},
                    {"value": "think", "label_key": "agent.claude_code.fields.think_level.options.think"},
                    {"value": "ultrathink", "label_key": "agent.claude_code.fields.think_level.options.ultrathink"}
                ]
            }
        ]
    }'::jsonb,
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
    }'::jsonb,
    files_template = '[
        {
            "condition": {"field": "mcp_enabled", "operator": "eq", "value": true},
            "path_template": "{{.sandbox.root_path}}/mcp-config.json",
            "content_template": "{\"mcpServers\":{\"agentsmesh\":{\"type\":\"http\",\"url\":\"http://127.0.0.1:{{.mcp_port}}/mcp\",\"headers\":{\"X-Pod-Key\":\"{{.pod_key}}\"}}}}",
            "mode": 384
        },
        {
            "path_template": "{{.sandbox.work_dir}}/.claude/skills/am-delegate",
            "is_directory": true
        },
        {
            "path_template": "{{.sandbox.work_dir}}/.claude/skills/am-delegate/SKILL.md",
            "content_template": "# AM Delegate Skill\n\nThis skill allows delegating tasks to other agents in the AgentsMesh platform.\n\nUse the MCP tool `delegate_task` to delegate work to another agent.",
            "mode": 420
        }
    ]'::jsonb
WHERE slug = 'claude-code';

-- Gemini CLI
UPDATE agent_types SET
    executable = 'gemini',
    config_schema = '{
        "fields": [
            {
                "name": "mcp_enabled",
                "type": "boolean",
                "default": true,
                "label_key": "agent.gemini_cli.fields.mcp_enabled.label",
                "desc_key": "agent.gemini_cli.fields.mcp_enabled.desc"
            },
            {
                "name": "sandbox_mode",
                "type": "boolean",
                "default": false,
                "label_key": "agent.gemini_cli.fields.sandbox_mode.label",
                "desc_key": "agent.gemini_cli.fields.sandbox_mode.desc"
            }
        ]
    }'::jsonb,
    command_template = '{
        "args": [
            {
                "condition": {"field": "sandbox_mode", "operator": "eq", "value": true},
                "args": ["--sandbox"]
            }
        ]
    }'::jsonb,
    files_template = '[
        {
            "condition": {"field": "mcp_enabled", "operator": "eq", "value": true},
            "path_template": "{{.sandbox.work_dir}}/.gemini/settings.json",
            "content_template": "{\"mcpServers\":{\"agentsmesh\":{\"httpUrl\":\"http://127.0.0.1:{{.mcp_port}}/mcp\",\"headers\":{\"X-Pod-Key\":\"{{.pod_key}}\"}}}}",
            "mode": 384
        }
    ]'::jsonb
WHERE slug = 'gemini-cli';

-- Codex CLI
UPDATE agent_types SET
    executable = 'codex',
    config_schema = '{
        "fields": [
            {
                "name": "mcp_enabled",
                "type": "boolean",
                "default": true,
                "label_key": "agent.codex_cli.fields.mcp_enabled.label",
                "desc_key": "agent.codex_cli.fields.mcp_enabled.desc"
            },
            {
                "name": "approval_mode",
                "type": "select",
                "default": "suggest",
                "label_key": "agent.codex_cli.fields.approval_mode.label",
                "desc_key": "agent.codex_cli.fields.approval_mode.desc",
                "options": [
                    {"value": "suggest", "label_key": "agent.codex_cli.fields.approval_mode.options.suggest"},
                    {"value": "auto-edit", "label_key": "agent.codex_cli.fields.approval_mode.options.auto_edit"},
                    {"value": "full-auto", "label_key": "agent.codex_cli.fields.approval_mode.options.full_auto"}
                ]
            }
        ]
    }'::jsonb,
    command_template = '{
        "args": [
            {
                "condition": {"field": "approval_mode", "operator": "not_empty"},
                "args": ["--approval-mode", "{{.config.approval_mode}}"]
            }
        ]
    }'::jsonb,
    files_template = '[
        {
            "condition": {"field": "mcp_enabled", "operator": "eq", "value": true},
            "path_template": "{{.sandbox.work_dir}}/.codex/mcp.json",
            "content_template": "{\"mcpServers\":{\"agentsmesh\":{\"type\":\"http\",\"url\":\"http://127.0.0.1:{{.mcp_port}}/mcp\",\"headers\":{\"X-Pod-Key\":\"{{.pod_key}}\"}}}}",
            "mode": 384
        }
    ]'::jsonb
WHERE slug = 'codex-cli';

-- Aider
UPDATE agent_types SET
    executable = 'aider',
    config_schema = '{
        "fields": [
            {
                "name": "model",
                "type": "string",
                "default": "",
                "label_key": "agent.aider.fields.model.label",
                "desc_key": "agent.aider.fields.model.desc"
            },
            {
                "name": "edit_format",
                "type": "select",
                "default": "",
                "label_key": "agent.aider.fields.edit_format.label",
                "desc_key": "agent.aider.fields.edit_format.desc",
                "options": [
                    {"value": "", "label_key": "agent.aider.fields.edit_format.options.default"},
                    {"value": "whole", "label_key": "agent.aider.fields.edit_format.options.whole"},
                    {"value": "diff", "label_key": "agent.aider.fields.edit_format.options.diff"},
                    {"value": "udiff", "label_key": "agent.aider.fields.edit_format.options.udiff"}
                ]
            }
        ]
    }'::jsonb,
    command_template = '{
        "args": [
            {
                "condition": {"field": "model", "operator": "not_empty"},
                "args": ["--model", "{{.config.model}}"]
            },
            {
                "condition": {"field": "edit_format", "operator": "not_empty"},
                "args": ["--edit-format", "{{.config.edit_format}}"]
            }
        ]
    }'::jsonb,
    files_template = NULL
WHERE slug = 'aider';

-- OpenCode
UPDATE agent_types SET
    executable = 'opencode',
    config_schema = '{
        "fields": [
            {
                "name": "mcp_enabled",
                "type": "boolean",
                "default": true,
                "label_key": "agent.opencode.fields.mcp_enabled.label",
                "desc_key": "agent.opencode.fields.mcp_enabled.desc"
            }
        ]
    }'::jsonb,
    command_template = '{
        "args": []
    }'::jsonb,
    files_template = '[
        {
            "condition": {"field": "mcp_enabled", "operator": "eq", "value": true},
            "path_template": "{{.sandbox.work_dir}}/.opencode/mcp.json",
            "content_template": "{\"mcpServers\":{\"agentsmesh\":{\"type\":\"http\",\"url\":\"http://127.0.0.1:{{.mcp_port}}/mcp\",\"headers\":{\"X-Pod-Key\":\"{{.pod_key}}\"}}}}",
            "mode": 384
        }
    ]'::jsonb
WHERE slug = 'opencode';

-- Create index for faster lookups
CREATE INDEX IF NOT EXISTS idx_agent_types_slug_active ON agent_types(slug) WHERE is_active = true;
