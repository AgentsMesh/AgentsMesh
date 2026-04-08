package binding

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
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
	t.Helper()
	return testkit.SetupTestDB(t)
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
