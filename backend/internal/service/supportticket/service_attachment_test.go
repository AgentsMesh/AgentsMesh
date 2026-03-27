package supportticket

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	domain "github.com/anthropics/agentsmesh/backend/internal/domain/supportticket"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/infra/storage"
	"gorm.io/gorm"
)

// --- Mock Storage ---

type mockStorage struct {
	uploadErr  error
	deleteErr  error
	getURLErr  error
	getURLVal  string
	uploaded   []string // track uploaded keys
	deleted    []string // track deleted keys
}

func (m *mockStorage) Upload(_ context.Context, key string, _ io.Reader, _ int64, _ string) (*storage.FileInfo, error) {
	if m.uploadErr != nil {
		return nil, m.uploadErr
	}
	m.uploaded = append(m.uploaded, key)
	return &storage.FileInfo{Key: key}, nil
}

func (m *mockStorage) Delete(_ context.Context, key string) error {
	m.deleted = append(m.deleted, key)
	return m.deleteErr
}

func (m *mockStorage) GetURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	if m.getURLErr != nil {
		return "", m.getURLErr
	}
	return m.getURLVal, nil
}

func (m *mockStorage) GetInternalURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", nil
}

func (m *mockStorage) Exists(_ context.Context, _ string) (bool, error) {
	return true, nil
}

func (m *mockStorage) PresignPutURL(_ context.Context, _ string, _ string, _ time.Duration) (string, error) {
	return "", nil
}

func (m *mockStorage) InternalPresignPutURL(_ context.Context, _ string, _ string, _ time.Duration) (string, error) {
	return "", nil
}

// testStorageCfg returns a StorageConfig with the given max file size in MB.
func testStorageCfg(maxMB int64) config.StorageConfig {
	return config.StorageConfig{MaxFileSize: maxMB}
}

func createServiceWithStorage(t *testing.T, stor *mockStorage, cfg config.StorageConfig) (*Service, *gorm.DB) {
	db := setupTestDB(t)
	repo := infra.NewSupportTicketRepository(db)
	service := NewService(repo, stor, cfg)
	return service, db
}

// --- UploadAttachment tests ---

func TestUploadAttachment(t *testing.T) {
	stor := &mockStorage{getURLVal: "https://example.com/file"}
	service, db := createServiceWithStorage(t, stor, testStorageCfg(10))
	ctx := context.Background()
	createTestUser(t, db, 1, "user@test.com")

	// Create ticket
	ticket, err := service.Create(ctx, 1, &CreateRequest{
		Title:    "Test",
		Category: domain.CategoryBug,
		Content:  "test content",
	})
	if err != nil {
		t.Fatalf("failed to create ticket: %v", err)
	}

	// Upload attachment
	reader := bytes.NewReader([]byte("file data"))
	att, err := service.UploadAttachment(ctx, ticket.ID, 1, nil, false, &UploadAttachmentRequest{
		FileName:    "test.png",
		ContentType: "image/png",
		Size:        9,
		Reader:      reader,
	})
	if err != nil {
		t.Fatalf("failed to upload attachment: %v", err)
	}
	if att.OriginalName != "test.png" {
		t.Errorf("expected OriginalName 'test.png', got %s", att.OriginalName)
	}
	if att.MimeType != "image/png" {
		t.Errorf("expected MimeType 'image/png', got %s", att.MimeType)
	}
	if att.TicketID != ticket.ID {
		t.Errorf("expected TicketID %d, got %d", ticket.ID, att.TicketID)
	}
	if len(stor.uploaded) != 1 {
		t.Errorf("expected 1 upload, got %d", len(stor.uploaded))
	}
	if !strings.Contains(stor.uploaded[0], "support-tickets/1/") {
		t.Errorf("storage key should contain user path, got %s", stor.uploaded[0])
	}
}

