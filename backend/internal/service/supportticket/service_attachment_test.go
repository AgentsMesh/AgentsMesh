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

func createServiceWithStorage(t *testing.T, stor *mockStorage, cfg config.StorageConfig) (*Service, *gorm.DB) {
	db := setupTestDB(t)
	repo := infra.NewSupportTicketRepository(db)
	service := NewService(repo, stor, cfg)
	return service, db
}

// --- UploadAttachment tests ---

func TestUploadAttachment(t *testing.T) {
	stor := &mockStorage{getURLVal: "https://example.com/file"}
	service, db := createServiceWithStorage(t, stor, config.StorageConfig{MaxFileSize: 10})
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
	service, db := createServiceWithStorage(t, stor, config.StorageConfig{MaxFileSize: 1}) // 1MB max
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
	service, db := createServiceWithStorage(t, stor, config.StorageConfig{MaxFileSize: 0})
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
	service, _ := createServiceWithStorage(t, stor, config.StorageConfig{MaxFileSize: 10})
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
	service, db := createServiceWithStorage(t, stor, config.StorageConfig{MaxFileSize: 10})
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
	service, db := createServiceWithStorage(t, stor, config.StorageConfig{MaxFileSize: 10})
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
	service, db := createServiceWithStorage(t, stor, config.StorageConfig{MaxFileSize: 10})
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
	service, db := createServiceWithStorage(t, stor, config.StorageConfig{MaxFileSize: 10})
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
	service, db := createServiceWithStorage(t, stor, config.StorageConfig{MaxFileSize: 10})
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

// --- GetAttachmentURL tests ---

func TestGetAttachmentURL(t *testing.T) {
	stor := &mockStorage{getURLVal: "https://s3.example.com/presigned"}
	service, db := createServiceWithStorage(t, stor, config.StorageConfig{MaxFileSize: 10})
	ctx := context.Background()
	createTestUser(t, db, 1, "user@test.com")

	ticket, _ := service.Create(ctx, 1, &CreateRequest{
		Title: "Test", Category: domain.CategoryBug, Content: "test",
	})

	att, _ := service.UploadAttachment(ctx, ticket.ID, 1, nil, false, &UploadAttachmentRequest{
		FileName: "file.png", ContentType: "image/png", Size: 100,
		Reader: bytes.NewReader([]byte("data")),
	})

	url, err := service.GetAttachmentURL(ctx, att.ID, 1)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if url != "https://s3.example.com/presigned" {
		t.Errorf("expected presigned URL, got %s", url)
	}
}

func TestGetAttachmentURL_NilStorage(t *testing.T) {
	service, _ := createTestService(t)
	ctx := context.Background()

	_, err := service.GetAttachmentURL(ctx, 1, 1)
	if !errors.Is(err, ErrStorageError) {
		t.Errorf("expected ErrStorageError, got %v", err)
	}
}

func TestGetAttachmentURL_NotFound(t *testing.T) {
	stor := &mockStorage{getURLVal: "url"}
	service, _ := createServiceWithStorage(t, stor, config.StorageConfig{})
	ctx := context.Background()

	_, err := service.GetAttachmentURL(ctx, 99999, 1)
	if !errors.Is(err, ErrAttachmentNotFound) {
		t.Errorf("expected ErrAttachmentNotFound, got %v", err)
	}
}

func TestGetAttachmentURL_WrongUser(t *testing.T) {
	stor := &mockStorage{getURLVal: "url"}
	service, db := createServiceWithStorage(t, stor, config.StorageConfig{MaxFileSize: 10})
	ctx := context.Background()
	createTestUser(t, db, 1, "user@test.com")
	createTestUser(t, db, 2, "other@test.com")

	ticket, _ := service.Create(ctx, 1, &CreateRequest{
		Title: "Test", Category: domain.CategoryBug, Content: "test",
	})

	att, _ := service.UploadAttachment(ctx, ticket.ID, 1, nil, false, &UploadAttachmentRequest{
		FileName: "file.png", ContentType: "image/png", Size: 100,
		Reader: bytes.NewReader([]byte("data")),
	})

	_, err := service.GetAttachmentURL(ctx, att.ID, 2) // wrong user
	if !errors.Is(err, ErrAccessDenied) {
		t.Errorf("expected ErrAccessDenied, got %v", err)
	}
}

// --- AdminGetAttachmentURL tests ---

func TestAdminGetAttachmentURL(t *testing.T) {
	stor := &mockStorage{getURLVal: "https://s3.example.com/admin-url"}
	service, db := createServiceWithStorage(t, stor, config.StorageConfig{MaxFileSize: 10})
	ctx := context.Background()
	createTestUser(t, db, 1, "user@test.com")

	ticket, _ := service.Create(ctx, 1, &CreateRequest{
		Title: "Test", Category: domain.CategoryBug, Content: "test",
	})

	att, _ := service.UploadAttachment(ctx, ticket.ID, 1, nil, false, &UploadAttachmentRequest{
		FileName: "file.png", ContentType: "image/png", Size: 100,
		Reader: bytes.NewReader([]byte("data")),
	})

	url, err := service.AdminGetAttachmentURL(ctx, att.ID)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if url != "https://s3.example.com/admin-url" {
		t.Errorf("expected admin URL, got %s", url)
	}
}

func TestAdminGetAttachmentURL_NilStorage(t *testing.T) {
	service, _ := createTestService(t)
	ctx := context.Background()

	_, err := service.AdminGetAttachmentURL(ctx, 1)
	if !errors.Is(err, ErrStorageError) {
		t.Errorf("expected ErrStorageError, got %v", err)
	}
}

func TestAdminGetAttachmentURL_NotFound(t *testing.T) {
	stor := &mockStorage{getURLVal: "url"}
	service, _ := createServiceWithStorage(t, stor, config.StorageConfig{})
	ctx := context.Background()

	_, err := service.AdminGetAttachmentURL(ctx, 99999)
	if !errors.Is(err, ErrAttachmentNotFound) {
		t.Errorf("expected ErrAttachmentNotFound, got %v", err)
	}
}

// --- GetAttachmentURL storage error tests ---

func TestGetAttachmentURL_StorageGetURLError(t *testing.T) {
	stor := &mockStorage{getURLVal: "url", getURLErr: errors.New("s3 presign error")}
	service, db := createServiceWithStorage(t, stor, config.StorageConfig{MaxFileSize: 10})
	ctx := context.Background()
	createTestUser(t, db, 1, "user@test.com")

	ticket, _ := service.Create(ctx, 1, &CreateRequest{
		Title: "Test", Category: domain.CategoryBug, Content: "test",
	})

	// Upload with a storage that succeeds on upload but fails on GetURL
	stor.getURLErr = nil // temporarily clear for upload
	att, _ := service.UploadAttachment(ctx, ticket.ID, 1, nil, false, &UploadAttachmentRequest{
		FileName: "file.png", ContentType: "image/png", Size: 100,
		Reader: bytes.NewReader([]byte("data")),
	})
	stor.getURLErr = errors.New("s3 presign error") // set error for GetURL

	_, err := service.GetAttachmentURL(ctx, att.ID, 1)
	if err == nil {
		t.Error("expected error from storage.GetURL, got nil")
	}
}

func TestAdminGetAttachmentURL_StorageGetURLError(t *testing.T) {
	stor := &mockStorage{getURLVal: "url"}
	service, db := createServiceWithStorage(t, stor, config.StorageConfig{MaxFileSize: 10})
	ctx := context.Background()
	createTestUser(t, db, 1, "user@test.com")

	ticket, _ := service.Create(ctx, 1, &CreateRequest{
		Title: "Test", Category: domain.CategoryBug, Content: "test",
	})

	att, _ := service.UploadAttachment(ctx, ticket.ID, 1, nil, false, &UploadAttachmentRequest{
		FileName: "file.png", ContentType: "image/png", Size: 100,
		Reader: bytes.NewReader([]byte("data")),
	})
	stor.getURLErr = errors.New("s3 presign error")

	_, err := service.AdminGetAttachmentURL(ctx, att.ID)
	if err == nil {
		t.Error("expected error from storage.GetURL, got nil")
	}
}

// --- UploadAttachment edge cases ---

func TestUploadAttachment_WrongUserWithNonAdminMessage(t *testing.T) {
	stor := &mockStorage{}
	service, db := createServiceWithStorage(t, stor, config.StorageConfig{MaxFileSize: 10})
	ctx := context.Background()
	createTestUser(t, db, 1, "user@test.com")
	createTestUser(t, db, 2, "other@test.com")

	ticket, _ := service.Create(ctx, 1, &CreateRequest{
		Title: "Test", Category: domain.CategoryBug, Content: "test",
	})

	// User 1 adds a normal (non-admin) message
	msg, _ := service.AddMessage(ctx, ticket.ID, 1, &AddMessageRequest{Content: "user msg"})

	// User 2 tries to upload with messageID pointing to a non-admin message
	_, err := service.UploadAttachment(ctx, ticket.ID, 2, &msg.ID, false, &UploadAttachmentRequest{
		FileName: "test.png", ContentType: "image/png", Size: 100,
		Reader: bytes.NewReader([]byte("data")),
	})
	if !errors.Is(err, ErrAccessDenied) {
		t.Errorf("expected ErrAccessDenied for non-admin message, got %v", err)
	}
}

// --- AdminListMessages tests ---

func TestAdminListMessages(t *testing.T) {
	service, db := createTestService(t)
	ctx := context.Background()
	createTestUser(t, db, 1, "user@test.com")
	createTestUser(t, db, 2, "admin@test.com")

	ticket, _ := service.Create(ctx, 1, &CreateRequest{
		Title: "Test", Category: domain.CategoryBug, Content: "initial message",
	})

	service.AdminAddReply(ctx, ticket.ID, 2, &AddMessageRequest{Content: "admin reply"})

	messages, err := service.AdminListMessages(ctx, ticket.ID)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}
	// First should be user message, second admin reply
	if messages[0].IsAdminReply {
		t.Error("first message should not be admin reply")
	}
	if !messages[1].IsAdminReply {
		t.Error("second message should be admin reply")
	}
}

