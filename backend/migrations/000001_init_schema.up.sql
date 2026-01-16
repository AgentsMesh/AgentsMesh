-- AgentsMesh Database Schema
-- Migration: 000001_init_schema

-- ==========================================
-- 1. Multi-tenant Core Tables
-- ==========================================

-- Organizations (Tenants)
CREATE TABLE organizations (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    logo_url TEXT,

    -- Subscription info
    subscription_plan VARCHAR(50) NOT NULL DEFAULT 'free',
    subscription_status VARCHAR(20) NOT NULL DEFAULT 'active',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_organizations_slug ON organizations(slug);

-- Teams
CREATE TABLE teams (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(organization_id, name)
);

CREATE INDEX idx_teams_organization ON teams(organization_id);

-- Users (Global unique)
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    username VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255),
    avatar_url TEXT,
    password_hash VARCHAR(255),

    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);

-- User External Identities (OAuth)
CREATE TABLE user_identities (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    provider VARCHAR(50) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    provider_username VARCHAR(255),
    access_token_encrypted TEXT,
    refresh_token_encrypted TEXT,
    token_expires_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(provider, provider_user_id)
);

CREATE INDEX idx_user_identities_user ON user_identities(user_id);
CREATE INDEX idx_user_identities_provider ON user_identities(provider, provider_user_id);

-- Organization Members
CREATE TABLE organization_members (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    role VARCHAR(50) NOT NULL DEFAULT 'member',

    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(organization_id, user_id)
);

CREATE INDEX idx_org_members_org ON organization_members(organization_id);
CREATE INDEX idx_org_members_user ON organization_members(user_id);

-- Team Members
CREATE TABLE team_members (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    role VARCHAR(50) NOT NULL DEFAULT 'member',

    UNIQUE(team_id, user_id)
);

CREATE INDEX idx_team_members_team ON team_members(team_id);
CREATE INDEX idx_team_members_user ON team_members(user_id);

-- ==========================================
-- 2. Git Provider Configuration
-- ==========================================

-- Git Providers (Organization level)
CREATE TABLE git_providers (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    provider_type VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    base_url VARCHAR(255) NOT NULL,

    client_id VARCHAR(255),
    client_secret_encrypted TEXT,
    bot_token_encrypted TEXT,

    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(organization_id, name)
);

CREATE INDEX idx_git_providers_org ON git_providers(organization_id);

-- Repositories
CREATE TABLE repositories (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    team_id BIGINT REFERENCES teams(id) ON DELETE SET NULL,
    git_provider_id BIGINT NOT NULL REFERENCES git_providers(id) ON DELETE CASCADE,

    external_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    full_path VARCHAR(500) NOT NULL,
    default_branch VARCHAR(100) DEFAULT 'main',

    ticket_prefix VARCHAR(10),

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(git_provider_id, external_id)
);

CREATE INDEX idx_repositories_org ON repositories(organization_id);
CREATE INDEX idx_repositories_team ON repositories(team_id);

-- ==========================================
-- 3. Code Agent Configuration
-- ==========================================

