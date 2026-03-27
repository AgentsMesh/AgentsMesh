package supportticket

import (
	"bytes"
	"context"
	"errors"
	"testing"

	domain "github.com/anthropics/agentsmesh/backend/internal/domain/supportticket"
)

// --- GetAttachmentURL tests ---

func TestGetAttachmentURL(t *testing.T) {
	stor := &mockStorage{getURLVal: "https://s3.example.com/presigned"}
	service, db := createServiceWithStorage(t, stor, testStorageCfg(10))
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
	service, _ := createServiceWithStorage(t, stor, testStorageCfg(0))
	ctx := context.Background()

	_, err := service.GetAttachmentURL(ctx, 99999, 1)
	if !errors.Is(err, ErrAttachmentNotFound) {
		t.Errorf("expected ErrAttachmentNotFound, got %v", err)
	}
}

func TestGetAttachmentURL_WrongUser(t *testing.T) {
	stor := &mockStorage{getURLVal: "url"}
	service, db := createServiceWithStorage(t, stor, testStorageCfg(10))
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
	service, db := createServiceWithStorage(t, stor, testStorageCfg(10))
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
	service, _ := createServiceWithStorage(t, stor, testStorageCfg(0))
	ctx := context.Background()

	_, err := service.AdminGetAttachmentURL(ctx, 99999)
	if !errors.Is(err, ErrAttachmentNotFound) {
		t.Errorf("expected ErrAttachmentNotFound, got %v", err)
	}
}

// --- GetAttachmentURL storage error tests ---

func TestGetAttachmentURL_StorageGetURLError(t *testing.T) {
	stor := &mockStorage{getURLVal: "url", getURLErr: errors.New("s3 presign error")}
	service, db := createServiceWithStorage(t, stor, testStorageCfg(10))
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
	service, db := createServiceWithStorage(t, stor, testStorageCfg(10))
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
	service, db := createServiceWithStorage(t, stor, testStorageCfg(10))
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