// --- DB error tests (drop table to simulate failures) ---

func TestCreate_DBError(t *testing.T) {
	service, db := createTestService(t)
	ctx := context.Background()

	// Drop the table to cause DB error
	db.Exec("DROP TABLE support_tickets")

	_, err := service.Create(ctx, 1, &CreateRequest{
		Title: "Test", Category: domain.CategoryBug, Content: "test",
	})
	if err == nil {
		t.Error("expected DB error, got nil")
	}
}

func TestCreate_MessageDBError(t *testing.T) {
	service, db := createTestService(t)
	ctx := context.Background()

	// Drop messages table only
	db.Exec("DROP TABLE support_ticket_messages")

	_, err := service.Create(ctx, 1, &CreateRequest{
		Title: "Test", Category: domain.CategoryBug, Content: "test with message",
	})
	if err == nil {
		t.Error("expected DB error for message creation, got nil")
	}
}

func TestListByUser_DBError(t *testing.T) {
	service, db := createTestService(t)
	ctx := context.Background()

	db.Exec("DROP TABLE support_tickets")

	_, err := service.ListByUser(ctx, 1, &ListQuery{})
	if err == nil {
		t.Error("expected DB error, got nil")
	}
}

func TestGetByID_DBError(t *testing.T) {
	service, db := createTestService(t)
	ctx := context.Background()

	db.Exec("DROP TABLE support_tickets")

	_, err := service.GetByID(ctx, 1, 1)
	if err == nil {
		t.Error("expected DB error, got nil")
	}
	// Should not be ErrTicketNotFound since it's a DB error, not record not found
	if errors.Is(err, ErrTicketNotFound) {
		t.Error("expected generic DB error, not ErrTicketNotFound")
	}
}