-- Agent Types (System predefined + Custom)
CREATE TABLE agent_types (
    id BIGSERIAL PRIMARY KEY,

    slug VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    description TEXT,

    launch_command VARCHAR(500) NOT NULL,
    default_args TEXT,

    credential_schema JSONB NOT NULL DEFAULT '[]',
    status_detection JSONB,

    is_builtin BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert builtin agent types
INSERT INTO agent_types (slug, name, launch_command, credential_schema, is_builtin) VALUES
('claude-code', 'Claude Code', 'claude', '[{"name":"api_key","type":"secret","env_var":"ANTHROPIC_API_KEY","required":true}]', true),
('codex-cli', 'OpenAI Codex', 'codex', '[{"name":"api_key","type":"secret","env_var":"OPENAI_API_KEY","required":true}]', true),
('gemini-cli', 'Gemini CLI', 'gemini', '[{"name":"api_key","type":"secret","env_var":"GOOGLE_API_KEY","required":true}]', true),
('aider', 'Aider', 'aider', '[{"name":"api_key","type":"secret","env_var":"OPENAI_API_KEY","required":false}]', true),
('opencode', 'OpenCode', 'opencode', '[{"name":"api_key","type":"secret","env_var":"OPENAI_API_KEY","required":false}]', true);

-- Organization Agent Configuration
CREATE TABLE organization_agents (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    agent_type_id BIGINT NOT NULL REFERENCES agent_types(id) ON DELETE CASCADE,

    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,

    credentials_encrypted JSONB,
    custom_launch_args TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(organization_id, agent_type_id)
);

CREATE INDEX idx_org_agents_org ON organization_agents(organization_id);

-- User Agent Credentials
CREATE TABLE user_agent_credentials (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    agent_type_id BIGINT NOT NULL REFERENCES agent_types(id) ON DELETE CASCADE,

    credentials_encrypted JSONB NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(user_id, agent_type_id)
);

CREATE INDEX idx_user_agent_creds_user ON user_agent_credentials(user_id);

-- Custom Agent Types (Organization level)
CREATE TABLE custom_agent_types (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    slug VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,

    launch_command VARCHAR(500) NOT NULL,
    default_args TEXT,
    credential_schema JSONB NOT NULL DEFAULT '[]',
    status_detection JSONB,

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(organization_id, slug)
);

CREATE INDEX idx_custom_agents_org ON custom_agent_types(organization_id);

-- ==========================================
-- 4. AgentPod Tables
-- ==========================================

-- Runner Registration Tokens
CREATE TABLE runner_registration_tokens (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    token_hash VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,

    created_by_id BIGINT NOT NULL REFERENCES users(id),

    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    max_uses INT,
    used_count INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reg_tokens_org ON runner_registration_tokens(organization_id);
CREATE INDEX idx_reg_tokens_hash ON runner_registration_tokens(token_hash);

-- Runners
CREATE TABLE runners (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    node_id VARCHAR(100) NOT NULL,
    description TEXT,
    auth_token_hash VARCHAR(255) NOT NULL,

    status VARCHAR(50) NOT NULL DEFAULT 'offline',
    last_heartbeat TIMESTAMPTZ,
    current_pods INT NOT NULL DEFAULT 0,
    max_concurrent_pods INT NOT NULL DEFAULT 5,
    runner_version VARCHAR(50),

    host_info JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(organization_id, node_id)
);

CREATE INDEX idx_runners_org ON runners(organization_id);
CREATE INDEX idx_runners_status ON runners(status);

-- Pods (AgentPod instances)
CREATE TABLE pods (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    team_id BIGINT REFERENCES teams(id) ON DELETE SET NULL,

    pod_key VARCHAR(100) NOT NULL UNIQUE,
    runner_id BIGINT NOT NULL REFERENCES runners(id) ON DELETE CASCADE,

    agent_type_id BIGINT REFERENCES agent_types(id),
    custom_agent_type_id BIGINT REFERENCES custom_agent_types(id),

    repository_id BIGINT REFERENCES repositories(id) ON DELETE SET NULL,
    ticket_id BIGINT, -- Will reference tickets table, added later to avoid circular dependency
    created_by_id BIGINT NOT NULL REFERENCES users(id),

    pty_pid INT,
    status VARCHAR(50) NOT NULL DEFAULT 'initializing',
    agent_status VARCHAR(50) NOT NULL DEFAULT 'unknown',

    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    last_activity TIMESTAMPTZ,

    initial_prompt TEXT,
    branch_name VARCHAR(255),
    worktree_path VARCHAR(500),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pods_org ON pods(organization_id);
CREATE INDEX idx_pods_team ON pods(team_id);
CREATE INDEX idx_pods_runner ON pods(runner_id);
CREATE INDEX idx_pods_key ON pods(pod_key);
CREATE INDEX idx_pods_status ON pods(status);

-- Channels
CREATE TABLE channels (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    team_id BIGINT REFERENCES teams(id) ON DELETE SET NULL,

    name VARCHAR(100) NOT NULL,
    description TEXT,
    document TEXT,

    repository_id BIGINT REFERENCES repositories(id) ON DELETE SET NULL,
    ticket_id BIGINT, -- Will reference tickets table

    created_by_pod VARCHAR(100),
    created_by_user_id BIGINT REFERENCES users(id),

    is_archived BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(organization_id, name)
);

CREATE INDEX idx_channels_org ON channels(organization_id);
CREATE INDEX idx_channels_team ON channels(team_id);

-- Channel Messages
CREATE TABLE channel_messages (
    id BIGSERIAL PRIMARY KEY,
    channel_id BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,

    sender_pod VARCHAR(100),
    sender_user_id BIGINT REFERENCES users(id),

    message_type VARCHAR(50) NOT NULL DEFAULT 'text',
    content TEXT NOT NULL,
    metadata JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_channel_messages_channel ON channel_messages(channel_id);
CREATE INDEX idx_channel_messages_time ON channel_messages(created_at);

-- Pod Bindings
CREATE TABLE pod_bindings (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    initiator_pod VARCHAR(100) NOT NULL,
    target_pod VARCHAR(100) NOT NULL,

    granted_scopes TEXT[],
    status VARCHAR(50) NOT NULL DEFAULT 'pending',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(initiator_pod, target_pod)
);

CREATE INDEX idx_pod_bindings_org ON pod_bindings(organization_id);
CREATE INDEX idx_pod_bindings_initiator ON pod_bindings(initiator_pod);
CREATE INDEX idx_pod_bindings_target ON pod_bindings(target_pod);

-- ==========================================
-- 5. Ticket Tables
-- ==========================================

-- Tickets
CREATE TABLE tickets (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    team_id BIGINT REFERENCES teams(id) ON DELETE SET NULL,

    number INT NOT NULL,
    identifier VARCHAR(50) NOT NULL UNIQUE,

    type VARCHAR(50) NOT NULL DEFAULT 'task',
    title VARCHAR(500) NOT NULL,
    description TEXT,
    content TEXT,

    status VARCHAR(50) NOT NULL DEFAULT 'backlog',
    priority VARCHAR(50) NOT NULL DEFAULT 'none',

    due_date TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    repository_id BIGINT REFERENCES repositories(id) ON DELETE CASCADE,
    reporter_id BIGINT NOT NULL REFERENCES users(id),
    parent_ticket_id BIGINT REFERENCES tickets(id) ON DELETE CASCADE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(repository_id, number)
);

CREATE INDEX idx_tickets_org ON tickets(organization_id);
CREATE INDEX idx_tickets_team ON tickets(team_id);
CREATE INDEX idx_tickets_repo ON tickets(repository_id);
CREATE INDEX idx_tickets_status ON tickets(status);
CREATE INDEX idx_tickets_identifier ON tickets(identifier);

-- Add foreign key for pods.ticket_id
ALTER TABLE pods ADD CONSTRAINT fk_pods_ticket
    FOREIGN KEY (ticket_id) REFERENCES tickets(id) ON DELETE SET NULL;

-- Add foreign key for channels.ticket_id
ALTER TABLE channels ADD CONSTRAINT fk_channels_ticket
    FOREIGN KEY (ticket_id) REFERENCES tickets(id) ON DELETE SET NULL;

-- Ticket Assignees
CREATE TABLE ticket_assignees (
    ticket_id BIGINT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY(ticket_id, user_id)
);

CREATE INDEX idx_ticket_assignees_ticket ON ticket_assignees(ticket_id);
CREATE INDEX idx_ticket_assignees_user ON ticket_assignees(user_id);

-- Labels
CREATE TABLE labels (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    repository_id BIGINT REFERENCES repositories(id) ON DELETE CASCADE,

    name VARCHAR(100) NOT NULL,
    color VARCHAR(7) NOT NULL DEFAULT '#6B7280',

    UNIQUE(organization_id, repository_id, name)
);

CREATE INDEX idx_labels_org ON labels(organization_id);
CREATE INDEX idx_labels_repo ON labels(repository_id);

-- Ticket Labels (Many-to-Many)
CREATE TABLE ticket_labels (
    ticket_id BIGINT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    label_id BIGINT NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
    PRIMARY KEY(ticket_id, label_id)
);

-- Ticket Merge Requests
CREATE TABLE ticket_merge_requests (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    ticket_id BIGINT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    pod_id BIGINT REFERENCES pods(id) ON DELETE SET NULL,

    mr_iid INT NOT NULL,
    mr_url TEXT NOT NULL UNIQUE,
    source_branch VARCHAR(255) NOT NULL,
    target_branch VARCHAR(255) NOT NULL DEFAULT 'main',
    title VARCHAR(500),
    state VARCHAR(50) NOT NULL DEFAULT 'opened',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ticket_mrs_org ON ticket_merge_requests(organization_id);
CREATE INDEX idx_ticket_mrs_ticket ON ticket_merge_requests(ticket_id);

-- ==========================================
-- 6. Billing Tables
-- ==========================================

-- Subscription Plans
CREATE TABLE subscription_plans (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,

    price_per_seat_monthly DECIMAL(10, 2) NOT NULL DEFAULT 0,
    included_pod_minutes INT NOT NULL DEFAULT 0,
    price_per_extra_minute DECIMAL(10, 4) NOT NULL DEFAULT 0,

    max_users INT NOT NULL,
    max_runners INT NOT NULL,
    max_repositories INT NOT NULL,

    features JSONB NOT NULL DEFAULT '{}',

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert default plans
INSERT INTO subscription_plans (name, display_name, max_users, max_runners, max_repositories, included_pod_minutes) VALUES
('free', 'Free', 3, 1, 3, 100),
('pro', 'Pro', 10, 5, 20, 1000),
('enterprise', 'Enterprise', -1, -1, -1, -1);

-- Subscriptions
CREATE TABLE subscriptions (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL UNIQUE REFERENCES organizations(id) ON DELETE CASCADE,
    plan_id BIGINT NOT NULL REFERENCES subscription_plans(id),

    status VARCHAR(50) NOT NULL DEFAULT 'active',
    billing_cycle VARCHAR(20) NOT NULL DEFAULT 'monthly',

    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end TIMESTAMPTZ NOT NULL,

    stripe_customer_id VARCHAR(255),
    stripe_subscription_id VARCHAR(255),

    custom_quotas JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_org ON subscriptions(organization_id);

-- Usage Records
CREATE TABLE usage_records (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    usage_type VARCHAR(50) NOT NULL,
    quantity DECIMAL(10, 2) NOT NULL,

    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,

    metadata JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_usage_org_period ON usage_records(organization_id, period_start, period_end);
CREATE INDEX idx_usage_type ON usage_records(usage_type);

-- ==========================================
-- 7. Audit Logs
-- ==========================================

CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT REFERENCES organizations(id) ON DELETE SET NULL,

    actor_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    actor_type VARCHAR(50) NOT NULL,

    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id BIGINT,

    details JSONB,
    ip_address INET,
    user_agent TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_org_time ON audit_logs(organization_id, created_at);
CREATE INDEX idx_audit_action ON audit_logs(action);
CREATE INDEX idx_audit_resource ON audit_logs(resource_type, resource_id);

-- ==========================================
-- 8. Trigger for updated_at
-- ==========================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply trigger to all tables with updated_at column
CREATE TRIGGER update_organizations_updated_at BEFORE UPDATE ON organizations FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_teams_updated_at BEFORE UPDATE ON teams FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_user_identities_updated_at BEFORE UPDATE ON user_identities FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_git_providers_updated_at BEFORE UPDATE ON git_providers FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_repositories_updated_at BEFORE UPDATE ON repositories FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_agent_types_updated_at BEFORE UPDATE ON agent_types FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_organization_agents_updated_at BEFORE UPDATE ON organization_agents FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_user_agent_credentials_updated_at BEFORE UPDATE ON user_agent_credentials FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_custom_agent_types_updated_at BEFORE UPDATE ON custom_agent_types FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_runners_updated_at BEFORE UPDATE ON runners FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_pods_updated_at BEFORE UPDATE ON pods FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_channels_updated_at BEFORE UPDATE ON channels FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_pod_bindings_updated_at BEFORE UPDATE ON pod_bindings FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_tickets_updated_at BEFORE UPDATE ON tickets FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_ticket_merge_requests_updated_at BEFORE UPDATE ON ticket_merge_requests FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_subscriptions_updated_at BEFORE UPDATE ON subscriptions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
