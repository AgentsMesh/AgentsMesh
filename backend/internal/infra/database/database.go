package database

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra/gormvalidate"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"
)

func New(cfg config.DatabaseConfig) (*gorm.DB, error) {
	logMode := logger.Info

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logMode),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Use(otelgorm.NewPlugin()); err != nil {
		slog.Warn("failed to enable GORM OpenTelemetry tracing", "error", err)
	}

	// Layer 2 of identifier contract defense: every domain model
	// implementing slugkit.IdentifierValidator gets its identifier fields
	// checked before Create/Update. See backend/pkg/slugkit/doc.go.
	if err := db.Use(&gormvalidate.Plugin{}); err != nil {
		return nil, fmt.Errorf("failed to register identifier validator plugin: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying DB: %w", err)
	}

	// Set connection pool settings for high-concurrency (100K runners support)
	sqlDB.SetMaxIdleConns(50)
	sqlDB.SetMaxOpenConns(300)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	if len(cfg.ReplicaDSNs) > 0 {
		var replicaDialectors []gorm.Dialector
		connectedCount := 0

		for i, dsn := range cfg.ReplicaDSNs {
			replicaDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
				Logger: logger.Default.LogMode(logMode),
			})
			if err != nil {
				slog.Warn("failed to connect to replica", "index", i, "error", err)
				continue
			}

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

		if len(replicaDialectors) > 0 {
			err = db.Use(dbresolver.Register(dbresolver.Config{
				Replicas: replicaDialectors,
				Policy:   dbresolver.RandomPolicy{},
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

func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying DB: %w", err)
	}
	return sqlDB.Close()
}