func TestAddMessage_DBError(t *testing.T) {
	service, db := createTestService(t)
	ctx := context.Background()
	createTestUser(t, db, 1, "user@test.com")

	ticket, _ := service.Create(ctx, 1, &CreateRequest{
		Title: "Test", Category: domain.CategoryBug, Content: "test",
	})

	// Drop messages table after ticket creation
	db.Exec("DROP TABLE support_ticket_messages")

	_, err := service.AddMessage(ctx, ticket.ID, 1, &AddMessageRequest{Content: "msg"})
	if err == nil {
		t.Error("expected DB error, got nil")
	}
}

func TestUploadAttachment_DBError(t *testing.T) {
	stor := &mockStorage{}
	service, db := createServiceWithStorage(t, stor, config.StorageConfig{MaxFileSize: 10})
	ctx := context.Background()
	createTestUser(t, db, 1, "user@test.com")

	ticket, _ := service.Create(ctx, 1, &CreateRequest{
		Title: "Test", Category: domain.CategoryBug, Content: "test",
	})

	// Drop attachments table to cause DB error on create attachment
	db.Exec("DROP TABLE support_ticket_attachments")

	_, err := service.UploadAttachment(ctx, ticket.ID, 1, nil, false, &UploadAttachmentRequest{
		FileName: "test.png", ContentType: "image/png", Size: 100,
		Reader: bytes.NewReader([]byte("data")),
	})
	if err == nil {
		t.Error("expected DB error, got nil")
	}
	// Should have attempted cleanup (delete uploaded file)
	if len(stor.deleted) != 1 {
		t.Errorf("expected 1 cleanup delete call, got %d", len(stor.deleted))
	}
}

