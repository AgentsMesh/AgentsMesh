package testkit

// runnerTableDDLs returns DDLs for runners, certificates, registration tokens.
func runnerTableDDLs() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS runners (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL, node_id TEXT NOT NULL DEFAULT '',
			name TEXT,
			description TEXT, status TEXT NOT NULL DEFAULT 'offline',
			last_heartbeat DATETIME,
			current_pods INTEGER NOT NULL DEFAULT 0, max_concurrent_pods INTEGER NOT NULL DEFAULT 5,
			runner_version TEXT,
			is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
			host_info BLOB, available_agents BLOB DEFAULT '[]', agent_versions BLOB DEFAULT '[]',
			visibility TEXT NOT NULL DEFAULT 'organization',
			registered_by_user_id INTEGER,
			cert_serial_number TEXT, cert_fingerprint TEXT, cert_expires_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_runners_organization_id ON runners(organization_id)`,
		`CREATE INDEX IF NOT EXISTS idx_runners_status ON runners(status)`,
		`CREATE TABLE IF NOT EXISTS runner_pending_auths (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			auth_key TEXT NOT NULL UNIQUE, machine_key TEXT NOT NULL,
			node_id TEXT, labels TEXT,
			authorized BOOLEAN NOT NULL DEFAULT FALSE,
			organization_id INTEGER, runner_id INTEGER,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS runner_grpc_registration_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			token_hash TEXT NOT NULL UNIQUE, organization_id INTEGER NOT NULL,
			name TEXT, labels TEXT,
			single_use BOOLEAN NOT NULL DEFAULT TRUE,
			max_uses INTEGER NOT NULL DEFAULT 1, used_count INTEGER NOT NULL DEFAULT 0,
			expires_at DATETIME NOT NULL, created_by INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS runner_certificates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			runner_id INTEGER NOT NULL, serial_number TEXT NOT NULL UNIQUE,
			fingerprint TEXT NOT NULL,
			issued_at DATETIME NOT NULL, expires_at DATETIME NOT NULL,
			revoked_at DATETIME, revocation_reason TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS runner_reactivation_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			runner_id INTEGER NOT NULL, token_hash TEXT NOT NULL UNIQUE,
			expires_at DATETIME NOT NULL, used_at DATETIME, created_by INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS runner_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL, runner_id INTEGER NOT NULL,
			request_id TEXT NOT NULL UNIQUE, storage_key TEXT,
			status TEXT NOT NULL DEFAULT 'pending', size_bytes INTEGER,
			error_message TEXT, requested_by_id INTEGER,
			completed_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}
}

// podTableDDLs returns DDLs for pods, autopilot controllers/iterations, AI providers.
func podTableDDLs() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS pods (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL DEFAULT 0, pod_key TEXT NOT NULL UNIQUE,
			runner_id INTEGER NOT NULL DEFAULT 0,
			agent_slug TEXT, custom_agent_slug TEXT,
			repository_id INTEGER, ticket_id INTEGER,
			created_by_id INTEGER NOT NULL DEFAULT 0,
			pty_p_id INTEGER, pty_pid INTEGER,
			status TEXT NOT NULL DEFAULT 'initializing',
			agent_status TEXT NOT NULL DEFAULT 'idle',
			agent_p_id INTEGER, agent_pid INTEGER,
			started_at DATETIME, finished_at DATETIME,
			last_activity DATETIME, agent_waiting_since DATETIME,
			prompt TEXT, branch_name TEXT, sandbox_path TEXT,
			model TEXT, permission_mode TEXT, think_level TEXT,
			error_code TEXT, error_message TEXT, title TEXT, alias TEXT,
			session_id TEXT, source_pod_key TEXT,
			credential_profile_id INTEGER,
			config_overrides TEXT DEFAULT '{}',
			interaction_mode TEXT NOT NULL DEFAULT 'pty',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS autopilot_controllers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL DEFAULT 0,
			autopilot_controller_key TEXT NOT NULL UNIQUE,
			pod_key TEXT NOT NULL DEFAULT '',
			pod_id INTEGER NOT NULL DEFAULT 0,
			runner_id INTEGER NOT NULL DEFAULT 0,
			prompt TEXT,
			phase TEXT NOT NULL DEFAULT 'initializing',
			current_iteration INTEGER NOT NULL DEFAULT 0,
			max_iterations INTEGER NOT NULL DEFAULT 10,
			iteration_timeout_sec INTEGER NOT NULL DEFAULT 300,
			circuit_breaker_state TEXT NOT NULL DEFAULT 'closed',
			circuit_breaker_reason TEXT,
			no_progress_threshold INTEGER NOT NULL DEFAULT 3,
			same_error_threshold INTEGER NOT NULL DEFAULT 5,
			approval_timeout_min INTEGER NOT NULL DEFAULT 30,
			control_agent_slug TEXT,
			control_prompt_template TEXT,
			mcp_config_json TEXT,
			user_takeover INTEGER NOT NULL DEFAULT 0,
			started_at DATETIME,
			last_iteration_at DATETIME,
			completed_at DATETIME,
			approval_request_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS autopilot_iterations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			autopilot_controller_id INTEGER NOT NULL,
			iteration INTEGER NOT NULL,
			phase TEXT NOT NULL,
			summary TEXT, files_changed TEXT, error_message TEXT,
			duration_ms INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_autopilot_iterations_controller
		 ON autopilot_iterations(autopilot_controller_id)`,
		`CREATE TABLE IF NOT EXISTS user_ai_providers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL, provider_type TEXT NOT NULL, name TEXT NOT NULL,
			is_default INTEGER NOT NULL DEFAULT 0, is_enabled INTEGER NOT NULL DEFAULT 1,
			encrypted_credentials TEXT NOT NULL, last_used_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS user_agentpod_settings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			default_agent_slug TEXT,
			default_model TEXT,
			default_perm_mode TEXT,
			terminal_font_size INTEGER,
			terminal_theme TEXT,
			default_cols INTEGER NOT NULL DEFAULT 120, default_rows INTEGER NOT NULL DEFAULT 40,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}
}
