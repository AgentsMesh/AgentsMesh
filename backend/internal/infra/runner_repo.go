package infra

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"gorm.io/gorm"
)

// Compile-time interface check.
var _ runner.RunnerRepository = (*runnerRepository)(nil)

type runnerRepository struct{ db *gorm.DB }

// NewRunnerRepository creates a new RunnerRepository backed by GORM.
func NewRunnerRepository(db *gorm.DB) runner.RunnerRepository {
	return &runnerRepository{db: db}
}

// --- Runner CRUD ---

func (r *runnerRepository) GetByID(ctx context.Context, id int64) (*runner.Runner, error) {
	var out runner.Runner
	if err := r.db.WithContext(ctx).First(&out, id).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &out, nil
}

func (r *runnerRepository) GetByNodeID(ctx context.Context, nodeID string) (*runner.Runner, error) {
	var out runner.Runner
	if err := r.db.WithContext(ctx).Where("node_id = ?", nodeID).First(&out).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &out, nil
}

func (r *runnerRepository) GetByNodeIDAndOrgID(ctx context.Context, nodeID string, orgID int64) (*runner.Runner, error) {
	var out runner.Runner
	if err := r.db.WithContext(ctx).
		Where("node_id = ? AND organization_id = ?", nodeID, orgID).
		First(&out).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &out, nil
}

func (r *runnerRepository) ExistsByNodeIDAndOrg(ctx context.Context, orgID int64, nodeID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&runner.Runner{}).
		Where("organization_id = ? AND node_id = ?", orgID, nodeID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *runnerRepository) Create(ctx context.Context, rn *runner.Runner) error {
	return r.db.WithContext(ctx).Create(rn).Error
}

func (r *runnerRepository) UpdateFields(ctx context.Context, runnerID int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&runner.Runner{}).
		Where("id = ?", runnerID).
		Updates(updates).Error
}

func (r *runnerRepository) UpdateFieldsCAS(ctx context.Context, runnerID int64, casField string, casValue interface{}, updates map[string]interface{}) (int64, error) {
	result := r.db.WithContext(ctx).Model(&runner.Runner{}).
		Where("id = ? AND "+casField+" = ?", runnerID, casValue).
		Updates(updates)
	return result.RowsAffected, result.Error
}

func (r *runnerRepository) Delete(ctx context.Context, runnerID int64) error {
	return r.db.WithContext(ctx).Delete(&runner.Runner{}, runnerID).Error
}

// --- Runner Queries ---

const visibilityFilter = "(visibility = 'organization' OR (visibility = 'private' AND registered_by_user_id = ?))"

func (r *runnerRepository) ListByOrg(ctx context.Context, orgID, userID int64) ([]*runner.Runner, error) {
	var runners []*runner.Runner
	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND "+visibilityFilter, orgID, userID).
		Find(&runners).Error; err != nil {
		return nil, err
	}
	return runners, nil
}

func (r *runnerRepository) ListAvailable(ctx context.Context, orgID, userID int64) ([]*runner.Runner, error) {
	var runners []*runner.Runner
	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND status = ? AND is_enabled = ? AND current_pods < max_concurrent_pods AND "+visibilityFilter,
			orgID, runner.RunnerStatusOnline, true, userID).
		Find(&runners).Error; err != nil {
		return nil, err
	}
	return runners, nil
}

func (r *runnerRepository) ListAvailableOrdered(ctx context.Context, orgID, userID int64) ([]*runner.Runner, error) {
	var runners []*runner.Runner
	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND status = ? AND is_enabled = ? AND current_pods < max_concurrent_pods AND "+visibilityFilter,
			orgID, runner.RunnerStatusOnline, true, userID).
		Order("current_pods ASC").
		Find(&runners).Error; err != nil {
		return nil, err
	}
	return runners, nil
}

func (r *runnerRepository) ListAvailableForAgent(ctx context.Context, orgID, userID int64, agentJSON string) ([]*runner.Runner, error) {
	var runners []*runner.Runner
	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND status = ? AND is_enabled = ? AND current_pods < max_concurrent_pods AND available_agents @> ? AND "+visibilityFilter,
			orgID, runner.RunnerStatusOnline, true, agentJSON, userID).
		Order("current_pods ASC").
		Find(&runners).Error; err != nil {
		return nil, err
	}
	return runners, nil
}

// --- Pod Count ---

func (r *runnerRepository) IncrementPods(ctx context.Context, runnerID int64) error {
	return r.db.WithContext(ctx).Exec(
		"UPDATE runners SET current_pods = current_pods + 1 WHERE id = ?", runnerID,
	).Error
}

func (r *runnerRepository) DecrementPods(ctx context.Context, runnerID int64) error {
	return r.db.WithContext(ctx).Exec(
		"UPDATE runners SET current_pods = GREATEST(current_pods - 1, 0) WHERE id = ?", runnerID,
	).Error
}

func (r *runnerRepository) MarkOfflineRunners(ctx context.Context, threshold time.Time) error {
	return r.db.WithContext(ctx).Model(&runner.Runner{}).
		Where("status = ? AND last_heartbeat < ?", runner.RunnerStatusOnline, threshold).
		Update("status", runner.RunnerStatusOffline).Error
}

func (r *runnerRepository) SetPodCount(ctx context.Context, runnerID int64, count int) error {
	return r.db.WithContext(ctx).Model(&runner.Runner{}).
		Where("id = ?", runnerID).
		Update("current_pods", count).Error
}

func (r *runnerRepository) BatchUpdateHeartbeats(ctx context.Context, items []runner.HeartbeatUpdate) (int, error) {
	updated := 0
	for _, item := range items {
		updates := map[string]interface{}{
			"last_heartbeat": item.Timestamp,
			"current_pods":   item.CurrentPods,
			"status":         item.Status,
		}
		if item.Version != "" {
			updates["runner_version"] = item.Version
		}

		result := r.db.WithContext(ctx).Model(&runner.Runner{}).
			Where("id = ?", item.RunnerID).
			Updates(updates)
		if result.Error != nil {
			continue
		}
		if result.RowsAffected > 0 {
			updated++
		}
	}
	return updated, nil
}

// --- Cross-domain Helpers ---

func (r *runnerRepository) GetOrgSlug(ctx context.Context, orgID int64) (string, error) {
	var org struct{ Slug string }
	if err := r.db.WithContext(ctx).Table("organizations").
		Select("slug").
		Where("id = ?", orgID).
		First(&org).Error; err != nil {
		if isNotFound(err) {
			return "", nil
		}
		return "", err
	}
	return org.Slug, nil
}

func (r *runnerRepository) CountLoopsByRunner(ctx context.Context, runnerID int64) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Raw(
		"SELECT COUNT(*) FROM loops WHERE runner_id = ?", runnerID,
	).Scan(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
