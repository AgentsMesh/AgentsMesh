package infra

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"gorm.io/gorm"
)

// Compile-time interface check
var _ agent.AgentRepository = (*agentRepo)(nil)

type agentRepo struct {
	db *gorm.DB
}

// NewAgentRepository creates a new GORM-based agent repository
func NewAgentRepository(db *gorm.DB) agent.AgentRepository {
	return &agentRepo{db: db}
}

func (r *agentRepo) ListBuiltinActive(ctx context.Context) ([]*agent.Agent, error) {
	var types []*agent.Agent
	err := r.db.WithContext(ctx).Where("is_builtin = ? AND is_active = ?", true, true).Find(&types).Error
	return types, err
}

func (r *agentRepo) ListAllActive(ctx context.Context) ([]*agent.Agent, error) {
	var types []*agent.Agent
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Find(&types).Error
	return types, err
}

func (r *agentRepo) GetBySlug(ctx context.Context, slug string) (*agent.Agent, error) {
	var a agent.Agent
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&a).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *agentRepo) ListCustomByOrg(ctx context.Context, orgID int64) ([]*agent.CustomAgent, error) {
	var types []*agent.CustomAgent
	err := r.db.WithContext(ctx).Where("organization_id = ? AND is_active = ?", orgID, true).Find(&types).Error
	return types, err
}

func (r *agentRepo) GetCustomBySlug(ctx context.Context, orgID int64, slug string) (*agent.CustomAgent, error) {
	var custom agent.CustomAgent
	if err := r.db.WithContext(ctx).Where("organization_id = ? AND slug = ?", orgID, slug).First(&custom).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &custom, nil
}

func (r *agentRepo) CustomSlugExists(ctx context.Context, orgID int64, slug string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&agent.CustomAgent{}).
		Where("organization_id = ? AND slug = ?", orgID, slug).
		Count(&count).Error
	return count > 0, err
}

func (r *agentRepo) CreateCustom(ctx context.Context, custom *agent.CustomAgent) error {
	return r.db.WithContext(ctx).Create(custom).Error
}

func (r *agentRepo) UpdateCustom(ctx context.Context, orgID int64, slug string, updates map[string]interface{}) (*agent.CustomAgent, error) {
	if err := r.db.WithContext(ctx).Model(&agent.CustomAgent{}).Where("organization_id = ? AND slug = ?", orgID, slug).Updates(updates).Error; err != nil {
		return nil, err
	}
	var custom agent.CustomAgent
	if err := r.db.WithContext(ctx).Where("organization_id = ? AND slug = ?", orgID, slug).First(&custom).Error; err != nil {
		return nil, err
	}
	return &custom, nil
}

func (r *agentRepo) DeleteCustom(ctx context.Context, orgID int64, slug string) error {
	return r.db.WithContext(ctx).Where("organization_id = ? AND slug = ?", orgID, slug).Delete(&agent.CustomAgent{}).Error
}

func (r *agentRepo) CountLoopReferences(ctx context.Context, orgID int64, slug string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Raw("SELECT COUNT(*) FROM loops WHERE agent_slug = ?", slug).Scan(&count).Error
	return count, err
}