func TestUploadAttachment_NilStorage(t *testing.T) {
	service, db := createTestService(t)
	ctx := context.Background()
	createTestUser(t, db, 1, "user@test.com")

	ticket, _ := service.Create(ctx, 1, &CreateRequest{
		Title:    "Test",
		Category: domain.CategoryBug,
		Content:  "test",
	})

	_, err := service.UploadAttachment(ctx, ticket.ID, 1, nil, false, &UploadAttachmentRequest{
		FileName: "test.png", ContentType: "image/png", Size: 100,
		Reader: bytes.NewReader([]byte("data")),
	})
	if !errors.Is(err, ErrStorageError) {
		t.Errorf("expected ErrStorageError, got %v", err)
	}
}

func TestUploadAttachment_FileTooLarge(t *testing.T) {
	stor := &mockStorage{}
	service, db := createServiceWithStorage(t, stor, testStorageCfg(1)) // 1MB max
	ctx := context.Background()
	createTestUser(t, db, 1, "user@test.com")

	ticket, _ := service.Create(ctx, 1, &CreateRequest{
		Title:    "Test",
		Category: domain.CategoryBug,
		Content:  "test",
	})

	_, err := service.UploadAttachment(ctx, ticket.ID, 1, nil, false, &UploadAttachmentRequest{
		FileName: "big.zip", ContentType: "application/zip",
		Size:   2 * 1024 * 1024, // 2MB > 1MB max
		Reader: bytes.NewReader([]byte("data")),
	})
	if !errors.Is(err, ErrFileTooLarge) {
		t.Errorf("expected ErrFileTooLarge, got %v", err)
	}
}

func TestUploadAttachment_DefaultMaxSize(t *testing.T) {
	stor := &mockStorage{}
	// MaxFileSize = 0 -> fallback to 10MB
	service, db := createServiceWithStorage(t, stor, testStorageCfg(0))
	ctx := context.Background()
	createTestUser(t, db, 1, "user@test.com")

	ticket, _ := service.Create(ctx, 1, &CreateRequest{
		Title: "Test", Category: domain.CategoryBug, Content: "test",
	})

	// 11MB should exceed 10MB default
	_, err := service.UploadAttachment(ctx, ticket.ID, 1, nil, false, &UploadAttachmentRequest{
		FileName: "big.zip", ContentType: "application/zip",
		Size:   11 * 1024 * 1024,
		Reader: bytes.NewReader([]byte("data")),
	})
	if !errors.Is(err, ErrFileTooLarge) {
		t.Errorf("expected ErrFileTooLarge with default max size, got %v", err)
	}
}

func TestUploadAttachment_TicketNotFound(t *testing.T) {
	stor := &mockStorage{}
	service, _ := createServiceWithStorage(t, stor, testStorageCfg(10))
	ctx := context.Background()

	_, err := service.UploadAttachment(ctx, 99999, 1, nil, false, &UploadAttachmentRequest{
		FileName: "test.png", ContentType: "image/png", Size: 100,
		Reader: bytes.NewReader([]byte("data")),
	})
	if !errors.Is(err, ErrTicketNotFound) {
		t.Errorf("expected ErrTicketNotFound, got %v", err)
	}
}

func TestUploadAttachment_WrongUser(t *testing.T) {
	stor := &mockStorage{}
	service, db := createServiceWithStorage(t, stor, testStorageCfg(10))
	ctx := context.Background()
	createTestUser(t, db, 1, "user@test.com")
	createTestUser(t, db, 2, "other@test.com")

	ticket, _ := service.Create(ctx, 1, &CreateRequest{
		Title: "Test", Category: domain.CategoryBug, Content: "test",
	})

	// User 2 tries to upload to User 1's ticket
	_, err := service.UploadAttachment(ctx, ticket.ID, 2, nil, false, &UploadAttachmentRequest{
		FileName: "test.png", ContentType: "image/png", Size: 100,
		Reader: bytes.NewReader([]byte("data")),
	})
	if !errors.Is(err, ErrAccessDenied) {
		t.Errorf("expected ErrAccessDenied, got %v", err)
	}
}

