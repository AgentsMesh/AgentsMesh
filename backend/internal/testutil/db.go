// Package testutil provides shared test infrastructure for backend integration tests.
// It consolidates duplicate setupTestDB() and factory functions from 80+ test files
// into a single reusable package.
package testutil

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SetupTestDB creates an in-memory SQLite database with all business tables.
// This replaces the per-package setupTestDB() pattern, eliminating duplication.
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("testutil: failed to open database: %v", err)
	}

	for _, ddl := range allTableDDLs() {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("testutil: failed to create table: %v\nDDL: %s", err, ddl[:min(len(ddl), 80)])
		}
	}

	return db
}

// allTableDDLs returns all table DDL statements in dependency order.
func allTableDDLs() []string {
	var ddls []string
	ddls = append(ddls, coreTableDDLs()...)
	ddls = append(ddls, runnerTableDDLs()...)
	ddls = append(ddls, podTableDDLs()...)
	ddls = append(ddls, channelTableDDLs()...)
	ddls = append(ddls, ticketTableDDLs()...)
	ddls = append(ddls, loopTableDDLs()...)
	ddls = append(ddls, billingTableDDLs()...)
	ddls = append(ddls, supportTableDDLs()...)
	return ddls
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
