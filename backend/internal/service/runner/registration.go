package runner

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

// DeleteRunner deletes a runner.
// Blocks deletion if any loops reference this runner (application-level RESTRICT).
func (s *Service) DeleteRunner(ctx context.Context, runnerID int64) error {
	var loopCount int64
	if err := s.db.WithContext(ctx).Raw("SELECT COUNT(*) FROM loops WHERE runner_id = ?", runnerID).Scan(&loopCount).Error; err != nil {
		return err
	}
	if loopCount > 0 {
		return ErrRunnerHasLoopRefs
	}
	return s.db.WithContext(ctx).Delete(&runner.Runner{}, runnerID).Error
}
