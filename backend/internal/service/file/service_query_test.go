package file

import (
	"context"
	"errors"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/file"
)

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
