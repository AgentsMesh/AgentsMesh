package ticket

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"gorm.io/gorm"
)

// ========== Assignee Operations ==========

// UpdateAssignees updates ticket assignees
func (s *Service) UpdateAssignees(ctx context.Context, ticketID int64, userIDs []int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Remove existing assignees
		if err := tx.Where("ticket_id = ?", ticketID).Delete(&ticket.Assignee{}).Error; err != nil {
			return err
		}
		// Add new assignees
		for _, userID := range userIDs {
			assignee := &ticket.Assignee{
				TicketID: ticketID,
				UserID:   userID,
			}
			if err := tx.Create(assignee).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// AddAssignee adds an assignee to a ticket
func (s *Service) AddAssignee(ctx context.Context, ticketID, userID int64) error {
	assignee := &ticket.Assignee{
		TicketID: ticketID,
		UserID:   userID,
	}
	return s.db.WithContext(ctx).Create(assignee).Error
}

// RemoveAssignee removes an assignee from a ticket
func (s *Service) RemoveAssignee(ctx context.Context, ticketID, userID int64) error {
	return s.db.WithContext(ctx).Where("ticket_id = ? AND user_id = ?", ticketID, userID).Delete(&ticket.Assignee{}).Error
}

// GetAssignees returns assignees for a ticket
func (s *Service) GetAssignees(ctx context.Context, ticketID int64) ([]*user.User, error) {
	var assignees []ticket.Assignee
	if err := s.db.WithContext(ctx).Where("ticket_id = ?", ticketID).Find(&assignees).Error; err != nil {
		return nil, err
	}

	if len(assignees) == 0 {
		return []*user.User{}, nil
	}

	userIDs := make([]int64, len(assignees))
	for i, a := range assignees {
		userIDs[i] = a.UserID
	}

	var users []*user.User
	if err := s.db.WithContext(ctx).Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}
