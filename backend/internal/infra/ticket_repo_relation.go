package infra

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"gorm.io/gorm"
)

// --- Relation, Commit, and MR methods for ticketRepository ---

// Relations

func (r *ticketRepository) GetRelation(ctx context.Context, relationID int64) (*ticket.Relation, error) {
	var rel ticket.Relation
	if err := r.db.WithContext(ctx).First(&rel, relationID).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &rel, nil
}

func (r *ticketRepository) CreateRelationPair(ctx context.Context, relation, reverse *ticket.Relation) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(relation).Error; err != nil {
			return err
		}
		return tx.Create(reverse).Error
	})
}

func (r *ticketRepository) DeleteRelationPair(ctx context.Context, relation *ticket.Relation, reverseType string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(relation).Error; err != nil {
			return err
		}
		return tx.Where(
			"source_ticket_id = ? AND target_ticket_id = ? AND relation_type = ?",
			relation.TargetTicketID, relation.SourceTicketID, reverseType,
		).Delete(&ticket.Relation{}).Error
	})
}

func (r *ticketRepository) ListRelations(ctx context.Context, ticketID int64) ([]*ticket.Relation, error) {
	var relations []*ticket.Relation
	if err := r.db.WithContext(ctx).
		Preload("TargetTicket").
		Where("source_ticket_id = ?", ticketID).
		Find(&relations).Error; err != nil {
		return nil, err
	}
	return relations, nil
}

// Commits

func (r *ticketRepository) CreateCommit(ctx context.Context, commit *ticket.Commit) error {
	return r.db.WithContext(ctx).Create(commit).Error
}

func (r *ticketRepository) DeleteCommit(ctx context.Context, commitID int64) error {
	return r.db.WithContext(ctx).Delete(&ticket.Commit{}, commitID).Error
}

func (r *ticketRepository) ListCommitsByTicket(ctx context.Context, ticketID int64) ([]*ticket.Commit, error) {
	var commits []*ticket.Commit
	if err := r.db.WithContext(ctx).
		Where("ticket_id = ?", ticketID).
		Order("committed_at DESC, created_at DESC").
		Find(&commits).Error; err != nil {
		return nil, err
	}
	return commits, nil
}

func (r *ticketRepository) GetCommitBySHA(ctx context.Context, repoID int64, sha string) (*ticket.Commit, error) {
	var commit ticket.Commit
	if err := r.db.WithContext(ctx).
		Where("repository_id = ? AND commit_sha = ?", repoID, sha).
		First(&commit).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &commit, nil
}

// Merge Requests

func (r *ticketRepository) CreateMR(ctx context.Context, mr *ticket.MergeRequest) error {
	return r.db.WithContext(ctx).Create(mr).Error
}

func (r *ticketRepository) UpdateMRState(ctx context.Context, mrID int64, state string) error {
	return r.db.WithContext(ctx).
		Model(&ticket.MergeRequest{}).
		Where("id = ?", mrID).
		Update("state", state).Error
}

func (r *ticketRepository) GetMRByURL(ctx context.Context, mrURL string) (*ticket.MergeRequest, error) {
	var mr ticket.MergeRequest
	if err := r.db.WithContext(ctx).Where("mr_url = ?", mrURL).First(&mr).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &mr, nil
}

func (r *ticketRepository) ListMRsByTicket(ctx context.Context, ticketID int64) ([]*ticket.MergeRequest, error) {
	var mrs []*ticket.MergeRequest
	if err := r.db.WithContext(ctx).Where("ticket_id = ?", ticketID).Find(&mrs).Error; err != nil {
		return nil, err
	}
	return mrs, nil
}
