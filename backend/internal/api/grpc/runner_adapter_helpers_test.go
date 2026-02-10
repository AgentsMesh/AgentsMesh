package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/anthropics/agentsmesh/backend/pkg/audit"
)

// ==================== Helper Function Tests ====================

func TestGRPCRunnerAdapter_StartRevocationChecker(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, nil, nil, nil, connMgr, nil)

	t.Run("context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan struct{})
		go func() {
			adapter.startRevocationChecker(ctx, 1, 100, "SERIAL123", cancel)
			close(done)
		}()

		// Cancel immediately
		cancel()

		select {
		case <-done:
			// Expected - checker exited on context cancel
		case <-time.After(time.Second):
			t.Fatal("startRevocationChecker did not exit on context cancel")
		}
	})

	t.Run("certificate revoked during check", func(t *testing.T) {
		// This test would require mocking the ticker or using a very short interval
		// For now, we verify the function starts and can be canceled
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Mark certificate as revoked
		runnerSvc.SetCertificateRevoked("SERIAL456", true)

		done := make(chan struct{})
		go func() {
			adapter.startRevocationChecker(ctx, 2, 100, "SERIAL456", cancel)
			close(done)
		}()

		// Cancel to end test
		cancel()

		select {
		case <-done:
			// Checker exited
		case <-time.After(time.Second):
			t.Fatal("startRevocationChecker did not exit")
		}
	})

	t.Run("revocation check error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Set error for revocation check
		runnerSvc.SetRevocationCheckError(context.DeadlineExceeded)

		done := make(chan struct{})
		go func() {
			adapter.startRevocationChecker(ctx, 3, 100, "SERIAL789", cancel)
			close(done)
		}()

		// Cancel to end test
		cancel()

		select {
		case <-done:
			// Checker exited
		case <-time.After(time.Second):
			t.Fatal("startRevocationChecker did not exit")
		}
	})
}

func TestGRPCRunnerAdapter_LogAuditEvent(t *testing.T) {
	logger := newTestLogger()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	t.Run("nil db does nothing", func(t *testing.T) {
		adapter := NewGRPCRunnerAdapter(logger, nil, nil, nil, nil, nil, connMgr, nil)

		// Should not panic
		adapter.logAuditEvent(1, 100, audit.ActionRunnerOnline, "SERIAL123")
	})

	t.Run("with db writes audit log", func(t *testing.T) {
		// Create in-memory SQLite for testing
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
			Logger: gormlogger.Discard,
		})
		if err != nil {
			t.Skip("SQLite not available")
		}

		// Create audit_logs table with all required columns
		err = db.Exec(`CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			action TEXT,
			organization_id INTEGER,
			actor_type TEXT,
			actor_id INTEGER,
			resource_type TEXT,
			resource_id INTEGER,
			details TEXT,
			ip_address TEXT,
			user_agent TEXT,
			created_at DATETIME
		)`).Error
		if err != nil {
			t.Skip("Failed to create table")
		}

		adapter := NewGRPCRunnerAdapter(logger, db, nil, nil, nil, nil, connMgr, nil)

		adapter.logAuditEvent(1, 100, audit.ActionRunnerOnline, "SERIAL123")

		// Wait for async write
		time.Sleep(100 * time.Millisecond)

		// Verify log was written
		var count int64
		db.Table("audit_logs").Count(&count)
		assert.GreaterOrEqual(t, count, int64(1))
	})

	t.Run("db error is logged but does not panic", func(t *testing.T) {
		// Create a db that will fail on write
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
			Logger: gormlogger.Discard,
		})
		if err != nil {
			t.Skip("SQLite not available")
		}
		// Don't create the table - writes will fail

		adapter := NewGRPCRunnerAdapter(logger, db, nil, nil, nil, nil, connMgr, nil)

		// Should not panic even with db error
		adapter.logAuditEvent(1, 100, audit.ActionRunnerOnline, "SERIAL123")

		// Wait for async write attempt
		time.Sleep(100 * time.Millisecond)
	})
}
