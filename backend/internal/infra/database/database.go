package database

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"
)

// New creates a new database connection with automatic read-write splitting
// When replicas are configured via DB_REPLICA_DSNS, the returned *gorm.DB automatically:
// - Routes SELECT queries to replicas (round-robin load balancing)
// - Routes INSERT/UPDATE/DELETE to master
// - Keeps transactions on master
//
// Services can use the returned *gorm.DB normally without any changes.
func New(cfg config.DatabaseConfig) (*gorm.DB, error) {
	logMode := logger.Info

	// Connect to master
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logMode),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL DB to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying DB: %w", err)
	}

	// Set connection pool settings for high-concurrency (100K runners support)
	// MaxIdleConns: Keep 50 idle connections ready for burst traffic
	// MaxOpenConns: Allow up to 300 concurrent connections
	// ConnMaxLifetime: Recycle connections hourly to prevent stale connections
	// ConnMaxIdleTime: Close idle connections after 10 minutes to free resources
	sqlDB.SetMaxIdleConns(50)
	sqlDB.SetMaxOpenConns(300)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	// Connect to replicas if configured
	if len(cfg.ReplicaDSNs) > 0 {
		var replicaDialectors []gorm.Dialector
		connectedCount := 0

		for i, dsn := range cfg.ReplicaDSNs {
			// Validate replica DSN by attempting connection
			replicaDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
				Logger: logger.Default.LogMode(logMode),
			})
			if err != nil {
				slog.Warn("failed to connect to replica", "index", i, "error", err)
				continue
			}

			// Configure replica connection pool
			replicaSQL, err := replicaDB.DB()
			if err != nil {
				slog.Warn("failed to get replica underlying DB", "index", i, "error", err)
				continue
			}

			replicaSQL.SetMaxIdleConns(50)
			replicaSQL.SetMaxOpenConns(300)
			replicaSQL.SetConnMaxLifetime(time.Hour)
			replicaSQL.SetConnMaxIdleTime(10 * time.Minute)

			replicaDialectors = append(replicaDialectors, postgres.Open(dsn))
			connectedCount++
		}

		// Register DBResolver plugin for automatic read-write splitting
		if len(replicaDialectors) > 0 {
			err = db.Use(dbresolver.Register(dbresolver.Config{
				Replicas: replicaDialectors,
				Policy:   dbresolver.RandomPolicy{}, // Random load balancing
			}).SetConnMaxIdleTime(10 * time.Minute).
				SetConnMaxLifetime(time.Hour).
				SetMaxIdleConns(50).
				SetMaxOpenConns(300))

			if err != nil {
				slog.Warn("failed to register dbresolver", "error", err)
			} else {
				slog.Info("database read-write splitting enabled",
					"replicas", len(replicaDialectors))
			}
		}

		slog.Info("database cluster initialized",
			"replicas_configured", len(cfg.ReplicaDSNs),
			"replicas_connected", connectedCount)
	}

	return db, nil
}

// Close closes the database connection
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying DB: %w", err)
	}
	return sqlDB.Close()
}
