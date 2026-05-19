package supportticket

import "context"

type Repository interface {
	CreateTicketWithMessage(ctx context.Context, ticket *SupportTicket, message *SupportTicketMessage) error
	GetByIDAndUser(ctx context.Context, id, userID int64) (*SupportTicket, error)
	GetByID(ctx context.Context, id int64) (*SupportTicket, error) // admin: with Preload User+AssignedAdmin
	GetTicketByID(ctx context.Context, ticketID int64) (*SupportTicket, error) // plain, no preload
	ListByUser(ctx context.Context, userID int64, status string, limit, offset int) ([]SupportTicket, int64, error)
	AdminList(ctx context.Context, search, status, category, priority string, limit, offset int) ([]SupportTicket, int64, error)

	AddMessageAndReopen(ctx context.Context, msg *SupportTicketMessage, ticketID int64) error
	AddAdminReplyAndTransition(ctx context.Context, msg *SupportTicketMessage, ticketID int64) error
	ListMessagesByTicketID(ctx context.Context, ticketID int64) ([]SupportTicketMessage, error)

	CreateAttachment(ctx context.Context, attachment *SupportTicketAttachment) error
	GetAttachmentByID(ctx context.Context, attachmentID int64) (*SupportTicketAttachment, error)

	UpdateStatus(ctx context.Context, ticketID int64, currentStatus, newStatus string, updates map[string]interface{}) (int64, error)
	AssignAdmin(ctx context.Context, ticketID, adminUserID int64) (int64, error)

	CountByStatus(ctx context.Context, status string) (int64, error)
}
