package infra

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"gorm.io/gorm"
)

// Compile-time interface compliance check.
var _ ticket.TicketRepository = (*ticketRepository)(nil)

type ticketRepository struct{ db *gorm.DB }

// NewTicketRepository returns a TicketRepository backed by GORM.
func NewTicketRepository(db *gorm.DB) ticket.TicketRepository {
	return &ticketRepository{db: db}
}

// --- Ticket CRUD ---

func (r *ticketRepository) GetByID(ctx context.Context, ticketID int64) (*ticket.Ticket, error) {
	var t ticket.Ticket
	if err := r.db.WithContext(ctx).
		Preload("Assignees.User").
		Preload("Labels").
		Preload("MergeRequests").
		Preload("SubTickets").
		First(&t, ticketID).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *ticketRepository) GetByOrgAndSlug(ctx context.Context, orgID int64, slug string) (*ticket.Ticket, error) {
	var t ticket.Ticket
	if err := r.db.WithContext(ctx).
		Preload("Assignees.User").
		Preload("Labels").
		Preload("MergeRequests").
		Preload("SubTickets").
		Where("organization_id = ? AND slug = ?", orgID, slug).
		First(&t).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *ticketRepository) List(ctx context.Context, f *ticket.TicketListFilter) ([]*ticket.Ticket, int64, error) {
	query := r.db.WithContext(ctx).Model(&ticket.Ticket{}).Where("organization_id = ?", f.OrganizationID)

	if f.RepositoryID != nil {
		query = query.Where("repository_id = ?", *f.RepositoryID)
	}
	if f.Status != "" {
		query = query.Where("status = ?", f.Status)
	}
	if f.Priority != "" {
		query = query.Where("priority = ?", f.Priority)
	}
	if f.ReporterID != nil {
		query = query.Where("reporter_id = ?", *f.ReporterID)
	}
	if f.ParentTicketID != nil {
		query = query.Where("parent_ticket_id = ?", *f.ParentTicketID)
	}
	if f.Query != "" {
		query = query.Where("title ILIKE ? OR slug ILIKE ?", "%"+f.Query+"%", "%"+f.Query+"%")
	}
	if f.AssigneeID != nil {
		query = query.Joins("JOIN ticket_assignees ON ticket_assignees.ticket_id = tickets.id").
			Where("ticket_assignees.user_id = ?", *f.AssigneeID)
	}
	if len(f.LabelIDs) > 0 {
		query = query.Joins("JOIN ticket_labels ON ticket_labels.ticket_id = tickets.id").
			Where("ticket_labels.label_id IN ?", f.LabelIDs)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tickets []*ticket.Ticket
	if err := query.
		Preload("Assignees.User").
		Preload("Labels").
		Order("created_at DESC").
		Limit(f.Limit).
		Offset(f.Offset).
		Find(&tickets).Error; err != nil {
		return nil, 0, err
	}
	return tickets, total, nil
}

func (r *ticketRepository) UpdateFields(ctx context.Context, ticketID int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&ticket.Ticket{}).Where("id = ?", ticketID).Updates(updates).Error
}

func (r *ticketRepository) CreateTicketAtomic(ctx context.Context, p *ticket.CreateTicketParams) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Generate next number (scoped to org + prefix to prevent race conditions)
		var maxNumber int
		likePattern := fmt.Sprintf("%s-%%", p.Prefix)
		if err := tx.Model(&ticket.Ticket{}).
			Where("organization_id = ? AND slug LIKE ?", p.Ticket.OrganizationID, likePattern).
			Select("COALESCE(MAX(number), 0)").
			Scan(&maxNumber).Error; err != nil {
			return err
		}

		p.Ticket.Number = maxNumber + 1
		p.Ticket.Slug = fmt.Sprintf("%s-%d", p.Prefix, p.Ticket.Number)

		if err := tx.Create(p.Ticket).Error; err != nil {
			return err
		}

		for _, uid := range p.AssigneeIDs {
			if err := tx.Create(&ticket.Assignee{TicketID: p.Ticket.ID, UserID: uid}).Error; err != nil {
				return err
			}
		}

		for _, lid := range p.LabelIDs {
			if err := tx.Create(&ticket.TicketLabel{TicketID: p.Ticket.ID, LabelID: lid}).Error; err != nil {
				return err
			}
		}

		for _, name := range p.LabelNames {
			var label ticket.Label
			if err := tx.Where("organization_id = ? AND name = ?", p.Ticket.OrganizationID, name).First(&label).Error; err != nil {
				continue // skip unknown labels
			}
			if err := tx.Create(&ticket.TicketLabel{TicketID: p.Ticket.ID, LabelID: label.ID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *ticketRepository) DeleteTicketAtomic(ctx context.Context, ticketID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("ticket_id = ?", ticketID).Delete(&ticket.Comment{}).Error; err != nil {
			return err
		}
		return tx.Delete(&ticket.Ticket{}, ticketID).Error
	})
}

func (r *ticketRepository) GetRepoTicketPrefix(ctx context.Context, repoID int64) (string, error) {
	var prefix sql.NullString
	if err := r.db.WithContext(ctx).
		Table("repositories").
		Where("id = ?", repoID).
		Select("ticket_prefix").
		Scan(&prefix).Error; err != nil {
		return "", err
	}
	if prefix.Valid {
		return prefix.String, nil
	}
	return "", nil
}

// --- Assignees ---

func (r *ticketRepository) ReplaceAssignees(ctx context.Context, ticketID int64, userIDs []int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("ticket_id = ?", ticketID).Delete(&ticket.Assignee{}).Error; err != nil {
			return err
		}
		for _, uid := range userIDs {
			if err := tx.Create(&ticket.Assignee{TicketID: ticketID, UserID: uid}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *ticketRepository) AddAssignee(ctx context.Context, ticketID, userID int64) error {
	return r.db.WithContext(ctx).Create(&ticket.Assignee{TicketID: ticketID, UserID: userID}).Error
}

func (r *ticketRepository) RemoveAssignee(ctx context.Context, ticketID, userID int64) error {
	return r.db.WithContext(ctx).Where("ticket_id = ? AND user_id = ?", ticketID, userID).Delete(&ticket.Assignee{}).Error
}

func (r *ticketRepository) GetAssigneeUsers(ctx context.Context, ticketID int64) ([]*user.User, error) {
	var assignees []ticket.Assignee
	if err := r.db.WithContext(ctx).Where("ticket_id = ?", ticketID).Find(&assignees).Error; err != nil {
		return nil, err
	}
	if len(assignees) == 0 {
		return []*user.User{}, nil
	}

	ids := make([]int64, len(assignees))
	for i, a := range assignees {
		ids[i] = a.UserID
	}

	var users []*user.User
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
