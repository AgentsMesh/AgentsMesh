package main

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// auditRewrite records (table, column, row_id, old, new) in
// identifier_backfill_audit. Run inside the caller's transaction.
func auditRewrite(tx *gorm.DB, table, column string, rowID int64, oldVal, newVal string) error {
	return tx.Exec(
		`INSERT INTO identifier_backfill_audit (table_name, column_name, row_id, old_value, new_value) VALUES (?, ?, ?, ?, ?)`,
		table, column, rowID, oldVal, newVal,
	).Error
}

// applyOrgScopedSlug updates a row's slug column and records the audit entry
// atomically. Used by channels/api_keys whose UNIQUE constraint is
// (organization_id, slug) — the actual UPDATE happens via the caller's SQL.
func applyOrgScopedSlug(ctx context.Context, db *gorm.DB, table string, rowID int64, oldName, newSlug string) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updateSQL := fmt.Sprintf("UPDATE %s SET slug = ? WHERE id = ?", table)
		if err := tx.Exec(updateSQL, newSlug, rowID).Error; err != nil {
			return fmt.Errorf("update %s: %w", table, err)
		}
		return auditRewrite(tx, table, "slug", rowID, oldName, newSlug)
	})
}
