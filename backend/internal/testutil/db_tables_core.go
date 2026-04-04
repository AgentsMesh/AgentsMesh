package testutil

// coreTableDDLs returns DDLs for users, organizations, agents, repositories.
func coreTableDDLs() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE, username TEXT NOT NULL UNIQUE, name TEXT,
			avatar_url TEXT, password_hash TEXT,
			is_active INTEGER NOT NULL DEFAULT 1, is_system_admin INTEGER NOT NULL DEFAULT 0,
			last_login_at DATETIME,
			is_email_verified INTEGER NOT NULL DEFAULT 0,
			email_verification_token TEXT, email_verification_expires_at DATETIME,
			password_reset_token TEXT, password_reset_expires_at DATETIME,
			default_git_credential_id INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS user_identities (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL, provider TEXT NOT NULL, provider_user_id TEXT NOT NULL,
			provider_username TEXT,
			access_token_encrypted TEXT, refresh_token_encrypted TEXT, token_expires_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS user_git_credentials (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL, name TEXT NOT NULL, credential_type TEXT NOT NULL,
			repository_provider_id INTEGER,
			pat_encrypted TEXT, public_key TEXT, private_key_encrypted TEXT,
			fingerprint TEXT, host_pattern TEXT,
			is_default INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS organizations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL, slug TEXT NOT NULL UNIQUE, logo_url TEXT,
			subscription_plan TEXT NOT NULL DEFAULT 'free',
			subscription_status TEXT NOT NULL DEFAULT 'active',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS organization_members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL, user_id INTEGER NOT NULL,
			role TEXT NOT NULL DEFAULT 'member',
			joined_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS teams (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL, name TEXT NOT NULL, description TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS team_members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			team_id INTEGER NOT NULL, user_id INTEGER NOT NULL, role TEXT NOT NULL DEFAULT 'member'
		)`,
		`CREATE TABLE IF NOT EXISTS agents (
			id INTEGER PRIMARY KEY, slug TEXT, name TEXT, launch_command TEXT, description TEXT,
			config_schema TEXT DEFAULT '{}', agentfile_source TEXT,
			supported_modes TEXT NOT NULL DEFAULT 'pty',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS repositories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			provider_type TEXT NOT NULL DEFAULT 'github', provider_base_url TEXT NOT NULL DEFAULT '',
			clone_url TEXT, http_clone_url TEXT, ssh_clone_url TEXT,
			external_id TEXT NOT NULL DEFAULT '', name TEXT NOT NULL, slug TEXT NOT NULL,
			default_branch TEXT NOT NULL DEFAULT 'main',
			ticket_prefix TEXT, visibility TEXT NOT NULL DEFAULT 'organization',
			imported_by_user_id INTEGER,
			preparation_script TEXT, preparation_timeout INTEGER DEFAULT 300,
			is_active INTEGER NOT NULL DEFAULT 1, webhook_config TEXT, deleted_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS git_providers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL, provider_type TEXT NOT NULL, name TEXT NOT NULL,
			base_url TEXT NOT NULL,
			client_id TEXT, client_secret_encrypted TEXT, bot_token_encrypted TEXT,
			ssh_key_id INTEGER, is_default INTEGER NOT NULL DEFAULT 0,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}
}
