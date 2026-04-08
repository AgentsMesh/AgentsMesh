package testkit

// channelTableDDLs returns DDLs for channels, messages, members, access, bindings.
func channelTableDDLs() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS channels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL, name TEXT NOT NULL, description TEXT,
			document TEXT, repository_id INTEGER, ticket_id INTEGER,
			created_by_pod TEXT, created_by_user_id INTEGER,
			is_archived INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS channel_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL,
			sender_pod TEXT, sender_user_id INTEGER,
			message_type TEXT NOT NULL, content TEXT NOT NULL,
			metadata TEXT, edited_at DATETIME,
			is_deleted INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS channel_members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL, user_id INTEGER NOT NULL,
			role TEXT NOT NULL DEFAULT 'member',
			is_muted INTEGER NOT NULL DEFAULT 0,
			joined_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS channel_read_states (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL, user_id INTEGER NOT NULL,
			last_read_message_id INTEGER NOT NULL DEFAULT 0,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS channel_pods (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL, pod_key TEXT NOT NULL,
			joined_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS channel_access (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL,
			pod_key TEXT, user_id INTEGER,
			last_access DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS pod_bindings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			channel_id INTEGER,
			initiator_pod TEXT NOT NULL, target_pod TEXT NOT NULL,
			granted_scopes TEXT DEFAULT '[]', pending_scopes TEXT DEFAULT '[]',
			status TEXT NOT NULL DEFAULT 'pending',
			requested_at DATETIME, responded_at DATETIME, expires_at DATETIME,
			rejection_reason TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}
}

// ticketTableDDLs returns DDLs for tickets, comments, labels, MRs, relations.
func ticketTableDDLs() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS tickets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL DEFAULT 0, number INTEGER NOT NULL,
			slug TEXT NOT NULL, title TEXT NOT NULL, content TEXT,
			status TEXT NOT NULL DEFAULT 'backlog',
			priority TEXT NOT NULL DEFAULT 'none', severity TEXT,
			estimate INTEGER, due_date DATETIME, started_at DATETIME, completed_at DATETIME,
			repository_id INTEGER, reporter_id INTEGER NOT NULL DEFAULT 0,
			parent_ticket_id INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS labels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL, repository_id INTEGER,
			name TEXT NOT NULL, color TEXT NOT NULL DEFAULT '#808080',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS ticket_labels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_id INTEGER NOT NULL, label_id INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS ticket_assignees (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_id INTEGER NOT NULL, user_id INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS ticket_comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_id INTEGER NOT NULL, user_id INTEGER NOT NULL,
			content TEXT NOT NULL, parent_id INTEGER, mentions TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS ticket_merge_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL, repository_id INTEGER NOT NULL,
			ticket_id INTEGER, pod_id INTEGER, mr_iid INTEGER NOT NULL,
			mr_url TEXT NOT NULL UNIQUE, source_branch TEXT NOT NULL,
			target_branch TEXT NOT NULL DEFAULT 'main',
			title TEXT, state TEXT NOT NULL DEFAULT 'opened',
			pipeline_status TEXT, pipeline_id INTEGER, pipeline_url TEXT,
			merge_commit_sha TEXT, merged_at DATETIME, merged_by_id INTEGER,
			last_synced_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS ticket_relations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			source_ticket_id INTEGER NOT NULL, target_ticket_id INTEGER NOT NULL,
			relation_type TEXT NOT NULL,
			created_by_id INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}
}