func TestUploadAttachment_TicketDBError(t *testing.T) {
	stor := &mockStorage{}
	service, db := createServiceWithStorage(t, stor, config.StorageConfig{MaxFileSize: 10})
	ctx := context.Background()

	// Drop tickets table to cause non-ErrRecordNotFound error
	db.Exec("DROP TABLE support_tickets")

	_, err := service.UploadAttachment(ctx, 1, 1, nil, false, &UploadAttachmentRequest{
		FileName: "test.png", ContentType: "image/png", Size: 100,
		Reader: bytes.NewReader([]byte("data")),
	})
	if err == nil {
		t.Error("expected DB error, got nil")
	}
	if errors.Is(err, ErrTicketNotFound) {
		t.Error("expected generic DB error, not ErrTicketNotFound")
	}
}

func TestGetAttachmentURL_DBError(t *testing.T) {
	stor := &mockStorage{getURLVal: "url"}
	service, db := createServiceWithStorage(t, stor, config.StorageConfig{})
	ctx := context.Background()

	db.Exec("DROP TABLE support_ticket_attachments")

	_, err := service.GetAttachmentURL(ctx, 1, 1)
	if err == nil {
		t.Error("expected DB error, got nil")
	}
	if errors.Is(err, ErrAttachmentNotFound) {
		t.Error("expected generic DB error, not ErrAttachmentNotFound")
	}
}

func TestGetAttachmentURL_TicketDeleted(t *testing.T) {
	stor := &mockStorage{getURLVal: "url"}
	service, db := createServiceWithStorage(t, stor, config.StorageConfig{MaxFileSize: 10})
	ctx := context.Background()
	createTestUser(t, db, 1, "user@test.com")

	ticket, _ := service.Create(ctx, 1, &CreateRequest{
		Title: "Test", Category: domain.CategoryBug, Content: "test",
	})

	att, _ := service.UploadAttachment(ctx, ticket.ID, 1, nil, false, &UploadAttachmentRequest{
		FileName: "file.png", ContentType: "image/png", Size: 100,
		Reader: bytes.NewReader([]byte("data")),
	})

	// Delete the ticket (simulate orphaned attachment)
	db.Exec("DELETE FROM support_tickets WHERE id = ?", ticket.ID)

	_, err := service.GetAttachmentURL(ctx, att.ID, 1)
	if !errors.Is(err, ErrTicketNotFound) {
		t.Errorf("expected ErrTicketNotFound for deleted ticket, got %v", err)
	}
}

