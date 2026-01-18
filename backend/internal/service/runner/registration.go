package runner

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

// DeleteRunner deletes a runner
func (s *Service) DeleteRunner(ctx context.Context, runnerID int64) error {
	return s.db.WithContext(ctx).Delete(&runner.Runner{}, runnerID).Error
}
