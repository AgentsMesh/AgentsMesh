package ticket

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/infra/git"
)

// FindOrCreateMR finds or creates an MR record from git provider data.
func (s *MRSyncService) FindOrCreateMR(ctx context.Context, orgID int64, t *ticket.Ticket, mrData *MRData, podID *int64) (*ticket.MergeRequest, error) {
	if mrData.WebURL == "" {
		return nil, errors.New("MR data must contain web URL")
	}

	// Try to find existing MR by URL
	existing, err := s.repo.GetMRByURL(ctx, mrData.WebURL)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		s.updateMRFromData(existing, mrData)
		if podID != nil && existing.PodID == nil {
			existing.PodID = podID
		}
		if err := s.repo.SaveMR(ctx, existing); err != nil {
			return nil, err
		}
		return existing, nil
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

	if err := s.repo.CreateMR(ctx, mr); err != nil {
		slog.Error("failed to create MR record", "ticket_id", t.ID, "mr_url", mrData.WebURL, "error", err)
		return nil, err
	}
	slog.Info("MR record created", "mr_id", mr.ID, "ticket_id", t.ID, "mr_url", mrData.WebURL)
	return mr, nil
}

// GetTicketMRs returns all MRs for a ticket.
func (s *MRSyncService) GetTicketMRs(ctx context.Context, ticketID int64) ([]*ticket.MergeRequest, error) {
	return s.repo.ListMRsByTicket(ctx, ticketID)
}

// GetPodMRs returns all MRs for a pod.
func (s *MRSyncService) GetPodMRs(ctx context.Context, podID int64) ([]*ticket.MergeRequest, error) {
	return s.repo.ListMRsByPod(ctx, podID)
}

// FindTicketByBranch finds a ticket by branch name pattern within an organization.
func (s *MRSyncService) FindTicketByBranch(ctx context.Context, organizationID int64, branchName string) (*ticket.Ticket, error) {
	match := ticketSlugRegex.FindString(branchName)
	if match == "" {
		return nil, nil
	}
	return s.repo.FindTicketByOrgAndSlug(ctx, organizationID, match)
}

// updateMRFromData updates MR record from provider data.
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

// buildMRData converts git provider MR to MRData.
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