func TestAdminGetAttachmentURL_DBError(t *testing.T) {
	stor := &mockStorage{getURLVal: "url"}
	service, db := createServiceWithStorage(t, stor, config.StorageConfig{})
	ctx := context.Background()

	db.Exec("DROP TABLE support_ticket_attachments")

	_, err := service.AdminGetAttachmentURL(ctx, 1)
	if err == nil {
		t.Error("expected DB error, got nil")
	}
	if errors.Is(err, ErrAttachmentNotFound) {
		t.Error("expected generic DB error, not ErrAttachmentNotFound")
	}
}

func TestAdminList_DBError(t *testing.T) {
	service, db := createTestService(t)
	ctx := context.Background()

	db.Exec("DROP TABLE support_tickets")

	_, err := service.AdminList(ctx, &AdminListQuery{})
	if err == nil {
		t.Error("expected DB error, got nil")
	}
}

func TestAdminGetByID_DBError(t *testing.T) {
	service, db := createTestService(t)
	ctx := context.Background()

	db.Exec("DROP TABLE support_tickets")

	_, err := service.AdminGetByID(ctx, 1)
	if err == nil {
		t.Error("expected DB error, got nil")
	}
	if errors.Is(err, ErrTicketNotFound) {
		t.Error("expected generic DB error, not ErrTicketNotFound")
	}
}

func TestAdminAddReply_DBError(t *testing.T) {
	service, db := createTestService(t)
	ctx := context.Background()
	createTestUser(t, db, 1, "user@test.com")

	ticket, _ := service.Create(ctx, 1, &CreateRequest{
		Title: "Test", Category: domain.CategoryBug, Content: "test",
	})

	// Drop messages table
	db.Exec("DROP TABLE support_ticket_messages")

	_, err := service.AdminAddReply(ctx, ticket.ID, 2, &AddMessageRequest{Content: "reply"})
	if err == nil {
		t.Error("expected DB error, got nil")
	}
}

func TestAdminUpdateStatus_DBError(t *testing.T) {
	service, db := createTestService(t)
	ctx := context.Background()

	db.Exec("DROP TABLE support_tickets")

	err := service.AdminUpdateStatus(ctx, 1, domain.StatusResolved)
	if err == nil {
		t.Error("expected DB error, got nil")
	}
}

func TestAdminAssign_DBError(t *testing.T) {
	service, db := createTestService(t)
	ctx := context.Background()

	db.Exec("DROP TABLE support_tickets")

	err := service.AdminAssign(ctx, 1, 2)
	if err == nil {
		t.Error("expected DB error, got nil")
	}
}

func TestAdminGetStats_DBError(t *testing.T) {
	service, db := createTestService(t)
	ctx := context.Background()

	db.Exec("DROP TABLE support_tickets")

	_, err := service.AdminGetStats(ctx)
	if err == nil {
		t.Error("expected DB error, got nil")
	}
}

func TestListMessages_DBError(t *testing.T) {
	service, db := createTestService(t)
	ctx := context.Background()
	createTestUser(t, db, 1, "user@test.com")

	ticket, _ := service.Create(ctx, 1, &CreateRequest{
		Title: "Test", Category: domain.CategoryBug, Content: "test",
	})

	// Drop messages table to trigger listMessagesByTicketID error
	db.Exec("DROP TABLE support_ticket_messages")

	_, err := service.ListMessages(ctx, ticket.ID, 1)
	if err == nil {
		t.Error("expected DB error, got nil")
	}
}

func TestAdminListMessages_DBError(t *testing.T) {
	service, db := createTestService(t)
	ctx := context.Background()

	// Drop messages table
	db.Exec("DROP TABLE support_ticket_messages")

	_, err := service.AdminListMessages(ctx, 1)
	if err == nil {
		t.Error("expected DB error, got nil")
	}
}
