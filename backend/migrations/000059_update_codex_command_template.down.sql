-- Revert Codex CLI command_template to old --approval-mode flag
UPDATE agent_types SET
    command_template = '{
        "args": [
            {
                "condition": {"field": "approval_mode", "operator": "not_empty"},
                "args": ["--approval-mode", "{{.config.approval_mode}}"]
            }
        ]
    }'::jsonb
WHERE slug = 'codex-cli';
