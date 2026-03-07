package infra

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/supportticket"
	"gorm.io/gorm"
)

// supportTicketRepo implements supportticket.Repository using GORM.
type supportTicketRepo struct {
	db *gorm.DB
}

// NewSupportTicketRepository creates a new support-ticket repository.
func NewSupportTicketRepository(db *gorm.DB) supportticket.Repository {
	return &supportTicketRepo{db: db}
}

func (r *supportTicketRepo) CreateTicketWithMessage(ctx context.Context, ticket *supportticket.SupportTicket, message *supportticket.SupportTicketMessage) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(ticket).Error; err != nil {
			return err
		}
		if message != nil {
			message.TicketID = ticket.ID
			if err := tx.Create(message).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *supportTicketRepo) GetByIDAndUser(ctx context.Context, id, userID int64) (*supportticket.SupportTicket, error) {
	var t supportticket.SupportTicket
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&t).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *supportTicketRepo) GetByID(ctx context.Context, id int64) (*supportticket.SupportTicket, error) {
	var t supportticket.SupportTicket
	if err := r.db.WithContext(ctx).Preload("User").Preload("AssignedAdmin").First(&t, id).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *supportTicketRepo) GetTicketByID(ctx context.Context, ticketID int64) (*supportticket.SupportTicket, error) {
	var t supportticket.SupportTicket
	if err := r.db.WithContext(ctx).First(&t, ticketID).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *supportTicketRepo) ListByUser(ctx context.Context, userID int64, status string, limit, offset int) ([]supportticket.SupportTicket, int64, error) {
	q := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if status != "" {
		q = q.Where("status = ?", status)
	}

	var total int64
	if err := q.Model(&supportticket.SupportTicket{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tickets []supportticket.SupportTicket
	if err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&tickets).Error; err != nil {
		return nil, 0, err
	}
	return tickets, total, nil
}

func (r *supportTicketRepo) AdminList(ctx context.Context, search, status, category, priority string, limit, offset int) ([]supportticket.SupportTicket, int64, error) {
	q := r.db.WithContext(ctx).Model(&supportticket.SupportTicket{}).
		Preload("User").Preload("AssignedAdmin")

	if search != "" {
		q = q.Where("title ILIKE ?", "%"+search+"%")
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if priority != "" {
		q = q.Where("priority = ?", priority)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tickets []supportticket.SupportTicket
	if err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&tickets).Error; err != nil {
		return nil, 0, err
	}
	return tickets, total, nil
}

func (r *supportTicketRepo) AddMessageAndReopen(ctx context.Context, msg *supportticket.SupportTicketMessage, ticketID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(msg).Error; err != nil {
			return err
		}
		// Reopen ticket if resolved/closed
		return tx.Model(&supportticket.SupportTicket{}).
			Where("id = ? AND status IN ?", ticketID, []string{supportticket.StatusResolved, supportticket.StatusClosed}).
			Updates(map[string]interface{}{
				"status":     supportticket.StatusOpen,
				"updated_at": time.Now(),
			}).Error
	})
}

func (r *supportTicketRepo) AddAdminReplyAndTransition(ctx context.Context, msg *supportticket.SupportTicketMessage, ticketID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(msg).Error; err != nil {
			return err
		}
		// Auto-transition open -> in_progress on first admin reply
		return tx.Model(&supportticket.SupportTicket{}).
			Where("id = ? AND status = ?", ticketID, supportticket.StatusOpen).
			Updates(map[string]interface{}{
				"status":     supportticket.StatusInProgress,
				"updated_at": time.Now(),
			}).Error
	})
}

func (r *supportTicketRepo) ListMessagesByTicketID(ctx context.Context, ticketID int64) ([]supportticket.SupportTicketMessage, error) {
	var messages []supportticket.SupportTicketMessage
	if err := r.db.WithContext(ctx).
		Where("ticket_id = ?", ticketID).
		Preload("User").
		Preload("Attachments").
		Order("created_at ASC").
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *supportTicketRepo) CreateAttachment(ctx context.Context, attachment *supportticket.SupportTicketAttachment) error {
	return r.db.WithContext(ctx).Create(attachment).Error
}

func (r *supportTicketRepo) GetAttachmentByID(ctx context.Context, attachmentID int64) (*supportticket.SupportTicketAttachment, error) {
	var a supportticket.SupportTicketAttachment
	if err := r.db.WithContext(ctx).First(&a, attachmentID).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *supportTicketRepo) UpdateStatus(ctx context.Context, ticketID int64, currentStatus, newStatus string, updates map[string]interface{}) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&supportticket.SupportTicket{}).
		Where("id = ? AND status = ?", ticketID, currentStatus).
		Updates(updates)
	return result.RowsAffected, result.Error
}

func (r *supportTicketRepo) AssignAdmin(ctx context.Context, ticketID, adminUserID int64) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&supportticket.SupportTicket{}).
		Where("id = ?", ticketID).
		Updates(map[string]interface{}{
			"assigned_admin_id": adminUserID,
			"updated_at":        time.Now(),
		})
	return result.RowsAffected, result.Error
}

func (r *supportTicketRepo) CountByStatus(ctx context.Context, status string) (int64, error) {
	q := r.db.WithContext(ctx).Model(&supportticket.SupportTicket{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var count int64
	err := q.Count(&count).Error
	return count, err
}

// Compile-time interface compliance check
var _ supportticket.Repository = (*supportTicketRepo)(nil)
