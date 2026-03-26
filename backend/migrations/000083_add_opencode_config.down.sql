UPDATE agent_types SET
    config_schema = '{
        "fields": [
            {
                "name": "mcp_enabled",
                "type": "boolean",
                "default": true
            }
        ]
    }'::jsonb,
    command_template = '{"args": []}'::jsonb
WHERE slug = 'opencode';
