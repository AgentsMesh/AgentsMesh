package ticket

import (
	"context"
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

// ========== Ticket Commits ==========

var ErrCommitNotFound = errors.New("commit not found")

// LinkCommit links a git commit to a ticket
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

	if err := s.db.WithContext(ctx).Create(commit).Error; err != nil {
		return nil, err
	}

	return commit, nil
}

// UnlinkCommit removes a commit link from a ticket
func (s *Service) UnlinkCommit(ctx context.Context, commitID int64) error {
	return s.db.WithContext(ctx).Delete(&ticket.Commit{}, commitID).Error
}

// ListCommits returns commits for a ticket
func (s *Service) ListCommits(ctx context.Context, ticketID int64) ([]*ticket.Commit, error) {
	var commits []*ticket.Commit
	if err := s.db.WithContext(ctx).
		Where("ticket_id = ?", ticketID).
		Order("committed_at DESC, created_at DESC").
		Find(&commits).Error; err != nil {
		return nil, err
	}
	return commits, nil
}

// GetCommitBySHA returns a commit by SHA
func (s *Service) GetCommitBySHA(ctx context.Context, repoID int64, commitSHA string) (*ticket.Commit, error) {
	var commit ticket.Commit
	if err := s.db.WithContext(ctx).
		Where("repository_id = ? AND commit_sha = ?", repoID, commitSHA).
		First(&commit).Error; err != nil {
		return nil, ErrCommitNotFound
	}
	return &commit, nil
}
