package admin

import (
	"context"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

// RunnerListQuery represents query parameters for runner listing
type RunnerListQuery struct {
	Search   string
	Status   string
	OrgID    *int64
	Page     int
	PageSize int
}

// RunnerWithOrg represents a runner with its organization info
type RunnerWithOrg struct {
	Runner       runner.Runner
	Organization *organization.Organization
}

// RunnerListResponse represents paginated runner list response
type RunnerListResponse struct {
	Data       []RunnerWithOrg `json:"data"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}

// ListRunners retrieves runners with filtering and pagination
func (s *Service) ListRunners(ctx context.Context, query *RunnerListQuery) (*RunnerListResponse, error) {
	db := s.db.Model(&runner.Runner{})

	// Apply filters
	if query.Search != "" {
		searchPattern := "%" + query.Search + "%"
		db = db.Where("node_id ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.OrgID != nil {
		db = db.Where("organization_id = ?", *query.OrgID)
	}

	// Count total
	var total int64
	if err := db.Count(&total); err != nil {
		return nil, fmt.Errorf("failed to count runners: %w", err)
	}

	// Apply pagination using helper
	p := normalizePagination(query.Page, query.PageSize, total)

	var runners []runner.Runner
	if err := db.
		Order("created_at DESC").
		Limit(p.PageSize).
		Offset(p.Offset).
		Find(&runners); err != nil {
		return nil, fmt.Errorf("failed to list runners: %w", err)
	}

	// Fetch organizations for the runners
	orgIDs := make(map[int64]bool)
	for _, r := range runners {
		orgIDs[r.OrganizationID] = true
	}

	orgIDList := make([]int64, 0, len(orgIDs))
	for id := range orgIDs {
		orgIDList = append(orgIDList, id)
	}

	orgMap := make(map[int64]*organization.Organization)
	if len(orgIDList) > 0 {
		var orgs []organization.Organization
		if err := s.db.Where("id IN ?", orgIDList).Find(&orgs); err != nil {
			return nil, fmt.Errorf("failed to fetch organizations: %w", err)
		}
		for i := range orgs {
			orgMap[orgs[i].ID] = &orgs[i]
		}
	}

	// Combine runners with organizations
	result := make([]RunnerWithOrg, len(runners))
	for i, r := range runners {
		result[i] = RunnerWithOrg{
			Runner:       r,
			Organization: orgMap[r.OrganizationID],
		}
	}

	return &RunnerListResponse{
		Data:       result,
		Total:      total,
		Page:       p.Page,
		PageSize:   p.PageSize,
		TotalPages: p.TotalPages,
	}, nil
}

// GetRunner retrieves a runner by ID
func (s *Service) GetRunner(ctx context.Context, runnerID int64) (*runner.Runner, error) {
	var r runner.Runner
	if err := s.db.First(&r, runnerID); err != nil {
		return nil, ErrRunnerNotFound
	}
	return &r, nil
}

// GetRunnerWithOrg retrieves a runner with its organization info
func (s *Service) GetRunnerWithOrg(ctx context.Context, runnerID int64) (*RunnerWithOrg, error) {
	var r runner.Runner
	if err := s.db.First(&r, runnerID); err != nil {
		return nil, ErrRunnerNotFound
	}

	var org organization.Organization
	if err := s.db.First(&org, r.OrganizationID); err == nil {
		return &RunnerWithOrg{Runner: r, Organization: &org}, nil
	}

	return &RunnerWithOrg{Runner: r}, nil
}

// DisableRunner disables a runner
func (s *Service) DisableRunner(ctx context.Context, runnerID int64) (*runner.Runner, error) {
	var r runner.Runner
	if err := s.db.First(&r, runnerID); err != nil {
		return nil, ErrRunnerNotFound
	}

	r.IsEnabled = false
	if err := s.db.Save(&r); err != nil {
		return nil, fmt.Errorf("failed to disable runner: %w", err)
	}

	return &r, nil
}

// EnableRunner enables a runner
func (s *Service) EnableRunner(ctx context.Context, runnerID int64) (*runner.Runner, error) {
	var r runner.Runner
	if err := s.db.First(&r, runnerID); err != nil {
		return nil, ErrRunnerNotFound
	}

	r.IsEnabled = true
	if err := s.db.Save(&r); err != nil {
		return nil, fmt.Errorf("failed to enable runner: %w", err)
	}

	return &r, nil
}

// DeleteRunner deletes a runner after checking for active pods and loop references.
func (s *Service) DeleteRunner(ctx context.Context, runnerID int64) (*runner.Runner, error) {
	var r runner.Runner
	if err := s.db.First(&r, runnerID); err != nil {
		return nil, ErrRunnerNotFound
	}

	// Check for active pods before deletion
	var podCount int64
	if err := s.db.Model(&agentpod.Pod{}).
		Where("runner_id = ? AND status IN ?", runnerID, agentpod.ActiveStatuses()).
		Count(&podCount); err != nil {
		return nil, fmt.Errorf("failed to check pods: %w", err)
	}
	if podCount > 0 {
		return nil, ErrRunnerHasActivePods
	}

	// Check for loop references before deletion (application-level RESTRICT)
	var loopCount int64
	s.db.GormDB().Raw("SELECT COUNT(*) FROM loops WHERE runner_id = ?", runnerID).Scan(&loopCount)
	if loopCount > 0 {
		return nil, ErrRunnerHasLoopRefs
	}

	if err := s.db.Delete(&r); err != nil {
		return nil, fmt.Errorf("failed to delete runner: %w", err)
	}

	return &r, nil
}
