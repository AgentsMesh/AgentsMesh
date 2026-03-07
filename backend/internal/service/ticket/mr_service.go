package ticket

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

// ========== Merge Request Operations ==========

// LinkMergeRequest links a merge request to a ticket.
func (s *Service) LinkMergeRequest(ctx context.Context, orgID, ticketID int64, podID *int64, mrIID int, mrURL, sourceBranch, targetBranch, title, state string) (*ticket.MergeRequest, error) {
	mr := &ticket.MergeRequest{
		OrganizationID: orgID,
		TicketID:       &ticketID,
		PodID:          podID,
		MRIID:          mrIID,
		MRURL:          mrURL,
		SourceBranch:   sourceBranch,
		TargetBranch:   targetBranch,
		Title:          title,
		State:          state,
	}

	if err := s.repo.CreateMR(ctx, mr); err != nil {
		return nil, err
	}
	return mr, nil
}

// UpdateMergeRequestState updates a merge request state.
func (s *Service) UpdateMergeRequestState(ctx context.Context, mrID int64, state string) error {
	return s.repo.UpdateMRState(ctx, mrID, state)
}

// GetMergeRequestByURL returns a merge request by URL.
func (s *Service) GetMergeRequestByURL(ctx context.Context, mrURL string) (*ticket.MergeRequest, error) {
	mr, err := s.repo.GetMRByURL(ctx, mrURL)
	if err != nil {
		return nil, err
	}
	if mr == nil {
		return nil, ErrMRNotFound
	}
	return mr, nil
}

// ListMergeRequests returns merge requests for a ticket.
func (s *Service) ListMergeRequests(ctx context.Context, ticketID int64) ([]*ticket.MergeRequest, error) {
	return s.repo.ListMRsByTicket(ctx, ticketID)
}
