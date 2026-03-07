package file

import (
	"bytes"
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/infra/storage"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestIsAllowedType(t *testing.T) {
	svc, _, _ := setupTestService(t)

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
	svc, _, _ := setupTestService(t)

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

			// Should contain org ID
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

// Benchmark tests
func BenchmarkUpload(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.Exec(`CREATE TABLE files (id INTEGER PRIMARY KEY, organization_id INTEGER, uploader_id INTEGER, original_name TEXT, storage_key TEXT UNIQUE, mime_type TEXT, size INTEGER, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`)

	mockStorage := storage.NewMockStorage()
	cfg := config.StorageConfig{MaxFileSize: 10, AllowedTypes: []string{"image/png"}}
	repo := infra.NewFileRepository(db)
	svc := NewService(repo, mockStorage, cfg)

	content := []byte("benchmark test content")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(content)
		_, _ = svc.Upload(context.Background(), &UploadRequest{
			OrganizationID: 1,
			UploaderID:     100,
			FileName:       "test.png",
			ContentType:    "image/png",
			Size:           int64(len(content)),
			Reader:         reader,
		})
	}
}

func BenchmarkIsAllowedType(b *testing.B) {
	svc, _, _ := setupTestService(&testing.T{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.isAllowedType("image/png")
	}
}
