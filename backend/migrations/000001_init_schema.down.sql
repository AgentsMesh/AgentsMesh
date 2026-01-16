-- AgentsMesh Database Schema Rollback
-- Migration: 000001_init_schema

-- Drop triggers first
DROP TRIGGER IF EXISTS update_subscriptions_updated_at ON subscriptions;
DROP TRIGGER IF EXISTS update_ticket_merge_requests_updated_at ON ticket_merge_requests;
DROP TRIGGER IF EXISTS update_tickets_updated_at ON tickets;
DROP TRIGGER IF EXISTS update_pod_bindings_updated_at ON pod_bindings;
DROP TRIGGER IF EXISTS update_channels_updated_at ON channels;
DROP TRIGGER IF EXISTS update_pods_updated_at ON pods;
DROP TRIGGER IF EXISTS update_runners_updated_at ON runners;
DROP TRIGGER IF EXISTS update_custom_agent_types_updated_at ON custom_agent_types;
DROP TRIGGER IF EXISTS update_user_agent_credentials_updated_at ON user_agent_credentials;
DROP TRIGGER IF EXISTS update_organization_agents_updated_at ON organization_agents;
DROP TRIGGER IF EXISTS update_agent_types_updated_at ON agent_types;
DROP TRIGGER IF EXISTS update_repositories_updated_at ON repositories;
DROP TRIGGER IF EXISTS update_git_providers_updated_at ON git_providers;
DROP TRIGGER IF EXISTS update_user_identities_updated_at ON user_identities;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_teams_updated_at ON teams;
DROP TRIGGER IF EXISTS update_organizations_updated_at ON organizations;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order of creation (respecting foreign key dependencies)

-- 7. Audit Logs
DROP TABLE IF EXISTS audit_logs;

-- 6. Billing Tables
DROP TABLE IF EXISTS usage_records;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS subscription_plans;

-- 5. Ticket Tables
DROP TABLE IF EXISTS ticket_merge_requests;
DROP TABLE IF EXISTS ticket_labels;
DROP TABLE IF EXISTS labels;
DROP TABLE IF EXISTS ticket_assignees;

-- Remove foreign key constraints before dropping tickets
ALTER TABLE IF EXISTS pods DROP CONSTRAINT IF EXISTS fk_pods_ticket;
ALTER TABLE IF EXISTS channels DROP CONSTRAINT IF EXISTS fk_channels_ticket;

DROP TABLE IF EXISTS tickets;

-- 4. AgentPod Tables
DROP TABLE IF EXISTS pod_bindings;
DROP TABLE IF EXISTS channel_messages;
DROP TABLE IF EXISTS channels;
DROP TABLE IF EXISTS pods;
DROP TABLE IF EXISTS runners;
DROP TABLE IF EXISTS runner_registration_tokens;

-- 3. Code Agent Configuration
DROP TABLE IF EXISTS custom_agent_types;
DROP TABLE IF EXISTS user_agent_credentials;
DROP TABLE IF EXISTS organization_agents;
DROP TABLE IF EXISTS agent_types;

-- 2. Git Provider Configuration
DROP TABLE IF EXISTS repositories;
DROP TABLE IF EXISTS git_providers;

-- 1. Multi-tenant Core Tables
DROP TABLE IF EXISTS team_members;
DROP TABLE IF EXISTS organization_members;
DROP TABLE IF EXISTS user_identities;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS organizations;
