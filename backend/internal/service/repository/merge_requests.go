package repository

import (
	"context"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/ticket"
)

// MergeRequestInfo represents merge request information returned by the service
type MergeRequestInfo struct {
	ID             int64
	MRIID          int
	Title          string
	State          string
	MRURL          string
	SourceBranch   string
	TargetBranch   string
	PipelineStatus *string
	PipelineID     *int64
	PipelineURL    *string
	TicketID       *int64
	PodID          *int64
}

// ListMergeRequests lists merge requests for a repository
// branch: optional filter by source branch
// state: filter by state (opened, merged, closed, all)
func (s *Service) ListMergeRequests(ctx context.Context, repoID int64, branch, state string) ([]*MergeRequestInfo, error) {
	// Build query
	query := s.db.WithContext(ctx).Model(&ticket.MergeRequest{}).Where("repository_id = ?", repoID)

	// Filter by branch if provided
	if branch != "" {
		query = query.Where("source_branch = ?", branch)
	}

	// Filter by state if not "all"
	if state != "" && state != "all" {
		query = query.Where("state = ?", state)
	}

	// Order by created_at desc
	query = query.Order("created_at DESC")

	var mrs []ticket.MergeRequest
	if err := query.Find(&mrs).Error; err != nil {
		return nil, err
	}

	// Transform to MergeRequestInfo
	result := make([]*MergeRequestInfo, 0, len(mrs))
	for _, mr := range mrs {
		result = append(result, &MergeRequestInfo{
			ID:             mr.ID,
			MRIID:          mr.MRIID,
			Title:          mr.Title,
			State:          mr.State,
			MRURL:          mr.MRURL,
			SourceBranch:   mr.SourceBranch,
			TargetBranch:   mr.TargetBranch,
			PipelineStatus: mr.PipelineStatus,
			PipelineID:     mr.PipelineID,
			PipelineURL:    mr.PipelineURL,
			TicketID:       mr.TicketID,
			PodID:          mr.PodID,
		})
	}

	return result, nil
}
