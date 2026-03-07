package infra

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"gorm.io/gorm"
)

// Compile-time interface compliance check.
var _ ticket.MRSyncRepository = (*mrSyncRepository)(nil)

type mrSyncRepository struct{ db *gorm.DB }

// NewMRSyncRepository returns a MRSyncRepository backed by GORM.
func NewMRSyncRepository(db *gorm.DB) ticket.MRSyncRepository {
	return &mrSyncRepository{db: db}
}

// --- MR CRUD ---

func (r *mrSyncRepository) GetMRByURL(ctx context.Context, mrURL string) (*ticket.MergeRequest, error) {
	var mr ticket.MergeRequest
	if err := r.db.WithContext(ctx).Where("mr_url = ?", mrURL).First(&mr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &mr, nil
}

func (r *mrSyncRepository) GetMRByURLWithTicket(ctx context.Context, mrURL string) (*ticket.MergeRequest, error) {
	var mr ticket.MergeRequest
	if err := r.db.WithContext(ctx).
		Preload("Ticket").
		Where("mr_url = ?", mrURL).
		First(&mr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &mr, nil
}

func (r *mrSyncRepository) SaveMR(ctx context.Context, mr *ticket.MergeRequest) error {
	return r.db.WithContext(ctx).Save(mr).Error
}

func (r *mrSyncRepository) CreateMR(ctx context.Context, mr *ticket.MergeRequest) error {
	return r.db.WithContext(ctx).Create(mr).Error
}

func (r *mrSyncRepository) ListMRsByTicket(ctx context.Context, ticketID int64) ([]*ticket.MergeRequest, error) {
	var mrs []*ticket.MergeRequest
	if err := r.db.WithContext(ctx).
		Where("ticket_id = ?", ticketID).
		Order("created_at DESC").
		Find(&mrs).Error; err != nil {
		return nil, err
	}
	return mrs, nil
}

func (r *mrSyncRepository) ListMRsByPod(ctx context.Context, podID int64) ([]*ticket.MergeRequest, error) {
	var mrs []*ticket.MergeRequest
	if err := r.db.WithContext(ctx).
		Where("pod_id = ?", podID).
		Order("created_at DESC").
		Find(&mrs).Error; err != nil {
		return nil, err
	}
	return mrs, nil
}

// --- Ticket look-ups ---

func (r *mrSyncRepository) FindTicketByOrgAndSlug(ctx context.Context, orgID int64, slug string) (*ticket.Ticket, error) {
	var t ticket.Ticket
	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND slug = ?", orgID, slug).
		First(&t).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *mrSyncRepository) GetTicketByID(ctx context.Context, ticketID int64) (*ticket.Ticket, error) {
	var t ticket.Ticket
	if err := r.db.WithContext(ctx).First(&t, ticketID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// --- Cross-domain helpers ---

func (r *mrSyncRepository) GetRepoExternalID(ctx context.Context, repoID int64) (string, error) {
	var repo struct{ ExternalID string }
	if err := r.db.WithContext(ctx).
		Table("repositories").
		Select("external_id").
		Where("id = ?", repoID).
		First(&repo).Error; err != nil {
		return "", err
	}
	return repo.ExternalID, nil
}

func (r *mrSyncRepository) FindPodsWithoutMR(ctx context.Context) ([]*ticket.PodForMRSync, error) {
	subquery := r.db.WithContext(ctx).
		Table("ticket_merge_requests").
		Select("pod_id").
		Where("pod_id IS NOT NULL")

	var pods []agentpod.Pod
	if err := r.db.WithContext(ctx).
		Where("branch_name IS NOT NULL").
		Where("ticket_id IS NOT NULL").
		Where("id NOT IN (?)", subquery).
		Where("status IN ?", []string{agentpod.StatusRunning, agentpod.StatusDisconnected}).
		Find(&pods).Error; err != nil {
		return nil, err
	}

	result := make([]*ticket.PodForMRSync, len(pods))
	for i, p := range pods {
		result[i] = &ticket.PodForMRSync{
			ID:             p.ID,
			OrganizationID: p.OrganizationID,
			BranchName:     p.BranchName,
			TicketID:       p.TicketID,
		}
	}
	return result, nil
}

func (r *mrSyncRepository) ListOpenMRsWithTicket(ctx context.Context) ([]*ticket.MergeRequest, error) {
	var mrs []*ticket.MergeRequest
	if err := r.db.WithContext(ctx).
		Preload("Ticket").
		Where("state != ?", ticket.MRStateMerged).
		Find(&mrs).Error; err != nil {
		return nil, err
	}
	return mrs, nil
}
