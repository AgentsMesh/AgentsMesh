-- ============================================================================
-- Codex CLI: Drop Node.js support, only support Rust rewrite (>= 0.100.0)
--
-- The Rust rewrite changed:
--   - MCP config: .codex/mcp.json (JSON) → config.toml (TOML)
--   - Approval values: suggest → on-request, auto-edit → on-failure, full-auto → never
--   - New CODEX_HOME env var for config directory override
--
-- MCP injection is now managed in Go code (CodexCLIBuilder), not DB templates.
-- ============================================================================

-- 1. Clear files_template — remove old .codex/mcp.json entry
UPDATE agent_types SET
    files_template = '[]'::jsonb
WHERE slug = 'codex-cli'
  AND files_template IS NOT NULL;

-- 2. Remove mcp_enabled field from config_schema
UPDATE agent_types SET
    config_schema = jsonb_set(
        config_schema,
        '{fields}',
        (
            SELECT COALESCE(jsonb_agg(field), '[]'::jsonb)
            FROM jsonb_array_elements(config_schema->'fields') AS field
            WHERE field->>'name' != 'mcp_enabled'
        )
    )
WHERE slug = 'codex-cli'
  AND config_schema->'fields' IS NOT NULL;

-- 3. Remove mcp_enabled conditional args from command_template
UPDATE agent_types SET
    command_template = jsonb_set(
        command_template,
        '{args}',
        (
            SELECT COALESCE(jsonb_agg(arg), '[]'::jsonb)
            FROM jsonb_array_elements(command_template->'args') AS arg
            WHERE arg->'condition'->>'field' != 'mcp_enabled'
        )
    )
WHERE slug = 'codex-cli'
  AND command_template->'args' IS NOT NULL;
