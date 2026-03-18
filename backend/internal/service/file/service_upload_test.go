package file

import (
	"context"
	"errors"
	"testing"
)

func TestRequestPresignedUpload_Success(t *testing.T) {
	svc, _ := setupTestService(t)

	resp, err := svc.RequestPresignedUpload(context.Background(), &PresignUploadRequest{
		OrganizationID: 1,
		FileName:       "test.png",
		ContentType:    "image/png",
		Size:           1024,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.PutURL == "" {
		t.Error("expected PutURL in response")
	}
	if resp.GetURL == "" {
		t.Error("expected GetURL in response")
	}
}

func TestRequestPresignedUpload_FileTooLarge(t *testing.T) {
	svc, _ := setupTestService(t)

	// MaxFileSize is 10MB, try 11MB
	_, err := svc.RequestPresignedUpload(context.Background(), &PresignUploadRequest{
		OrganizationID: 1,
		FileName:       "large.png",
		ContentType:    "image/png",
		Size:           11 * 1024 * 1024,
	})

	if err == nil {
		t.Fatal("expected error for large file")
	}
	if !errors.Is(err, ErrFileTooLarge) {
		t.Errorf("expected ErrFileTooLarge, got %v", err)
	}
}

func TestRequestPresignedUpload_InvalidFileType(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.RequestPresignedUpload(context.Background(), &PresignUploadRequest{
		OrganizationID: 1,
		FileName:       "test.exe",
		ContentType:    "application/x-executable",
		Size:           1024,
	})

	if err == nil {
		t.Fatal("expected error for invalid file type")
	}
	if !errors.Is(err, ErrInvalidFileType) {
		t.Errorf("expected ErrInvalidFileType, got %v", err)
	}
}

func TestRequestPresignedUpload_StorageError(t *testing.T) {
	svc, mockStorage := setupTestService(t)
	mockStorage.PresignPutURLErr = errors.New("storage unavailable")

	_, err := svc.RequestPresignedUpload(context.Background(), &PresignUploadRequest{
		OrganizationID: 1,
		FileName:       "test.png",
		ContentType:    "image/png",
		Size:           1024,
	})

	if err == nil {
		t.Fatal("expected error for storage failure")
	}
	if !errors.Is(err, ErrStorageError) {
		t.Errorf("expected ErrStorageError, got %v", err)
	}
}
