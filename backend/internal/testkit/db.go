package testkit

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("testkit: failed to open database: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}

	for _, ddl := range allTableDDLs() {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("testkit: failed to create table: %v\nDDL: %s", err, ddl[:min(len(ddl), 80)])
		}
	}

	return db
}

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
	ddls = append(ddls, blockstoreTableDDLs()...)
	return ddls
}
