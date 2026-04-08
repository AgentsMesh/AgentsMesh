package supportticket

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testkit.SetupTestDB(t)
}

func createTestService(t *testing.T) (*Service, *gorm.DB) {
	db := setupTestDB(t)
	repo := infra.NewSupportTicketRepository(db)
	service := NewService(repo, nil, config.StorageConfig{})
	return service, db
}

func createTestUser(t *testing.T, db *gorm.DB, userID int64, email string) {
	t.Helper()
	err := db.Exec(
		`INSERT INTO users (id, email, username, name, is_active, is_system_admin, is_email_verified) VALUES (?, ?, ?, ?, 1, 0, 1)`,
		userID, email, email, email,
	).Error
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
}

func TestNewService(t *testing.T) {
	db := setupTestDB(t)
	repo := infra.NewSupportTicketRepository(db)
	service := NewService(repo, nil, config.StorageConfig{})

	if service == nil {
		t.Fatal("expected non-nil service")
	}
}
