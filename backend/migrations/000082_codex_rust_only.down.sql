-- Revert Codex CLI Rust-only changes: restore mcp_enabled field, files_template, and command_template

-- Restore files_template with .codex/mcp.json entry
UPDATE agent_types SET
    files_template = '[
        {
            "condition": {"field": "mcp_enabled", "operator": "eq", "value": true},
            "path_template": "{{.sandbox.work_dir}}/.codex/mcp.json",
            "content_template": "{\"mcpServers\":{\"agentsmesh\":{\"type\":\"http\",\"url\":\"http://127.0.0.1:{{.mcp_port}}/mcp\",\"headers\":{\"X-Pod-Key\":\"{{.pod_key}}\"}}}}",
            "mode": 384
        }
    ]'::jsonb
WHERE slug = 'codex-cli';

-- Restore mcp_enabled field in config_schema (prepend, skip if already present)
UPDATE agent_types SET
    config_schema = jsonb_set(
        config_schema,
        '{fields}',
        jsonb_build_array(
            jsonb_build_object(
                'name', 'mcp_enabled',
                'type', 'boolean',
                'default', true
            )
        ) || (
            SELECT COALESCE(jsonb_agg(field), '[]'::jsonb)
            FROM jsonb_array_elements(config_schema->'fields') AS field
            WHERE field->>'name' != 'mcp_enabled'
        )
    )
WHERE slug = 'codex-cli';

-- Restore mcp_enabled conditional arg in command_template
UPDATE agent_types SET
    command_template = jsonb_set(
        command_template,
        '{args}',
        COALESCE(command_template->'args', '[]'::jsonb) || '[{"condition": {"field": "mcp_enabled", "operator": "eq", "value": true}, "args": ["--mcp-config", "{{.sandbox.root_path}}/mcp-config.json"]}]'::jsonb
    )
WHERE slug = 'codex-cli'
  AND NOT EXISTS (
      SELECT 1
      FROM jsonb_array_elements(command_template->'args') AS arg
      WHERE arg->'condition'->>'field' = 'mcp_enabled'
  );
