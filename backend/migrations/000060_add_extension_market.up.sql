-- Extension Market: Skill Registries, Market Items, and Installation tables
-- Consolidated from migrations 056-062 (pre-release feature)

-- ============================================================================
-- Table 1: skill_registries — Skill import sources (GitHub repositories)
-- ============================================================================
CREATE TABLE IF NOT EXISTS skill_registries (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT REFERENCES organizations(id) ON DELETE CASCADE,  -- NULL = platform-level
    repository_url VARCHAR(500) NOT NULL,
    branch VARCHAR(100) DEFAULT 'main',
    source_type VARCHAR(20) DEFAULT 'auto',  -- auto / collection / single
    detected_type VARCHAR(20),  -- collection / single (actual detection result)
    compatible_agents JSONB DEFAULT '["claude-code"]',  -- agent type whitelist
    auth_type VARCHAR(20) DEFAULT 'none',  -- none / github_pat / gitlab_pat / ssh_key
    auth_credential TEXT,  -- encrypted credential (PAT, SSH key, etc.)
    last_synced_at TIMESTAMP WITH TIME ZONE,
    last_commit_sha VARCHAR(40),
    sync_status VARCHAR(20) DEFAULT 'pending',  -- pending/syncing/success/failed
    sync_error TEXT,
    skill_count INTEGER DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_skill_registries_org_id ON skill_registries(organization_id);
CREATE INDEX idx_skill_registries_active ON skill_registries(is_active);

-- Unique indexes for repository_url per organization (NULL org = platform-level)
CREATE UNIQUE INDEX idx_skill_registries_unique_url
    ON skill_registries(organization_id, repository_url)
    WHERE organization_id IS NOT NULL;
CREATE UNIQUE INDEX idx_skill_registries_unique_url_platform
    ON skill_registries(repository_url)
    WHERE organization_id IS NULL;

-- ============================================================================
-- Table 2: skill_registry_overrides — Per-org overrides for platform registries
-- ============================================================================
CREATE TABLE IF NOT EXISTS skill_registry_overrides (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    registry_id BIGINT NOT NULL REFERENCES skill_registries(id) ON DELETE CASCADE,
    is_disabled BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(organization_id, registry_id)
);

CREATE INDEX idx_skill_registry_overrides_org ON skill_registry_overrides(organization_id);

-- ============================================================================
-- Table 3: skill_market_items — Skills marketplace (imported Skill metadata)
-- ============================================================================
CREATE TABLE IF NOT EXISTS skill_market_items (
    id BIGSERIAL PRIMARY KEY,
    registry_id BIGINT NOT NULL REFERENCES skill_registries(id) ON DELETE CASCADE,
    slug VARCHAR(100) NOT NULL,
    display_name VARCHAR(100),
    description VARCHAR(1024),
    license VARCHAR(100),
    compatibility VARCHAR(500),
    allowed_tools TEXT,
    metadata JSONB DEFAULT '{}',
    category VARCHAR(50),
    content_sha VARCHAR(64) NOT NULL,
    storage_key VARCHAR(500) NOT NULL,
    package_size BIGINT,
    version INTEGER DEFAULT 1,
    agent_type_filter JSONB DEFAULT '["claude-code"]',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(registry_id, slug)
);

CREATE INDEX idx_skill_market_items_registry ON skill_market_items(registry_id);
CREATE INDEX idx_skill_market_items_category ON skill_market_items(category);
CREATE INDEX idx_skill_market_items_active ON skill_market_items(is_active);

-- ============================================================================
-- Table 4: mcp_market_items — MCP Server template library
-- ============================================================================
CREATE TABLE IF NOT EXISTS mcp_market_items (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    icon VARCHAR(50),
    transport_type VARCHAR(20) DEFAULT 'stdio',
    command VARCHAR(500),
    default_args JSONB DEFAULT '[]',
    default_http_url VARCHAR(500),
    default_http_headers JSONB DEFAULT '[]',
    env_var_schema JSONB DEFAULT '[]',
    agent_type_filter JSONB,
    category VARCHAR(50),
    source VARCHAR(20) DEFAULT 'seed',  -- seed / registry / admin
    registry_name VARCHAR(200),
    version VARCHAR(50),
    repository_url VARCHAR(500),
    registry_meta JSONB DEFAULT '{}',
    last_synced_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Unique index on registry_name for upsert operations (only for non-null values)
CREATE UNIQUE INDEX IF NOT EXISTS idx_mcp_market_items_registry_name
    ON mcp_market_items(registry_name) WHERE registry_name IS NOT NULL;

-- ============================================================================
-- Table 5: installed_mcp_servers — MCP Server installation instances
-- ============================================================================
CREATE TABLE IF NOT EXISTS installed_mcp_servers (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    market_item_id BIGINT REFERENCES mcp_market_items(id) ON DELETE SET NULL,
    scope VARCHAR(20) NOT NULL CHECK (scope IN ('org', 'user')),
    installed_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    name VARCHAR(100),
    slug VARCHAR(100) NOT NULL,
    transport_type VARCHAR(20) DEFAULT 'stdio',
    command VARCHAR(500),
    args JSONB DEFAULT '[]',
    http_url VARCHAR(500),
    http_headers JSONB DEFAULT '{}',
    env_vars JSONB DEFAULT '{}',  -- encrypted storage
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_installed_mcp_servers_repo ON installed_mcp_servers(organization_id, repository_id);

-- Partial unique indexes to handle NULL installed_by correctly
CREATE UNIQUE INDEX idx_installed_mcp_servers_unique
    ON installed_mcp_servers(organization_id, repository_id, scope, installed_by, slug)
    WHERE installed_by IS NOT NULL;
CREATE UNIQUE INDEX idx_installed_mcp_servers_unique_no_user
    ON installed_mcp_servers(organization_id, repository_id, scope, slug)
    WHERE installed_by IS NULL;

-- ============================================================================
-- Table 6: installed_skills — Skills installation instances
-- ============================================================================
CREATE TABLE IF NOT EXISTS installed_skills (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    market_item_id BIGINT REFERENCES skill_market_items(id) ON DELETE SET NULL,
    scope VARCHAR(20) NOT NULL CHECK (scope IN ('org', 'user')),
    installed_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    slug VARCHAR(100) NOT NULL,
    install_source VARCHAR(20) NOT NULL,  -- market / github / upload
    source_url VARCHAR(500),
    content_sha VARCHAR(64),
    storage_key VARCHAR(500),
    package_size BIGINT,
    pinned_version INTEGER,
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_installed_skills_repo ON installed_skills(organization_id, repository_id);

-- Partial unique indexes to handle NULL installed_by correctly
CREATE UNIQUE INDEX idx_installed_skills_unique
    ON installed_skills(organization_id, repository_id, scope, installed_by, slug)
    WHERE installed_by IS NOT NULL;
CREATE UNIQUE INDEX idx_installed_skills_unique_no_user
    ON installed_skills(organization_id, repository_id, scope, slug)
    WHERE installed_by IS NULL;

-- ============================================================================
-- Seed: MCP Market items (commonly used MCP servers)
-- ============================================================================
INSERT INTO mcp_market_items (slug, name, description, icon, transport_type, command, default_args, env_var_schema, category) VALUES
('jira', 'Jira', 'Connect to Jira for issue tracking and project management', 'jira', 'stdio', 'npx', '["-y", "@modelcontextprotocol/server-jira"]'::jsonb, '[{"name": "JIRA_URL", "label": "Jira URL", "required": true, "placeholder": "https://your-domain.atlassian.net"}, {"name": "JIRA_EMAIL", "label": "Email", "required": true}, {"name": "JIRA_API_TOKEN", "label": "API Token", "required": true, "sensitive": true}]'::jsonb, 'productivity'),
('postgres', 'PostgreSQL', 'Query and manage PostgreSQL databases', 'database', 'stdio', 'npx', '["-y", "@modelcontextprotocol/server-postgres"]'::jsonb, '[{"name": "DATABASE_URL", "label": "Database URL", "required": true, "sensitive": true, "placeholder": "postgresql://user:pass@host:5432/db"}]'::jsonb, 'database'),
('slack', 'Slack', 'Send and read messages in Slack channels', 'slack', 'stdio', 'npx', '["-y", "@modelcontextprotocol/server-slack"]'::jsonb, '[{"name": "SLACK_BOT_TOKEN", "label": "Bot Token", "required": true, "sensitive": true}, {"name": "SLACK_TEAM_ID", "label": "Team ID", "required": true}]'::jsonb, 'communication'),
('github', 'GitHub', 'Interact with GitHub repositories, issues, and pull requests', 'github', 'stdio', 'npx', '["-y", "@modelcontextprotocol/server-github"]'::jsonb, '[{"name": "GITHUB_TOKEN", "label": "Personal Access Token", "required": true, "sensitive": true}]'::jsonb, 'development'),
('filesystem', 'Filesystem', 'Read and write files on the local filesystem', 'folder', 'stdio', 'npx', '["-y", "@modelcontextprotocol/server-filesystem", "/workspace"]'::jsonb, '[]'::jsonb, 'utility'),
('memory', 'Memory', 'Persistent memory storage for context across sessions', 'brain', 'stdio', 'npx', '["-y", "@modelcontextprotocol/server-memory"]'::jsonb, '[]'::jsonb, 'utility')
ON CONFLICT (slug) DO NOTHING;

-- ============================================================================
-- Update Claude Code agent type: clean up DB templates
-- Plugin dir and built-in Skills are now managed in Go code (ClaudeCodeBuilder),
-- not in DB templates. Remove mcp_enabled arg and clear files_template.
-- ============================================================================

-- Remove the mcp_enabled conditional arg from command_template
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
WHERE slug = 'claude-code'
  AND command_template->'args' IS NOT NULL;

-- Clear files_template — built-in Skills are now embedded in Go code
UPDATE agent_types SET
    files_template = '[]'::jsonb
WHERE slug = 'claude-code'
  AND files_template IS NOT NULL;
