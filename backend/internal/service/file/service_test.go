package file

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/file"
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
	svc := NewService(db, mockStorage, cfg)
	return svc, mockStorage, db
}

func TestNewService(t *testing.T) {
	svc, _, _ := setupTestService(t)
	if svc == nil {
		t.Error("expected service to be created")
	}
}

func TestUpload_Success(t *testing.T) {
	svc, mockStorage, _ := setupTestService(t)

	content := []byte("test image content")
	reader := bytes.NewReader(content)

	resp, err := svc.Upload(context.Background(), &UploadRequest{
		OrganizationID: 1,
		UploaderID:     100,
		FileName:       "test.png",
		ContentType:    "image/png",
		Size:           int64(len(content)),
		Reader:         reader,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.File == nil {
		t.Fatal("expected file in response")
	}
	if resp.File.OriginalName != "test.png" {
		t.Errorf("expected OriginalName 'test.png', got %s", resp.File.OriginalName)
	}
	if resp.File.MimeType != "image/png" {
		t.Errorf("expected MimeType 'image/png', got %s", resp.File.MimeType)
	}
	if resp.File.OrganizationID != 1 {
		t.Errorf("expected OrganizationID 1, got %d", resp.File.OrganizationID)
	}
	if resp.File.UploaderID != 100 {
		t.Errorf("expected UploaderID 100, got %d", resp.File.UploaderID)
	}
	if resp.URL == "" {
		t.Error("expected URL in response")
	}

	// Verify file was stored
	if mockStorage.FileCount() != 1 {
		t.Errorf("expected 1 file in storage, got %d", mockStorage.FileCount())
	}
}

func TestUpload_FileTooLarge(t *testing.T) {
	svc, _, _ := setupTestService(t)

	// MaxFileSize is 10MB, try to upload 11MB
	content := make([]byte, 11*1024*1024)
	reader := bytes.NewReader(content)

	_, err := svc.Upload(context.Background(), &UploadRequest{
		OrganizationID: 1,
		UploaderID:     100,
		FileName:       "large.png",
		ContentType:    "image/png",
		Size:           int64(len(content)),
		Reader:         reader,
	})

	if err == nil {
		t.Fatal("expected error for large file")
	}
	if !errors.Is(err, ErrFileTooLarge) {
		t.Errorf("expected ErrFileTooLarge, got %v", err)
	}
}

func TestUpload_InvalidFileType(t *testing.T) {
	svc, _, _ := setupTestService(t)

	content := []byte("test content")
	reader := bytes.NewReader(content)

	_, err := svc.Upload(context.Background(), &UploadRequest{
		OrganizationID: 1,
		UploaderID:     100,
		FileName:       "test.exe",
		ContentType:    "application/x-executable",
		Size:           int64(len(content)),
		Reader:         reader,
	})

	if err == nil {
		t.Fatal("expected error for invalid file type")
	}
	if !errors.Is(err, ErrInvalidFileType) {
		t.Errorf("expected ErrInvalidFileType, got %v", err)
	}
}

func TestUpload_StorageError(t *testing.T) {
	svc, mockStorage, _ := setupTestService(t)
	mockStorage.UploadErr = errors.New("storage unavailable")

	content := []byte("test content")
	reader := bytes.NewReader(content)

	_, err := svc.Upload(context.Background(), &UploadRequest{
		OrganizationID: 1,
		UploaderID:     100,
		FileName:       "test.png",
		ContentType:    "image/png",
		Size:           int64(len(content)),
		Reader:         reader,
	})

	if err == nil {
		t.Fatal("expected error for storage failure")
	}
	if !errors.Is(err, ErrStorageError) {
		t.Errorf("expected ErrStorageError, got %v", err)
	}
}

func TestGetByID_Success(t *testing.T) {
	svc, _, db := setupTestService(t)

	// Insert a file directly
	f := &file.File{
		OrganizationID: 1,
		UploaderID:     100,
		OriginalName:   "test.png",
		StorageKey:     "orgs/1/files/2024/01/abc123.png",
		MimeType:       "image/png",
		Size:           1024,
	}
	db.Create(f)

	result, err := svc.GetByID(context.Background(), f.ID, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected file")
	}
	if result.ID != f.ID {
		t.Errorf("expected ID %d, got %d", f.ID, result.ID)
	}
	if result.OriginalName != "test.png" {
		t.Errorf("expected OriginalName 'test.png', got %s", result.OriginalName)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	svc, _, _ := setupTestService(t)

	_, err := svc.GetByID(context.Background(), 999, 1)
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	if !errors.Is(err, ErrFileNotFound) {
		t.Errorf("expected ErrFileNotFound, got %v", err)
	}
}

func TestGetByID_WrongOrganization(t *testing.T) {
	svc, _, db := setupTestService(t)

	// Insert a file for org 1
	f := &file.File{
		OrganizationID: 1,
		UploaderID:     100,
		OriginalName:   "test.png",
		StorageKey:     "orgs/1/files/2024/01/abc123.png",
		MimeType:       "image/png",
		Size:           1024,
	}
	db.Create(f)

	// Try to get it with org 2
	_, err := svc.GetByID(context.Background(), f.ID, 2)
	if err == nil {
		t.Fatal("expected error for wrong organization")
	}
	if !errors.Is(err, ErrFileNotFound) {
		t.Errorf("expected ErrFileNotFound, got %v", err)
	}
}

func TestDelete_Success(t *testing.T) {
	svc, mockStorage, db := setupTestService(t)

	// First upload a file
	content := []byte("test content")
	reader := bytes.NewReader(content)

	resp, err := svc.Upload(context.Background(), &UploadRequest{
		OrganizationID: 1,
		UploaderID:     100,
		FileName:       "test.png",
		ContentType:    "image/png",
		Size:           int64(len(content)),
		Reader:         reader,
	})
	if err != nil {
		t.Fatalf("failed to upload: %v", err)
	}

	// Verify file exists
	if mockStorage.FileCount() != 1 {
		t.Fatalf("expected 1 file in storage")
	}

	// Delete the file
	err = svc.Delete(context.Background(), resp.File.ID, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was deleted from storage
	if mockStorage.FileCount() != 0 {
		t.Errorf("expected 0 files in storage, got %d", mockStorage.FileCount())
	}

	// Verify file was deleted from database
	var count int64
	db.Model(&file.File{}).Where("id = ?", resp.File.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 files in database, got %d", count)
	}
}

func TestDelete_NotFound(t *testing.T) {
	svc, _, _ := setupTestService(t)

	err := svc.Delete(context.Background(), 999, 1)
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	if !errors.Is(err, ErrFileNotFound) {
		t.Errorf("expected ErrFileNotFound, got %v", err)
	}
}

func TestDelete_StorageError(t *testing.T) {
	svc, mockStorage, db := setupTestService(t)

	// Insert a file directly
	f := &file.File{
		OrganizationID: 1,
		UploaderID:     100,
		OriginalName:   "test.png",
		StorageKey:     "orgs/1/files/2024/01/abc123.png",
		MimeType:       "image/png",
		Size:           1024,
	}
	db.Create(f)

	// Simulate storage error
	mockStorage.DeleteErr = errors.New("storage unavailable")

	err := svc.Delete(context.Background(), f.ID, 1)
	if err == nil {
		t.Fatal("expected error for storage failure")
	}
	if !errors.Is(err, ErrStorageError) {
		t.Errorf("expected ErrStorageError, got %v", err)
	}
}

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
	svc := NewService(db, mockStorage, cfg)

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
