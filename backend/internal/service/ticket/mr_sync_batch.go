package ticket

import (
	"context"
	"errors"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/agentpod"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/ticket"
	"gorm.io/gorm"
)

// CheckPodForNewMR checks if a pod's branch has an MR
func (s *MRSyncService) CheckPodForNewMR(ctx context.Context, pod *agentpod.Pod) (*ticket.MergeRequest, error) {
	if pod.BranchName == nil || pod.TicketID == nil {
		return nil, nil
	}

	if s.gitProvider == nil {
		return nil, ErrNoGitProvider
	}

	// Get ticket with repository
	var t ticket.Ticket
	if err := s.db.WithContext(ctx).
		Preload("Repository").
		First(&t, *pod.TicketID).Error; err != nil {
		return nil, err
	}

	if t.RepositoryID == nil {
		return nil, ErrNoRepositoryLink
	}

	// Get repository info
	var repo struct {
		ExternalID string
	}
	if err := s.db.WithContext(ctx).
		Table("repositories").
		Select("external_id").
		Where("id = ?", *t.RepositoryID).
		First(&repo).Error; err != nil {
		return nil, err
	}

	// Fetch MRs from git provider
	mrs, err := s.gitProvider.ListMergeRequestsByBranch(ctx, repo.ExternalID, *pod.BranchName, "all")
	if err != nil {
		return nil, err
	}

	if len(mrs) == 0 {
		return nil, nil
	}

	// Convert to MRData
	mr := mrs[0]
	mrData := s.buildMRData(mr)

	return s.FindOrCreateMR(ctx, pod.OrganizationID, &t, mrData, &pod.ID)
}

// BatchCheckPods checks active pods for new MRs
func (s *MRSyncService) BatchCheckPods(ctx context.Context) ([]*ticket.MergeRequest, error) {
	if s.gitProvider == nil {
		return nil, ErrNoGitProvider
	}

	// Find pods with branch but no MR record
	var pods []*agentpod.Pod
	subquery := s.db.WithContext(ctx).
		Table("ticket_merge_requests").
		Select("pod_id").
		Where("pod_id IS NOT NULL")

	err := s.db.WithContext(ctx).
		Where("branch_name IS NOT NULL").
		Where("ticket_id IS NOT NULL").
		Where("id NOT IN (?)", subquery).
		Where("status IN ?", []string{
			agentpod.StatusRunning,
			agentpod.StatusDisconnected,
		}).
		Find(&pods).Error

	if err != nil {
		return nil, err
	}

	var newMRs []*ticket.MergeRequest
	for _, pod := range pods {
		mr, err := s.CheckPodForNewMR(ctx, pod)
		if err != nil {
			continue // Log and continue
		}
		if mr != nil {
			newMRs = append(newMRs, mr)
		}
	}

	return newMRs, nil
}

// BatchSyncMRStatus syncs status for open MRs
func (s *MRSyncService) BatchSyncMRStatus(ctx context.Context) ([]*ticket.MergeRequest, error) {
	if s.gitProvider == nil {
		return nil, ErrNoGitProvider
	}

	// Find non-merged MRs
	var mrs []*ticket.MergeRequest
	err := s.db.WithContext(ctx).
		Preload("Ticket").
		Where("state != ?", ticket.MRStateMerged).
		Find(&mrs).Error

	if err != nil {
		return nil, err
	}

	var updated []*ticket.MergeRequest
	for _, mr := range mrs {
		if mr.Ticket == nil || mr.Ticket.RepositoryID == nil {
			continue
		}

		// Get repository info
		var repo struct {
			ExternalID string
		}
		if err := s.db.WithContext(ctx).
			Table("repositories").
			Select("external_id").
			Where("id = ?", *mr.Ticket.RepositoryID).
			First(&repo).Error; err != nil {
			continue
		}

		// Fetch MR from git provider
		mrInfo, err := s.gitProvider.GetMergeRequest(ctx, repo.ExternalID, mr.MRIID)
		if err != nil {
			continue
		}

		mrData := s.buildMRData(mrInfo)
		s.updateMRFromData(mr, mrData)

		if err := s.db.WithContext(ctx).Save(mr).Error; err != nil {
			continue
		}

		updated = append(updated, mr)
	}

	return updated, nil
}

// SyncMRByURL syncs a single MR by its URL
func (s *MRSyncService) SyncMRByURL(ctx context.Context, mrURL string) (*ticket.MergeRequest, error) {
	var mr ticket.MergeRequest
	if err := s.db.WithContext(ctx).
		Preload("Ticket").
		Where("mr_url = ?", mrURL).
		First(&mr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMRNotFound
		}
		return nil, err
	}

	if mr.Ticket == nil || mr.Ticket.RepositoryID == nil {
		return nil, ErrNoRepositoryLink
	}

	// Get repository info
	var repo struct {
		ExternalID string
	}
	if err := s.db.WithContext(ctx).
		Table("repositories").
		Select("external_id").
		Where("id = ?", *mr.Ticket.RepositoryID).
		First(&repo).Error; err != nil {
		return nil, err
	}

	// Fetch MR from git provider
	mrInfo, err := s.gitProvider.GetMergeRequest(ctx, repo.ExternalID, mr.MRIID)
	if err != nil {
		return nil, err
	}

	mrData := s.buildMRData(mrInfo)
	s.updateMRFromData(&mr, mrData)

	if err := s.db.WithContext(ctx).Save(&mr).Error; err != nil {
		return nil, err
	}

	return &mr, nil
}