package testkit

// supportTableDDLs returns DDLs for extensions, API keys, promo codes,
// support tickets, invitations, SSO configs, notifications, token usage,
// agent messages, and custom agents.
func supportTableDDLs() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS installed_mcp_servers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL, repository_id INTEGER NOT NULL,
			market_item_id INTEGER, scope TEXT NOT NULL DEFAULT 'org',
			installed_by INTEGER, name TEXT, slug TEXT NOT NULL,
			transport_type TEXT NOT NULL DEFAULT 'stdio',
			command TEXT, args BLOB,
			http_url TEXT, http_headers BLOB,
			env_vars BLOB,
			is_enabled INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS installed_skills (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL, repository_id INTEGER NOT NULL,
			market_item_id INTEGER, scope TEXT NOT NULL DEFAULT 'org',
			installed_by INTEGER, slug TEXT NOT NULL,
			install_source TEXT NOT NULL DEFAULT 'market',
			source_url TEXT, content_sha TEXT, storage_key TEXT,
			package_size INTEGER NOT NULL DEFAULT 0, pinned_version INTEGER,
			is_enabled INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS skill_registries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER, repository_url TEXT NOT NULL,
			branch TEXT DEFAULT 'main', source_type TEXT DEFAULT 'auto',
			detected_type TEXT, compatible_agents BLOB,
			auth_type TEXT DEFAULT 'none', auth_credential TEXT,
			last_synced_at DATETIME, last_commit_sha TEXT,
			sync_status TEXT DEFAULT 'pending', sync_error TEXT,
			skill_count INTEGER DEFAULT 0,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS skill_market_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			registry_id INTEGER NOT NULL, slug TEXT NOT NULL,
			display_name TEXT, description TEXT, license TEXT,
			compatibility TEXT, allowed_tools TEXT, metadata BLOB,
			category TEXT, content_sha TEXT NOT NULL, storage_key TEXT NOT NULL,
			package_size INTEGER DEFAULT 0, version INTEGER DEFAULT 1,
			agent_filter BLOB,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS mcp_market_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE, name TEXT NOT NULL, description TEXT,
			icon TEXT, transport_type TEXT DEFAULT 'stdio',
			command TEXT, default_args BLOB,
			default_http_url TEXT, default_http_headers BLOB,
			env_var_schema BLOB, agent_filter BLOB,
			category TEXT, is_active INTEGER NOT NULL DEFAULT 1,
			source TEXT DEFAULT 'seed', registry_name TEXT,
			version TEXT, repository_url TEXT, registry_meta BLOB,
			last_synced_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS skill_registry_overrides (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL, registry_id INTEGER NOT NULL,
			is_disabled INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(organization_id, registry_id)
		)`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL, name TEXT NOT NULL, description TEXT,
			key_prefix TEXT NOT NULL, key_hash TEXT NOT NULL UNIQUE,
			scopes TEXT NOT NULL DEFAULT '[]',
			is_enabled INTEGER NOT NULL DEFAULT 1,
			expires_at DATETIME, last_used_at DATETIME, created_by INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS promo_codes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE, name TEXT NOT NULL, description TEXT,
			type TEXT NOT NULL, plan_name TEXT NOT NULL,
			duration_months INTEGER NOT NULL,
			max_uses INTEGER, used_count INTEGER NOT NULL DEFAULT 0,
			max_uses_per_org INTEGER NOT NULL DEFAULT 1,
			starts_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME, is_active INTEGER NOT NULL DEFAULT 1,
			created_by_id INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS promo_code_redemptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			promo_code_id INTEGER NOT NULL, organization_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL, plan_name TEXT NOT NULL,
			duration_months INTEGER NOT NULL,
			previous_plan_name TEXT, previous_period_end DATETIME,
			new_period_end DATETIME NOT NULL, ip_address TEXT, user_agent TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS support_tickets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL, title TEXT NOT NULL,
			category TEXT NOT NULL DEFAULT 'other',
			status TEXT NOT NULL DEFAULT 'open',
			priority TEXT NOT NULL DEFAULT 'medium',
			assigned_admin_id INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			resolved_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS support_ticket_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_id INTEGER NOT NULL, user_id INTEGER NOT NULL,
			content TEXT NOT NULL, is_admin_reply INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS support_ticket_attachments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_id INTEGER NOT NULL, message_id INTEGER,
			uploader_id INTEGER NOT NULL, original_name TEXT NOT NULL,
			storage_key TEXT NOT NULL, mime_type TEXT NOT NULL, size INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS invitations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL, email TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'member',
			token TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'pending',
			invited_by INTEGER NOT NULL DEFAULT 0, expires_at DATETIME NOT NULL,
			accepted_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS sso_configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			domain TEXT NOT NULL, name TEXT NOT NULL, protocol TEXT NOT NULL,
			is_enabled INTEGER NOT NULL DEFAULT 0,
			enforce_sso INTEGER NOT NULL DEFAULT 0,
			oidc_issuer_url TEXT, oidc_client_id TEXT,
			oidc_client_secret_encrypted TEXT, oidc_scopes TEXT,
			saml_idp_metadata_url TEXT, saml_idp_metadata_xml TEXT,
			saml_idp_sso_url TEXT, saml_idp_cert_encrypted TEXT,
			saml_sp_entity_id TEXT, saml_name_id_format TEXT,
			ldap_host TEXT, ldap_port INTEGER, ldap_use_tls INTEGER,
			ldap_bind_dn TEXT, ldap_bind_password_encrypted TEXT,
			ldap_base_dn TEXT, ldap_user_filter TEXT,
			ldap_email_attr TEXT, ldap_name_attr TEXT, ldap_username_attr TEXT,
			created_by INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_sso_configs_domain_protocol
		 ON sso_configs (domain, protocol)`,
		`CREATE TABLE IF NOT EXISTS notification_preferences (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL, source TEXT NOT NULL, entity_id TEXT NOT NULL DEFAULT '',
			is_muted INTEGER NOT NULL DEFAULT 0, channels TEXT DEFAULT '{}',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, source, entity_id)
		)`,
		`CREATE TABLE IF NOT EXISTS token_usages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL, pod_id INTEGER, pod_key TEXT,
			user_id INTEGER, runner_id INTEGER, agent_slug TEXT, model TEXT,
			input_tokens INTEGER NOT NULL DEFAULT 0,
			output_tokens INTEGER NOT NULL DEFAULT 0,
			cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
			cache_read_tokens INTEGER NOT NULL DEFAULT 0,
			session_started_at DATETIME, session_ended_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_admin_audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			admin_user_id INTEGER NOT NULL, action TEXT NOT NULL,
			target_type TEXT NOT NULL, target_id TEXT,
			old_data TEXT, new_data TEXT,
			ip_address TEXT, user_agent TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS custom_agents (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL, slug TEXT NOT NULL,
			name TEXT NOT NULL, description TEXT,
			launch_command TEXT NOT NULL, default_args TEXT,
			agentfile_source TEXT,
			is_active INTEGER NOT NULL DEFAULT 1,
			supported_modes TEXT NOT NULL DEFAULT 'pty',
			created_by_id INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(organization_id, slug)
		)`,
		`CREATE TABLE IF NOT EXISTS user_agent_credential_profiles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL, name TEXT NOT NULL, agent_slug TEXT NOT NULL,
			description TEXT, is_runner_host INTEGER NOT NULL DEFAULT 0,
			credentials_encrypted TEXT,
			encrypted_credentials TEXT NOT NULL DEFAULT '{}',
			is_default INTEGER NOT NULL DEFAULT 0,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS user_agent_configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL, agent_slug TEXT NOT NULL,
			config_values BLOB NOT NULL DEFAULT '{}',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, agent_slug)
		)`,
		`CREATE TABLE IF NOT EXISTS agent_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sender_pod TEXT NOT NULL, receiver_pod TEXT NOT NULL,
			message_type TEXT NOT NULL, content TEXT NOT NULL DEFAULT '{}',
			status TEXT NOT NULL DEFAULT 'pending',
			delivery_attempts INTEGER NOT NULL DEFAULT 0,
			max_retries INTEGER NOT NULL DEFAULT 3,
			last_delivery_attempt DATETIME, next_retry_at DATETIME,
			delivery_error TEXT,
			delivered_at DATETIME, read_at DATETIME,
			parent_message_id INTEGER, correlation_id TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			actor_id INTEGER, actor_type TEXT NOT NULL,
			action TEXT NOT NULL, resource_type TEXT NOT NULL, resource_id INTEGER,
			details TEXT, ip_address TEXT, user_agent TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS agent_message_dead_letters (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			original_message_id INTEGER NOT NULL UNIQUE,
			reason TEXT NOT NULL, final_attempt INTEGER NOT NULL,
			moved_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			replayed_at DATETIME, replay_result TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}
}
