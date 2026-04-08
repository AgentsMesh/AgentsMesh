package agentpod

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing.
// Delegates to testkit.SetupTestDB for shared schema.
func setupTestDB(t *testing.T) *gorm.DB {
	db := testkit.SetupTestDB(t)

	// Seed default test data used by many pod tests
	db.Exec("INSERT INTO runners (id, organization_id, node_id, status, current_pods) VALUES (1, 1, 'runner-001', 'online', 0)")
	db.Exec("INSERT INTO users (id, username, name, email) VALUES (1, 'testuser', 'Test User', 'test@example.com')")

	return db
}

// Helper functions
func intPtr(i int64) *int64 {
	return &i
}

func strPtr(s string) *string {
	return &s
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// newTestPodService wraps *gorm.DB into PodRepository for testing.
func newTestPodService(db *gorm.DB) *PodService {
	return NewPodService(infra.NewPodRepository(db))
}

// newTestSettingsService wraps *gorm.DB into SettingsRepository for testing.
func newTestSettingsService(db *gorm.DB) *SettingsService {
	return NewSettingsService(infra.NewSettingsRepository(db))
}

// newTestAIProviderService wraps *gorm.DB into AIProviderRepository for testing.
// Accepts nil db for tests that don't hit the DB (pure logic tests).
func newTestAIProviderService(db *gorm.DB, enc *crypto.Encryptor) *AIProviderService {
	if db == nil {
		return NewAIProviderService(nil, enc)
	}
	return NewAIProviderService(infra.NewAIProviderRepository(db), enc)
}

// newTestAutopilotService wraps *gorm.DB into AutopilotRepository for testing.
func newTestAutopilotService(db *gorm.DB) *AutopilotControllerService {
	return NewAutopilotControllerService(infra.NewAutopilotRepository(db))
}

func TestNewPodService(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	if svc == nil {
		t.Error("NewPodService returned nil")
	}
	if svc.repo == nil {
		t.Error("Service repo not set correctly")
	}
}

// suppress unused import for agentpod domain
var _ = agentpod.StatusRunning
