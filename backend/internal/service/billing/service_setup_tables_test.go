package billing

func getPaymentOrdersTableSQL() string {
	return `CREATE TABLE IF NOT EXISTS payment_orders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		order_no TEXT NOT NULL UNIQUE,
		external_order_no TEXT,
		order_type TEXT NOT NULL,
		plan_id INTEGER,
		billing_cycle TEXT,
		seats INTEGER DEFAULT 1,
		currency TEXT NOT NULL DEFAULT 'USD',
		amount REAL NOT NULL,
		discount_amount REAL DEFAULT 0,
		actual_amount REAL NOT NULL,
		payment_provider TEXT NOT NULL,
		payment_method TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		metadata TEXT,
		failure_reason TEXT,
		idempotency_key TEXT UNIQUE,
		expires_at DATETIME,
		paid_at DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		created_by_id INTEGER NOT NULL
	)`
}

func getPaymentTransactionsTableSQL() string {
	return `CREATE TABLE IF NOT EXISTS payment_transactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		payment_order_id INTEGER NOT NULL,
		transaction_type TEXT NOT NULL,
		external_transaction_id TEXT,
		amount REAL NOT NULL,
		currency TEXT NOT NULL DEFAULT 'USD',
		status TEXT NOT NULL,
		webhook_event_id TEXT,
		webhook_event_type TEXT,
		raw_payload TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`
}

func getInvoicesTableSQL() string {
	return `CREATE TABLE IF NOT EXISTS invoices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		payment_order_id INTEGER,
		invoice_no TEXT NOT NULL UNIQUE,
		status TEXT NOT NULL DEFAULT 'draft',
		currency TEXT NOT NULL DEFAULT 'USD',
		subtotal REAL NOT NULL,
		tax_amount REAL DEFAULT 0,
		total REAL NOT NULL,
		billing_name TEXT,
		billing_email TEXT,
		billing_address TEXT,
		period_start DATETIME NOT NULL,
		period_end DATETIME NOT NULL,
		line_items TEXT NOT NULL DEFAULT '[]',
		pdf_url TEXT,
		issued_at DATETIME,
		due_at DATETIME,
		paid_at DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`
}

func getAuxiliaryTablesSQL() string {
	return `CREATE TABLE IF NOT EXISTS organization_members (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		role TEXT NOT NULL DEFAULT 'member',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS runners (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS repositories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS pods (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		agent_status TEXT NOT NULL DEFAULT 'idle',
		last_activity DATETIME,
		agent_waiting_since DATETIME,
		finished_at DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS invitations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		email TEXT NOT NULL,
		accepted_at DATETIME,
		expires_at DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS plan_prices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		plan_id INTEGER NOT NULL,
		currency TEXT NOT NULL,
		price_monthly REAL NOT NULL,
		price_yearly REAL NOT NULL,
		stripe_price_id_monthly TEXT,
		stripe_price_id_yearly TEXT,
		lemonsqueezy_variant_id_monthly TEXT,
		lemonsqueezy_variant_id_yearly TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(plan_id, currency)
	);
	CREATE TABLE IF NOT EXISTS webhook_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_id TEXT NOT NULL,
		provider TEXT NOT NULL,
		event_type TEXT NOT NULL,
		processed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(event_id, provider)
	)`
}
