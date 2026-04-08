package testkit

// loopTableDDLs returns DDLs for loops and loop runs.
func loopTableDDLs() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS loops (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL, name TEXT NOT NULL, slug TEXT NOT NULL,
			description TEXT, agent_slug TEXT, custom_agent_slug TEXT,
			permission_mode TEXT NOT NULL DEFAULT 'bypassPermissions',
			prompt_template TEXT NOT NULL DEFAULT '',
			repository_id INTEGER, runner_id INTEGER, branch_name TEXT,
			ticket_id INTEGER, credential_profile_id INTEGER,
			config_overrides BLOB DEFAULT NULL, prompt_variables BLOB DEFAULT NULL,
			execution_mode TEXT NOT NULL DEFAULT 'autopilot',
			cron_expression TEXT, autopilot_config BLOB DEFAULT NULL,
			callback_url TEXT, status TEXT NOT NULL DEFAULT 'enabled',
			sandbox_strategy TEXT NOT NULL DEFAULT 'persistent',
			session_persistence INTEGER NOT NULL DEFAULT 1,
			concurrency_policy TEXT NOT NULL DEFAULT 'skip',
			max_concurrent_runs INTEGER NOT NULL DEFAULT 1,
			max_retained_runs INTEGER NOT NULL DEFAULT 0,
			timeout_minutes INTEGER NOT NULL DEFAULT 60,
			idle_timeout_sec INTEGER NOT NULL DEFAULT 30,
			sandbox_path TEXT, last_pod_key TEXT,
			created_by_id INTEGER NOT NULL DEFAULT 0,
			total_runs INTEGER NOT NULL DEFAULT 0,
			successful_runs INTEGER NOT NULL DEFAULT 0,
			failed_runs INTEGER NOT NULL DEFAULT 0,
			last_run_at DATETIME, next_run_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(organization_id, slug)
		)`,
		`CREATE TABLE IF NOT EXISTS loop_runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL, loop_id INTEGER NOT NULL,
			run_number INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			pod_key TEXT, autopilot_controller_key TEXT,
			trigger_type TEXT NOT NULL DEFAULT 'manual', trigger_source TEXT,
			trigger_params BLOB DEFAULT NULL, resolved_prompt TEXT,
			started_at DATETIME, finished_at DATETIME, duration_sec INTEGER,
			exit_summary TEXT, error_message TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}
}

// billingTableDDLs returns DDLs for plans, subscriptions, payments, invoices, licenses.
func billingTableDDLs() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS subscription_plans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE, display_name TEXT NOT NULL,
			price_per_seat_monthly REAL NOT NULL DEFAULT 0,
			price_per_seat_yearly REAL NOT NULL DEFAULT 0,
			included_pod_minutes INTEGER NOT NULL DEFAULT 0,
			price_per_extra_minute REAL NOT NULL DEFAULT 0,
			max_users INTEGER NOT NULL DEFAULT 0, max_runners INTEGER NOT NULL DEFAULT 0,
			max_concurrent_pods INTEGER NOT NULL DEFAULT 0,
			max_repositories INTEGER NOT NULL DEFAULT 0,
			features TEXT,
			stripe_price_id_monthly TEXT, stripe_price_id_yearly TEXT,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS subscriptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL UNIQUE, plan_id INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			billing_cycle TEXT NOT NULL DEFAULT 'monthly',
			current_period_start DATETIME NOT NULL, current_period_end DATETIME NOT NULL,
			payment_provider TEXT, payment_method TEXT,
			auto_renew INTEGER NOT NULL DEFAULT 0, seat_count INTEGER NOT NULL DEFAULT 1,
			stripe_customer_id TEXT, stripe_subscription_id TEXT,
			lemonsqueezy_customer_id TEXT, lemonsqueezy_subscription_id TEXT,
			alipay_agreement_no TEXT, wechat_contract_id TEXT,
			canceled_at DATETIME, cancel_at_period_end INTEGER NOT NULL DEFAULT 0,
			frozen_at DATETIME, downgrade_to_plan TEXT, next_billing_cycle TEXT,
			custom_quotas TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS usage_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL, usage_type TEXT NOT NULL,
			quantity REAL NOT NULL DEFAULT 0,
			period_start DATETIME NOT NULL, period_end DATETIME NOT NULL,
			metadata TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS payment_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL, order_no TEXT NOT NULL UNIQUE,
			external_order_no TEXT, order_type TEXT NOT NULL,
			plan_id INTEGER, billing_cycle TEXT, seats INTEGER DEFAULT 1,
			currency TEXT NOT NULL DEFAULT 'USD', amount REAL NOT NULL,
			discount_amount REAL DEFAULT 0, actual_amount REAL NOT NULL,
			payment_provider TEXT NOT NULL, payment_method TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			metadata TEXT, failure_reason TEXT, idempotency_key TEXT UNIQUE,
			expires_at DATETIME, paid_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_by_id INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS invoices (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL, payment_order_id INTEGER,
			invoice_no TEXT NOT NULL UNIQUE, status TEXT NOT NULL DEFAULT 'draft',
			currency TEXT NOT NULL DEFAULT 'USD',
			subtotal REAL NOT NULL, tax_amount REAL DEFAULT 0, total REAL NOT NULL,
			billing_name TEXT, billing_email TEXT, billing_address TEXT,
			period_start DATETIME NOT NULL, period_end DATETIME NOT NULL,
			line_items TEXT NOT NULL DEFAULT '[]', pdf_url TEXT,
			issued_at DATETIME, due_at DATETIME, paid_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS plan_prices (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			plan_id INTEGER NOT NULL, currency TEXT NOT NULL,
			price_monthly REAL NOT NULL, price_yearly REAL NOT NULL,
			stripe_price_id_monthly TEXT, stripe_price_id_yearly TEXT,
			lemonsqueezy_variant_id_monthly TEXT, lemonsqueezy_variant_id_yearly TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(plan_id, currency)
		)`,
		`CREATE TABLE IF NOT EXISTS webhook_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id TEXT NOT NULL, provider TEXT NOT NULL, event_type TEXT NOT NULL,
			processed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(event_id, provider)
		)`,
		`CREATE TABLE IF NOT EXISTS payment_transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			payment_order_id INTEGER NOT NULL, transaction_type TEXT NOT NULL,
			external_transaction_id TEXT, amount REAL NOT NULL,
			currency TEXT NOT NULL DEFAULT 'USD', status TEXT NOT NULL,
			webhook_event_id TEXT, webhook_event_type TEXT, raw_payload TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS licenses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			license_key TEXT NOT NULL UNIQUE,
			organization_name TEXT NOT NULL, contact_email TEXT NOT NULL,
			plan_name TEXT NOT NULL,
			max_users INTEGER NOT NULL DEFAULT -1, max_runners INTEGER NOT NULL DEFAULT -1,
			max_repositories INTEGER NOT NULL DEFAULT -1, max_concurrent_pods INTEGER NOT NULL DEFAULT -1,
			features TEXT DEFAULT '{}',
			issued_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME,
			signature TEXT NOT NULL, public_key_fingerprint TEXT,
			is_active INTEGER NOT NULL DEFAULT 1,
			revoked_at DATETIME, revocation_reason TEXT,
			activated_at DATETIME, activated_org_id INTEGER, last_verified_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}
}
