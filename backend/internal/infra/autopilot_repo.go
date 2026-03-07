package infra

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"gorm.io/gorm"
)

var _ agentpod.AutopilotRepository = (*autopilotRepo)(nil)

type autopilotRepo struct{ db *gorm.DB }

// NewAutopilotRepository creates a new AutopilotRepository backed by GORM.
func NewAutopilotRepository(db *gorm.DB) agentpod.AutopilotRepository {
	return &autopilotRepo{db: db}
}

func (r *autopilotRepo) Create(ctx context.Context, controller *agentpod.AutopilotController) error {
	return r.db.WithContext(ctx).Create(controller).Error
}

func (r *autopilotRepo) Save(ctx context.Context, controller *agentpod.AutopilotController) error {
	return r.db.WithContext(ctx).Save(controller).Error
}

func (r *autopilotRepo) GetByOrgAndKey(ctx context.Context, orgID int64, key string) (*agentpod.AutopilotController, error) {
	var controller agentpod.AutopilotController
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND autopilot_controller_key = ?", orgID, key).
		First(&controller).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &controller, nil
}

func (r *autopilotRepo) GetByKey(ctx context.Context, key string) (*agentpod.AutopilotController, error) {
	var controller agentpod.AutopilotController
	err := r.db.WithContext(ctx).
		Where("autopilot_controller_key = ?", key).
		First(&controller).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &controller, nil
}

func (r *autopilotRepo) GetActiveForPod(ctx context.Context, podKey string) (*agentpod.AutopilotController, error) {
	var controller agentpod.AutopilotController
	err := r.db.WithContext(ctx).
		Where("pod_key = ? AND phase NOT IN ?", podKey, agentpod.TerminalPhases()).
		First(&controller).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &controller, nil
}

func (r *autopilotRepo) ListByOrg(ctx context.Context, orgID int64) ([]*agentpod.AutopilotController, error) {
	var controllers []*agentpod.AutopilotController
	err := r.db.WithContext(ctx).
		Where("organization_id = ?", orgID).
		Order("created_at DESC").Find(&controllers).Error
	return controllers, err
}

func (r *autopilotRepo) UpdateStatusByKey(ctx context.Context, key string, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&agentpod.AutopilotController{}).
		Where("autopilot_controller_key = ?", key).
		Updates(updates).Error
}

func (r *autopilotRepo) ListIterations(ctx context.Context, controllerID int64) ([]*agentpod.AutopilotIteration, error) {
	var iterations []*agentpod.AutopilotIteration
	err := r.db.WithContext(ctx).
		Where("autopilot_controller_id = ?", controllerID).
		Order("iteration ASC").Find(&iterations).Error
	return iterations, err
}

func (r *autopilotRepo) CreateIteration(ctx context.Context, iteration *agentpod.AutopilotIteration) error {
	return r.db.WithContext(ctx).Create(iteration).Error
}
