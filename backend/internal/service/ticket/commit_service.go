package ticket

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

var ErrCommitNotFound = errors.New("commit not found")

func (s *Service) LinkCommit(ctx context.Context, orgID, ticketID, repoID int64, podID *int64, commitSHA, commitMessage string, commitURL, authorName, authorEmail *string, committedAt *time.Time) (*ticket.Commit, error) {
	commit := &ticket.Commit{
		OrganizationID: orgID,
		TicketID:       ticketID,
		RepositoryID:   repoID,
		PodID:          podID,
		CommitSHA:      commitSHA,
		CommitMessage:  commitMessage,
		CommitURL:      commitURL,
		AuthorName:     authorName,
		AuthorEmail:    authorEmail,
		CommittedAt:    committedAt,
	}

	if err := s.repo.CreateCommit(ctx, commit); err != nil {
		slog.ErrorContext(ctx, "failed to link commit", "ticket_id", ticketID, "commit_sha", commitSHA, "error", err)
		return nil, err
	}
	slog.InfoContext(ctx, "commit linked", "commit_id", commit.ID, "ticket_id", ticketID, "commit_sha", commitSHA)
	return commit, nil
}

func (s *Service) UnlinkCommit(ctx context.Context, commitID int64) error {
	if err := s.repo.DeleteCommit(ctx, commitID); err != nil {
		slog.ErrorContext(ctx, "failed to unlink commit", "commit_id", commitID, "error", err)
		return err
	}
	slog.InfoContext(ctx, "commit unlinked", "commit_id", commitID)
	return nil
}

func (s *Service) ListCommits(ctx context.Context, ticketID int64) ([]*ticket.Commit, error) {
	return s.repo.ListCommitsByTicket(ctx, ticketID)
}

func (s *Service) GetCommitBySHA(ctx context.Context, repoID int64, commitSHA string) (*ticket.Commit, error) {
	commit, err := s.repo.GetCommitBySHA(ctx, repoID, commitSHA)
	if err != nil {
		return nil, err
	}
	if commit == nil {
		return nil, ErrCommitNotFound
	}
	return commit, nil
}
