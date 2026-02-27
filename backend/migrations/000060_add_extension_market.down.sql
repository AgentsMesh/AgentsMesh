-- Revert Extension Market tables and Claude Code template changes

-- Restore Claude Code command_template: re-add --mcp-config arg
UPDATE agent_types SET
    command_template = jsonb_set(
        command_template,
        '{args}',
        command_template->'args' || '[{"condition": {"field": "mcp_enabled", "operator": "eq", "value": true}, "args": ["--mcp-config", "{{.sandbox.root_path}}/mcp-config.json"]}]'::jsonb
    )
WHERE slug = 'claude-code'
  AND command_template->'args' IS NOT NULL;

-- Restore Claude Code files_template: re-add mcp-config.json entry
UPDATE agent_types SET
    files_template = jsonb_build_array(
        jsonb_build_object(
            'condition', jsonb_build_object('field', 'mcp_enabled', 'operator', 'eq', 'value', true),
            'path_template', '{{.sandbox.root_path}}/mcp-config.json',
            'content_template', '{"mcpServers":{"agentsmesh":{"type":"http","url":"http://127.0.0.1:{{.mcp_port}}/mcp","headers":{"X-Pod-Key":"{{.pod_key}}"}}}}',
            'mode', 384
        )
    ) || COALESCE(files_template, '[]'::jsonb)
WHERE slug = 'claude-code';

-- Drop partial unique indexes explicitly before dropping tables
DROP INDEX IF EXISTS idx_installed_skills_unique;
DROP INDEX IF EXISTS idx_installed_skills_unique_no_user;
DROP INDEX IF EXISTS idx_installed_mcp_servers_unique;
DROP INDEX IF EXISTS idx_installed_mcp_servers_unique_no_user;
DROP INDEX IF EXISTS idx_skill_registries_unique_url;
DROP INDEX IF EXISTS idx_skill_registries_unique_url_platform;
DROP INDEX IF EXISTS idx_mcp_market_items_registry_name;

-- Drop tables in reverse order (respecting foreign keys)
DROP TABLE IF EXISTS installed_skills;
DROP TABLE IF EXISTS installed_mcp_servers;
DROP TABLE IF EXISTS mcp_market_items;
DROP TABLE IF EXISTS skill_market_items;
DROP TABLE IF EXISTS skill_registry_overrides;
DROP TABLE IF EXISTS skill_registries;
