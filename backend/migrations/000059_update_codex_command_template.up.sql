-- Update Codex CLI command_template to use new --ask-for-approval flag (codex >= 0.1.2025042500).
-- The server-side version adapter (CodexCLIBuilder) automatically translates
-- this back to --approval-mode for older Codex CLI versions.
UPDATE agent_types SET
    command_template = '{
        "args": [
            {
                "condition": {"field": "approval_mode", "operator": "not_empty"},
                "args": ["--ask-for-approval", "{{.config.approval_mode}}"]
            }
        ]
    }'::jsonb
WHERE slug = 'codex-cli';
