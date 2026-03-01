package ticket

import (
	"context"
	"errors"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/ticket"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/infra/git"
	"gorm.io/gorm"
)

// FindOrCreateMR finds or creates an MR record from git provider data
func (s *MRSyncService) FindOrCreateMR(ctx context.Context, orgID int64, t *ticket.Ticket, mrData *MRData, podID *int64) (*ticket.MergeRequest, error) {
	if mrData.WebURL == "" {
		return nil, errors.New("MR data must contain web URL")
	}

	// Try to find existing MR by URL
	var existing ticket.MergeRequest
	err := s.db.WithContext(ctx).
		Where("mr_url = ?", mrData.WebURL).
		First(&existing).Error

	if err == nil {
		// Update existing record
		s.updateMRFromData(&existing, mrData)
		if podID != nil && existing.PodID == nil {
			existing.PodID = podID
		}
		if err := s.db.WithContext(ctx).Save(&existing).Error; err != nil {
			return nil, err
		}
		return &existing, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Create new record
	now := time.Now()
	mr := &ticket.MergeRequest{
		OrganizationID: orgID,
		TicketID:       &t.ID,
		PodID:          podID,
		MRIID:          mrData.IID,
		MRURL:          mrData.WebURL,
		SourceBranch:   mrData.SourceBranch,
		TargetBranch:   mrData.TargetBranch,
		Title:          mrData.Title,
		State:          mrData.State,
		PipelineStatus: mrData.PipelineStatus,
		PipelineID:     mrData.PipelineID,
		PipelineURL:    mrData.PipelineURL,
		MergeCommitSHA: mrData.MergeCommitSHA,
		MergedAt:       mrData.MergedAt,
		LastSyncedAt:   &now,
	}

	if err := s.db.WithContext(ctx).Create(mr).Error; err != nil {
		return nil, err
	}

	return mr, nil
}

// GetTicketMRs returns all MRs for a ticket
func (s *MRSyncService) GetTicketMRs(ctx context.Context, ticketID int64) ([]*ticket.MergeRequest, error) {
	var mrs []*ticket.MergeRequest
	if err := s.db.WithContext(ctx).
		Where("ticket_id = ?", ticketID).
		Order("created_at DESC").
		Find(&mrs).Error; err != nil {
		return nil, err
	}
	return mrs, nil
}

// GetPodMRs returns all MRs for a pod
func (s *MRSyncService) GetPodMRs(ctx context.Context, podID int64) ([]*ticket.MergeRequest, error) {
	var mrs []*ticket.MergeRequest
	if err := s.db.WithContext(ctx).
		Where("pod_id = ?", podID).
		Order("created_at DESC").
		Find(&mrs).Error; err != nil {
		return nil, err
	}
	return mrs, nil
}

// FindTicketByBranch finds a ticket by branch name pattern within an organization
func (s *MRSyncService) FindTicketByBranch(ctx context.Context, organizationID int64, branchName string) (*ticket.Ticket, error) {
	match := ticketSlugRegex.FindString(branchName)
	if match == "" {
		return nil, nil
	}

	var t ticket.Ticket
	if err := s.db.WithContext(ctx).
		Where("organization_id = ? AND slug = ?", organizationID, match).
		First(&t).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &t, nil
}

// updateMRFromData updates MR record from provider data
func (s *MRSyncService) updateMRFromData(mr *ticket.MergeRequest, data *MRData) {
	mr.Title = data.Title
	mr.State = data.State
	mr.PipelineStatus = data.PipelineStatus
	mr.PipelineID = data.PipelineID
	mr.PipelineURL = data.PipelineURL
	mr.MergeCommitSHA = data.MergeCommitSHA
	mr.MergedAt = data.MergedAt
	now := time.Now()
	mr.LastSyncedAt = &now
}

// buildMRData converts git provider MR to MRData
func (s *MRSyncService) buildMRData(mr *git.MergeRequest) *MRData {
	data := &MRData{
		IID:          mr.IID,
		WebURL:       mr.WebURL,
		Title:        mr.Title,
		SourceBranch: mr.SourceBranch,
		TargetBranch: mr.TargetBranch,
		State:        mr.State,
	}

	if mr.PipelineStatus != "" {
		data.PipelineStatus = &mr.PipelineStatus
	}
	if mr.PipelineID != 0 {
		id := int64(mr.PipelineID)
		data.PipelineID = &id
	}
	if mr.PipelineURL != "" {
		data.PipelineURL = &mr.PipelineURL
	}
	if mr.MergeCommitSHA != "" {
		data.MergeCommitSHA = &mr.MergeCommitSHA
	}
	if mr.MergedAt != nil {
		data.MergedAt = mr.MergedAt
	}

	return data
}
