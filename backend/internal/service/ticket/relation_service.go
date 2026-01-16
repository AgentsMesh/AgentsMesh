package ticket

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"gorm.io/gorm"
)

// ========== Ticket Relations ==========

var (
	ErrRelationNotFound = errors.New("relation not found")
	ErrSelfRelation     = errors.New("cannot create relation to self")
)

// GetReverseRelationType returns the reverse relation type
func GetReverseRelationType(relationType string) string {
	switch relationType {
	case ticket.RelationTypeBlocks:
		return ticket.RelationTypeBlockedBy
	case ticket.RelationTypeBlockedBy:
		return ticket.RelationTypeBlocks
	case ticket.RelationTypeDuplicate:
		return ticket.RelationTypeDuplicate
	default:
		return ticket.RelationTypeRelates
	}
}

// CreateRelation creates a relation between two tickets
func (s *Service) CreateRelation(ctx context.Context, orgID, sourceTicketID, targetTicketID int64, relationType string) (*ticket.Relation, error) {
	if sourceTicketID == targetTicketID {
		return nil, ErrSelfRelation
	}

	var result *ticket.Relation
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create the primary relation
		relation := &ticket.Relation{
			OrganizationID: orgID,
			SourceTicketID: sourceTicketID,
			TargetTicketID: targetTicketID,
			RelationType:   relationType,
		}
		if err := tx.Create(relation).Error; err != nil {
			return err
		}
		result = relation

		// Create the reverse relation
		reverseType := GetReverseRelationType(relationType)
		reverseRelation := &ticket.Relation{
			OrganizationID: orgID,
			SourceTicketID: targetTicketID,
			TargetTicketID: sourceTicketID,
			RelationType:   reverseType,
		}
		if err := tx.Create(reverseRelation).Error; err != nil {
			return err
		}

		return nil
	})

	return result, err
}

// DeleteRelation deletes a relation and its reverse
func (s *Service) DeleteRelation(ctx context.Context, relationID int64) error {
	// Get the relation first
	var relation ticket.Relation
	if err := s.db.WithContext(ctx).First(&relation, relationID).Error; err != nil {
		return ErrRelationNotFound
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete the relation
		if err := tx.Delete(&relation).Error; err != nil {
			return err
		}

		// Delete the reverse relation
		reverseType := GetReverseRelationType(relation.RelationType)
		return tx.Where(
			"source_ticket_id = ? AND target_ticket_id = ? AND relation_type = ?",
			relation.TargetTicketID, relation.SourceTicketID, reverseType,
		).Delete(&ticket.Relation{}).Error
	})
}

// ListRelations returns relations for a ticket
func (s *Service) ListRelations(ctx context.Context, ticketID int64) ([]*ticket.Relation, error) {
	var relations []*ticket.Relation
	if err := s.db.WithContext(ctx).
		Preload("TargetTicket").
		Where("source_ticket_id = ?", ticketID).
		Find(&relations).Error; err != nil {
		return nil, err
	}
	return relations, nil
}
