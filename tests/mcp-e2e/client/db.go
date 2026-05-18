package client

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// DB is a thin wrapper around the dev Postgres for direct fact assertions
// (e.g. confirm a block.create produced a row in `blocks`). Tests should
// prefer REST/MCP for reads where possible and only drop down to DB when
// asserting invariants the API doesn't expose (op_log entries, FK integrity).
type DB struct {
	conn *sql.DB
}

func OpenDB(dsn string) (*DB, error) {
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &DB{conn: conn}, nil
}

func (d *DB) Close() error { return d.conn.Close() }

// QueryRow exposes the underlying database connection for ad-hoc fact
// assertions in specs that need a query the helper methods don't cover.
// Using this directly is fine for one-off SQL — pull it into a named helper
// once two specs share the same query.
func (d *DB) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return d.conn.QueryRowContext(ctx, query, args...)
}

// CountBlocksByType returns the count of non-deleted blocks of a specific
// type within a workspace. Used by trigger-fire specs to assert agent_event
// blocks materialised in response to a target write.
func (d *DB) CountBlocksByType(ctx context.Context, workspaceID, blockType string) (int, error) {
	var n int
	err := d.conn.QueryRowContext(ctx,
		`SELECT count(*) FROM blocks
		   WHERE workspace_id = $1 AND type = $2 AND deleted_at IS NULL`,
		workspaceID, blockType,
	).Scan(&n)
	return n, err
}

// CountBlocks returns the number of non-deleted blocks in a workspace. Useful
// for asserting that a block.create or block.delete actually changed state.
func (d *DB) CountBlocks(ctx context.Context, workspaceID string) (int, error) {
	var n int
	err := d.conn.QueryRowContext(ctx,
		`SELECT count(*) FROM blocks WHERE workspace_id = $1 AND deleted_at IS NULL`,
		workspaceID,
	).Scan(&n)
	return n, err
}

// CountOps returns the number of op_log rows produced under a workspace. Each
// successful block.create/update/delete/add_ref/... must add at least one.
func (d *DB) CountOps(ctx context.Context, workspaceID string) (int, error) {
	var n int
	err := d.conn.QueryRowContext(ctx,
		`SELECT count(*) FROM block_ops WHERE workspace_id = $1`,
		workspaceID,
	).Scan(&n)
	return n, err
}

// GetWorkspaceIDBySlug looks up the UUID for an org's workspace by slug.
// Bypasses MCP so tests can verify the MCP-returned UUID matches DB truth.
func (d *DB) GetWorkspaceIDBySlug(ctx context.Context, orgSlug, wsSlug string) (string, error) {
	var id string
	err := d.conn.QueryRowContext(ctx,
		`SELECT bw.id::text
		   FROM block_workspaces bw
		   JOIN organizations o ON o.id = bw.organization_id
		  WHERE o.slug = $1 AND bw.slug = $2`,
		orgSlug, wsSlug,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("workspace %s not found in org %s", wsSlug, orgSlug)
	}
	return id, err
}

// GetBlockTexts returns (data.text, top-level text) for a block. Used by the
// data-vs-search field separation spec (issue #366): agents that put content
// only in top-level text leave data.text empty, which makes the UI render
// blank even though memory.retrieve still hits the block.
func (d *DB) GetBlockTexts(ctx context.Context, blockID string) (dataText, topText sql.NullString, err error) {
	err = d.conn.QueryRowContext(ctx,
		`SELECT data->>'text', text FROM blocks WHERE id = $1`,
		blockID,
	).Scan(&dataText, &topText)
	if err == sql.ErrNoRows {
		err = fmt.Errorf("block %s not found", blockID)
	}
	return
}