func TestUploadAttachment_AdminReplyBypass(t *testing.T) {
	stor := &mockStorage{}
	service, db := createServiceWithStorage(t, stor, testStorageCfg(10))
	ctx := context.Background()
	createTestUser(t, db, 1, "user@test.com")
	createTestUser(t, db, 2, "admin@test.com")

	ticket, _ := service.Create(ctx, 1, &CreateRequest{
		Title: "Test", Category: domain.CategoryBug, Content: "test",
	})

	// Admin adds reply
	msg, _ := service.AdminAddReply(ctx, ticket.ID, 2, &AddMessageRequest{Content: "reply"})

	// Admin uploads attachment to their own reply
	att, err := service.UploadAttachment(ctx, ticket.ID, 2, &msg.ID, true, &UploadAttachmentRequest{
		FileName: "admin.png", ContentType: "image/png", Size: 100,
		Reader: bytes.NewReader([]byte("data")),
	})
	if err != nil {
		t.Fatalf("expected admin reply bypass, got error: %v", err)
	}
	if att.UploaderID != 2 {
		t.Errorf("expected UploaderID 2, got %d", att.UploaderID)
	}
}

func TestUploadAttachment_UploadError(t *testing.T) {
	stor := &mockStorage{uploadErr: errors.New("s3 error")}
	service, db := createServiceWithStorage(t, stor, testStorageCfg(10))
	ctx := context.Background()
	createTestUser(t, db, 1, "user@test.com")

	ticket, _ := service.Create(ctx, 1, &CreateRequest{
		Title: "Test", Category: domain.CategoryBug, Content: "test",
	})

	_, err := service.UploadAttachment(ctx, ticket.ID, 1, nil, false, &UploadAttachmentRequest{
		FileName: "test.png", ContentType: "image/png", Size: 100,
		Reader: bytes.NewReader([]byte("data")),
	})
	if !errors.Is(err, ErrStorageError) {
		t.Errorf("expected ErrStorageError, got %v", err)
	}
}

func TestUploadAttachment_NoExtension(t *testing.T) {
	stor := &mockStorage{}
	service, db := createServiceWithStorage(t, stor, testStorageCfg(10))
	ctx := context.Background()
	createTestUser(t, db, 1, "user@test.com")

	ticket, _ := service.Create(ctx, 1, &CreateRequest{
		Title: "Test", Category: domain.CategoryBug, Content: "test",
	})

	att, err := service.UploadAttachment(ctx, ticket.ID, 1, nil, false, &UploadAttachmentRequest{
		FileName: "README", ContentType: "text/plain", Size: 100,
		Reader: bytes.NewReader([]byte("data")),
	})
	if err != nil {
		t.Fatalf("failed to upload: %v", err)
	}
	if !strings.HasSuffix(stor.uploaded[0], ".bin") {
		t.Errorf("expected .bin extension for no-extension file, got %s", stor.uploaded[0])
	}
	_ = att
}

func TestUploadAttachment_WithMessageID(t *testing.T) {
	stor := &mockStorage{}
	service, db := createServiceWithStorage(t, stor, testStorageCfg(10))
	ctx := context.Background()
	createTestUser(t, db, 1, "user@test.com")

	ticket, _ := service.Create(ctx, 1, &CreateRequest{
		Title: "Test", Category: domain.CategoryBug, Content: "test",
	})

	msg, _ := service.AddMessage(ctx, ticket.ID, 1, &AddMessageRequest{Content: "msg"})

	att, err := service.UploadAttachment(ctx, ticket.ID, 1, &msg.ID, false, &UploadAttachmentRequest{
		FileName: "file.txt", ContentType: "text/plain", Size: 100,
		Reader: bytes.NewReader([]byte("data")),
	})
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if att.MessageID == nil || *att.MessageID != msg.ID {
		t.Errorf("expected MessageID %d", msg.ID)
	}
}
