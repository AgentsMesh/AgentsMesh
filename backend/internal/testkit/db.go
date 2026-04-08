// Package testkit provides shared test infrastructure for backend integration tests.
// It consolidates DB setup, factory functions, and test context helpers
// into a single reusable package.
package testkit

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SetupTestDB creates an in-memory SQLite database with all business tables.
// This is the single source of truth for test schema — all services should
// use this instead of maintaining local DDL definitions.
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("testkit: failed to open database: %v", err)
	}

	for _, ddl := range allTableDDLs() {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("testkit: failed to create table: %v\nDDL: %s", err, ddl[:min(len(ddl), 80)])
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
