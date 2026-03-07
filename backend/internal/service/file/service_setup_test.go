package file

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/infra/storage"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Create files table
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			uploader_id INTEGER NOT NULL,
			original_name TEXT NOT NULL,
			storage_key TEXT NOT NULL UNIQUE,
			mime_type TEXT NOT NULL,
			size INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create files table: %v", err)
	}

	return db
}

func setupTestService(t *testing.T) (*Service, *storage.MockStorage, *gorm.DB) {
	db := setupTestDB(t)
	mockStorage := storage.NewMockStorage()
	cfg := config.StorageConfig{
		MaxFileSize:  10, // 10MB
		AllowedTypes: []string{"image/jpeg", "image/png", "image/gif", "application/pdf"},
	}
	repo := infra.NewFileRepository(db)
	svc := NewService(repo, mockStorage, cfg)
	return svc, mockStorage, db
}

func TestNewService(t *testing.T) {
	svc, _, _ := setupTestService(t)
	if svc == nil {
		t.Error("expected service to be created")
	}
}
