package file

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra/storage"
)

func TestIsAllowedType(t *testing.T) {
	svc, _ := setupTestService(t)

	tests := []struct {
		name        string
		contentType string
		expected    bool
	}{
		{"jpeg", "image/jpeg", true},
		{"png", "image/png", true},
		{"gif", "image/gif", true},
		{"pdf", "application/pdf", true},
		{"exe", "application/x-executable", false},
		{"text", "text/plain", false},
		{"webp", "image/webp", false}, // Not in allowed list
		{"jpeg with charset", "image/jpeg; charset=utf-8", true},
		{"case insensitive", "IMAGE/PNG", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.isAllowedType(tt.contentType)
			if result != tt.expected {
				t.Errorf("contentType %s: expected %v, got %v", tt.contentType, tt.expected, result)
			}
		})
	}
}

func TestGenerateStorageKey(t *testing.T) {
	svc, _ := setupTestService(t)

	tests := []struct {
		name     string
		orgID    int64
		fileName string
	}{
		{"png file", 1, "test.png"},
		{"jpeg file", 2, "photo.jpeg"},
		{"no extension", 3, "noext"},
		{"multiple dots", 4, "file.name.with.dots.pdf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := svc.generateStorageKey(tt.orgID, tt.fileName)

			if key == "" {
				t.Error("expected non-empty key")
			}

			// Key should be unique (contains UUID)
			key2 := svc.generateStorageKey(tt.orgID, tt.fileName)
			if key == key2 {
				t.Error("expected unique keys for same input")
			}
		})
	}
}

func BenchmarkPresignUpload(b *testing.B) {
	mockStorage := storage.NewMockStorage()
	cfg := config.StorageConfig{MaxFileSize: 10, AllowedTypes: []string{"image/png"}}
	svc := NewService(mockStorage, cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.RequestPresignedUpload(context.Background(), &PresignUploadRequest{
			OrganizationID: 1,
			FileName:       "test.png",
			ContentType:    "image/png",
			Size:           1024,
		})
	}
}

func BenchmarkIsAllowedType(b *testing.B) {
	svc, _ := setupTestService(&testing.T{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.isAllowedType("image/png")
	}
}
