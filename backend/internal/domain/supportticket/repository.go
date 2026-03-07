package supportticket

import "context"

// Repository defines the data-access interface for support tickets.
// Implementations live in infra/; the service layer depends only on this interface.
type Repository interface {
	// Ticket CRUD
	CreateTicketWithMessage(ctx context.Context, ticket *SupportTicket, message *SupportTicketMessage) error
	GetByIDAndUser(ctx context.Context, id, userID int64) (*SupportTicket, error)
	GetByID(ctx context.Context, id int64) (*SupportTicket, error) // admin: with Preload User+AssignedAdmin
	GetTicketByID(ctx context.Context, ticketID int64) (*SupportTicket, error) // plain, no preload
	ListByUser(ctx context.Context, userID int64, status string, limit, offset int) ([]SupportTicket, int64, error)
	AdminList(ctx context.Context, search, status, category, priority string, limit, offset int) ([]SupportTicket, int64, error)

	// Messages
	AddMessageAndReopen(ctx context.Context, msg *SupportTicketMessage, ticketID int64) error
	AddAdminReplyAndTransition(ctx context.Context, msg *SupportTicketMessage, ticketID int64) error
	ListMessagesByTicketID(ctx context.Context, ticketID int64) ([]SupportTicketMessage, error)

	// Attachments
	CreateAttachment(ctx context.Context, attachment *SupportTicketAttachment) error
	GetAttachmentByID(ctx context.Context, attachmentID int64) (*SupportTicketAttachment, error)

	// Admin operations
	UpdateStatus(ctx context.Context, ticketID int64, currentStatus, newStatus string, updates map[string]interface{}) (int64, error)
	AssignAdmin(ctx context.Context, ticketID, adminUserID int64) (int64, error)

	// Stats
	CountByStatus(ctx context.Context, status string) (int64, error)
}
