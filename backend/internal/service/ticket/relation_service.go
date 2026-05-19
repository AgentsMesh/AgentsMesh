package ticket

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

var (
	ErrRelationNotFound = errors.New("relation not found")
	ErrSelfRelation     = errors.New("cannot create relation to self")
)

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

func (s *Service) CreateRelation(ctx context.Context, orgID, sourceTicketID, targetTicketID int64, relationType string) (*ticket.Relation, error) {
	if sourceTicketID == targetTicketID {
		return nil, ErrSelfRelation
	}

	relation := &ticket.Relation{
		OrganizationID: orgID,
		SourceTicketID: sourceTicketID,
		TargetTicketID: targetTicketID,
		RelationType:   relationType,
	}

	reverseRelation := &ticket.Relation{
		OrganizationID: orgID,
		SourceTicketID: targetTicketID,
		TargetTicketID: sourceTicketID,
		RelationType:   GetReverseRelationType(relationType),
	}

	if err := s.repo.CreateRelationPair(ctx, relation, reverseRelation); err != nil {
		return nil, err
	}
	return relation, nil
}

func (s *Service) DeleteRelation(ctx context.Context, relationID int64) error {
	relation, err := s.repo.GetRelation(ctx, relationID)
	if err != nil {
		return err
	}
	if relation == nil {
		return ErrRelationNotFound
	}

	reverseType := GetReverseRelationType(relation.RelationType)
	return s.repo.DeleteRelationPair(ctx, relation, reverseType)
}

func (s *Service) ListRelations(ctx context.Context, ticketID int64) ([]*ticket.Relation, error) {
	return s.repo.ListRelations(ctx, ticketID)
}
