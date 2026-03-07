package infra

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"gorm.io/gorm"
)

// Compile-time interface check
var _ agent.AgentTypeRepository = (*agentTypeRepo)(nil)

type agentTypeRepo struct {
	db *gorm.DB
}

// NewAgentTypeRepository creates a new GORM-based agent type repository
func NewAgentTypeRepository(db *gorm.DB) agent.AgentTypeRepository {
	return &agentTypeRepo{db: db}
}

func (r *agentTypeRepo) ListBuiltinActive(ctx context.Context) ([]*agent.AgentType, error) {
	var types []*agent.AgentType
	err := r.db.WithContext(ctx).Where("is_builtin = ? AND is_active = ?", true, true).Find(&types).Error
	return types, err
}

func (r *agentTypeRepo) ListAllActive(ctx context.Context) ([]*agent.AgentType, error) {
	var types []*agent.AgentType
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Find(&types).Error
	return types, err
}

func (r *agentTypeRepo) GetByID(ctx context.Context, id int64) (*agent.AgentType, error) {
	var agentType agent.AgentType
	if err := r.db.WithContext(ctx).First(&agentType, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &agentType, nil
}

func (r *agentTypeRepo) GetBySlug(ctx context.Context, slug string) (*agent.AgentType, error) {
	var agentType agent.AgentType
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&agentType).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &agentType, nil
}

func (r *agentTypeRepo) ListCustomByOrg(ctx context.Context, orgID int64) ([]*agent.CustomAgentType, error) {
	var types []*agent.CustomAgentType
	err := r.db.WithContext(ctx).Where("organization_id = ? AND is_active = ?", orgID, true).Find(&types).Error
	return types, err
}

func (r *agentTypeRepo) GetCustomByID(ctx context.Context, id int64) (*agent.CustomAgentType, error) {
	var custom agent.CustomAgentType
	if err := r.db.WithContext(ctx).First(&custom, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &custom, nil
}

func (r *agentTypeRepo) CustomSlugExists(ctx context.Context, orgID int64, slug string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&agent.CustomAgentType{}).
		Where("organization_id = ? AND slug = ?", orgID, slug).
		Count(&count).Error
	return count > 0, err
}

func (r *agentTypeRepo) CreateCustom(ctx context.Context, custom *agent.CustomAgentType) error {
	return r.db.WithContext(ctx).Create(custom).Error
}

func (r *agentTypeRepo) UpdateCustom(ctx context.Context, id int64, updates map[string]interface{}) (*agent.CustomAgentType, error) {
	if err := r.db.WithContext(ctx).Model(&agent.CustomAgentType{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	var custom agent.CustomAgentType
	if err := r.db.WithContext(ctx).First(&custom, id).Error; err != nil {
		return nil, err
	}
	return &custom, nil
}

func (r *agentTypeRepo) DeleteCustom(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&agent.CustomAgentType{}, id).Error
}

func (r *agentTypeRepo) CountLoopReferences(ctx context.Context, customID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Raw("SELECT COUNT(*) FROM loops WHERE custom_agent_type_id = ?", customID).Scan(&count).Error
	return count, err
}
