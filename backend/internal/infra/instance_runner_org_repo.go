package infra

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/service/instance"
	"gorm.io/gorm"
)

// Compile-time interface compliance check.
var _ instance.RunnerOrgQuerier = (*runnerOrgQuerier)(nil)

type runnerOrgQuerier struct{ db *gorm.DB }

// NewRunnerOrgQuerier creates a new GORM-backed RunnerOrgQuerier.
func NewRunnerOrgQuerier(db *gorm.DB) instance.RunnerOrgQuerier {
	return &runnerOrgQuerier{db: db}
}

func (r *runnerOrgQuerier) GetOrgIDsByRunnerIDs(ctx context.Context, runnerIDs []int64) ([]int64, error) {
	var orgIDs []int64
	err := r.db.WithContext(ctx).
		Table("runners").
		Where("id IN ?", runnerIDs).
		Distinct("organization_id").
		Pluck("organization_id", &orgIDs).Error
	return orgIDs, err
}
