package supportticket

import (
	"bytes"
	"context"
	"errors"
	"testing"

	domain "github.com/anthropics/agentsmesh/backend/internal/domain/supportticket"
)

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
	service, db := createServiceWithStorage(t, stor, testStorageCfg(10))
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
	service, db := createServiceWithStorage(t, stor, testStorageCfg(10))
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
	service, db := createServiceWithStorage(t, stor, testStorageCfg(0))
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

	// Delete the ticket (simulate orphaned attachment)
	db.Exec("DELETE FROM support_tickets WHERE id = ?", ticket.ID)

	_, err := service.GetAttachmentURL(ctx, att.ID, 1)
	if !errors.Is(err, ErrTicketNotFound) {
		t.Errorf("expected ErrTicketNotFound for deleted ticket, got %v", err)
	}
}

func TestAdminGetAttachmentURL_DBError(t *testing.T) {
	stor := &mockStorage{getURLVal: "url"}
	service, db := createServiceWithStorage(t, stor, testStorageCfg(0))
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
