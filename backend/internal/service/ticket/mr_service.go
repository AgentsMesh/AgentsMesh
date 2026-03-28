package ticket

import (
	"context"
	"log/slog"

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
		slog.Error("failed to link merge request", "ticket_id", ticketID, "mr_url", mrURL, "error", err)
		return nil, err
	}
	slog.Info("merge request linked", "mr_id", mr.ID, "ticket_id", ticketID, "mr_url", mrURL)
	return mr, nil
}

// UpdateMergeRequestState updates a merge request state.
func (s *Service) UpdateMergeRequestState(ctx context.Context, mrID int64, state string) error {
	if err := s.repo.UpdateMRState(ctx, mrID, state); err != nil {
		slog.Error("failed to update merge request state", "mr_id", mrID, "state", state, "error", err)
		return err
	}
	slog.Info("merge request state updated", "mr_id", mrID, "state", state)
	return nil
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
