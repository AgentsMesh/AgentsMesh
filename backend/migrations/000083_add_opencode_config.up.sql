UPDATE agent_types SET
    config_schema = '{
        "fields": [
            {
                "name": "mcp_enabled",
                "type": "boolean",
                "default": true
            },
            {
                "name": "skip_permissions",
                "type": "boolean",
                "default": false
            },
            {
                "name": "models",
                "type": "model_list"
            }
        ]
    }'::jsonb
WHERE slug = 'opencode';
