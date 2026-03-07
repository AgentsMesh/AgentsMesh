package binding

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MockPodQuerier implements PodQuerier for testing
type MockPodQuerier struct {
	pods map[string]map[string]interface{}
	err  error
}

func NewMockPodQuerier() *MockPodQuerier {
	return &MockPodQuerier{
		pods: make(map[string]map[string]interface{}),
	}
}

func (m *MockPodQuerier) AddPod(key string, info map[string]interface{}) {
	m.pods[key] = info
}

func (m *MockPodQuerier) GetPodInfo(ctx context.Context, podKey string) (map[string]interface{}, error) {
	if m.err != nil {
		return nil, m.err
	}
	if info, ok := m.pods[podKey]; ok {
		return info, nil
	}
	return nil, nil
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Create pod_bindings table
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS pod_bindings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			initiator_pod TEXT NOT NULL,
			target_pod TEXT NOT NULL,
			granted_scopes TEXT,
			pending_scopes TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			requested_at DATETIME,
			responded_at DATETIME,
			expires_at DATETIME,
			rejection_reason TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create pod_bindings table: %v", err)
	}

	return db
}

// newTestService creates a binding Service backed by an in-memory DB for testing.
func newTestService(db *gorm.DB, querier PodQuerier) *Service {
	return NewService(infra.NewBindingRepository(db), querier)
}

func TestNewService(t *testing.T) {
	db := setupTestDB(t)
	querier := NewMockPodQuerier()
	service := newTestService(db, querier)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewServiceWithoutQuerier(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db, nil)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
}
