package runner

import (
	"context"
)

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
