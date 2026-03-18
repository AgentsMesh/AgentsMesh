package file

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra/storage"
)

func setupTestService(t *testing.T) (*Service, *storage.MockStorage) {
	mockStorage := storage.NewMockStorage()
	cfg := config.StorageConfig{
		MaxFileSize:  10, // 10MB
		AllowedTypes: []string{"image/jpeg", "image/png", "image/gif", "application/pdf"},
	}
	svc := NewService(mockStorage, cfg)
	return svc, mockStorage
}

func TestNewService(t *testing.T) {
	svc, _ := setupTestService(t)
	if svc == nil {
		t.Error("expected service to be created")
	}
}
