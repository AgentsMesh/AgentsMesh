package repository

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
)

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

func (s *Service) ListMergeRequests(ctx context.Context, repoID int64, branch, state string) ([]*MergeRequestInfo, error) {
	rows, err := s.repo.ListMergeRequests(ctx, repoID, branch, state)
	if err != nil {
		return nil, err
	}

	result := make([]*MergeRequestInfo, 0, len(rows))
	for _, mr := range rows {
		result = append(result, mrRowToInfo(mr))
	}
	return result, nil
}

func mrRowToInfo(mr gitprovider.MergeRequestRow) *MergeRequestInfo {
	return &MergeRequestInfo{
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
	}
}
