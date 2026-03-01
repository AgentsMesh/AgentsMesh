package file

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/file"
)

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
