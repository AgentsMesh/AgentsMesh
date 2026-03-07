package runner

import (
	"context"
)

// DeleteRunner deletes a runner.
// Blocks deletion if any loops reference this runner (application-level RESTRICT).
func (s *Service) DeleteRunner(ctx context.Context, runnerID int64) error {
	loopCount, err := s.repo.CountLoopsByRunner(ctx, runnerID)
	if err != nil {
		return err
	}
	if loopCount > 0 {
		return ErrRunnerHasLoopRefs
	}
	return s.repo.Delete(ctx, runnerID)
}
