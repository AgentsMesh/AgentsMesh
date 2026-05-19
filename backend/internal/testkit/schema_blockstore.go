package testkit

// blockstoreTableDDLs returns SQLite-compatible DDL for the Block Store tables.
// The production Postgres migration uses UUID / JSONB / TSVECTOR / GIN / partial
// unique indexes; here we substitute TEXT / TEXT / omit FTS / plain indexes
// and push uniqueness invariants into the service layer (which is exactly
// where our integration tests want them exercised anyway).
func blockstoreTableDDLs() []string {
	return []string{
		`CREATE TABLE block_workspaces (
			id              TEXT PRIMARY KEY,
			organization_id INTEGER NOT NULL,
			slug            TEXT NOT NULL,
			name            TEXT NOT NULL,
			root_block_id   TEXT,
			created_by      INTEGER NOT NULL,
			created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (organization_id, slug)
		)`,
		`CREATE INDEX idx_block_workspaces_org ON block_workspaces (organization_id)`,

		`CREATE TABLE blocks (
			id              TEXT PRIMARY KEY,
			workspace_id    TEXT NOT NULL,
			type            TEXT NOT NULL,
			data            TEXT NOT NULL DEFAULT '{}',
			text            TEXT,
			meta            TEXT NOT NULL DEFAULT '{}',
			created_by      INTEGER NOT NULL,
			created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at      DATETIME
		)`,
		`CREATE INDEX idx_blocks_workspace_type ON blocks (workspace_id, type)`,

		`CREATE TABLE block_refs (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			workspace_id    TEXT NOT NULL,
			from_id         TEXT NOT NULL,
			to_id           TEXT NOT NULL,
			rel             TEXT NOT NULL,
			order_key       TEXT,
			anchor          TEXT,
			meta            TEXT NOT NULL DEFAULT '{}',
			created_by      INTEGER NOT NULL,
			created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE UNIQUE INDEX idx_block_refs_unique_edge ON block_refs (from_id, to_id, rel, IFNULL(anchor, ''))`,
		`CREATE INDEX idx_block_refs_children ON block_refs (from_id, rel, order_key)`,
		`CREATE INDEX idx_block_refs_backlinks ON block_refs (to_id, rel)`,

		`CREATE TABLE block_ops (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			workspace_id    TEXT NOT NULL,
			idempotency_key TEXT UNIQUE,
			actor_type      TEXT NOT NULL,
			actor_id        INTEGER NOT NULL,
			op              TEXT NOT NULL,
			target_block    TEXT,
			target_ref      INTEGER,
			payload         TEXT NOT NULL,
			forward         TEXT NOT NULL,
			inverse         TEXT NOT NULL,
			context         TEXT NOT NULL DEFAULT '{}',
			parent_op_id    INTEGER,
			applied_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CHECK (
				(target_block IS NOT NULL AND target_ref IS NULL)
			 OR (target_block IS NULL AND target_ref IS NOT NULL)
			)
		)`,
		`CREATE INDEX idx_block_ops_stream ON block_ops (workspace_id, id)`,

		`CREATE TABLE block_embeddings (
			block_id    TEXT PRIMARY KEY,
			model       TEXT NOT NULL,
			dims        INTEGER NOT NULL,
			vector      TEXT NOT NULL,
			source_hash TEXT NOT NULL,
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX idx_block_embeddings_model ON block_embeddings (model)`,
	}
}
