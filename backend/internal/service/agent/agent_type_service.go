package agent

import (
	"context"
	"errors"

	"github.com/anthropics/agentmesh/backend/internal/domain/agent"
	"gorm.io/gorm"
)

// Errors for AgentTypeService
var (
	ErrAgentTypeNotFound = errors.New("agent type not found")
	ErrAgentSlugExists   = errors.New("agent type slug already exists")
)

// AgentTypeInfo is a simplified agent type for Runner initialization
type AgentTypeInfo struct {
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	Executable    string `json:"executable"`
	LaunchCommand string `json:"launch_command"`
}

// CreateCustomAgentRequest represents a custom agent creation request
type CreateCustomAgentRequest struct {
	Slug             string
	Name             string
	Description      *string
	LaunchCommand    string
	DefaultArgs      *string
	CredentialSchema agent.CredentialSchema
	StatusDetection  agent.StatusDetection
}

// AgentTypeService handles agent type operations
type AgentTypeService struct {
	db *gorm.DB
}

// NewAgentTypeService creates a new agent type service
func NewAgentTypeService(db *gorm.DB) *AgentTypeService {
	return &AgentTypeService{db: db}
}

// ListBuiltinAgentTypes returns all builtin agent types
func (s *AgentTypeService) ListBuiltinAgentTypes(ctx context.Context) ([]*agent.AgentType, error) {
	var types []*agent.AgentType
	err := s.db.WithContext(ctx).Where("is_builtin = ? AND is_active = ?", true, true).Find(&types).Error
	return types, err
}

// GetAgentTypesForRunner returns agent types for Runner initialization handshake
// This implements the runner.AgentTypesProvider interface
func (s *AgentTypeService) GetAgentTypesForRunner() []AgentTypeInfo {
	var types []*agent.AgentType
	if err := s.db.Where("is_active = ?", true).Find(&types).Error; err != nil {
		return nil
	}

	result := make([]AgentTypeInfo, 0, len(types))
	for _, t := range types {
		result = append(result, AgentTypeInfo{
			Slug:          t.Slug,
			Name:          t.Name,
			Executable:    t.Executable,
			LaunchCommand: t.LaunchCommand,
		})
	}
	return result
}

// GetAgentType returns an agent type by ID
func (s *AgentTypeService) GetAgentType(ctx context.Context, id int64) (*agent.AgentType, error) {
	var agentType agent.AgentType
	if err := s.db.WithContext(ctx).First(&agentType, id).Error; err != nil {
		return nil, ErrAgentTypeNotFound
	}
	return &agentType, nil
}

// GetAgentTypeBySlug returns an agent type by slug
func (s *AgentTypeService) GetAgentTypeBySlug(ctx context.Context, slug string) (*agent.AgentType, error) {
	var agentType agent.AgentType
	if err := s.db.WithContext(ctx).Where("slug = ?", slug).First(&agentType).Error; err != nil {
		return nil, ErrAgentTypeNotFound
	}
	return &agentType, nil
}

// CreateCustomAgentType creates a custom agent type for an organization
func (s *AgentTypeService) CreateCustomAgentType(ctx context.Context, orgID int64, req *CreateCustomAgentRequest) (*agent.CustomAgentType, error) {
	// Check if slug already exists
	var existing agent.CustomAgentType
	if err := s.db.WithContext(ctx).Where("organization_id = ? AND slug = ?", orgID, req.Slug).First(&existing).Error; err == nil {
		return nil, ErrAgentSlugExists
	}

	customAgent := &agent.CustomAgentType{
		OrganizationID:   orgID,
		Slug:             req.Slug,
		Name:             req.Name,
		Description:      req.Description,
		LaunchCommand:    req.LaunchCommand,
		DefaultArgs:      req.DefaultArgs,
		CredentialSchema: req.CredentialSchema,
		StatusDetection:  req.StatusDetection,
		IsActive:         true,
	}

	if err := s.db.WithContext(ctx).Create(customAgent).Error; err != nil {
		return nil, err
	}

	return customAgent, nil
}

// UpdateCustomAgentType updates a custom agent type
func (s *AgentTypeService) UpdateCustomAgentType(ctx context.Context, id int64, updates map[string]interface{}) (*agent.CustomAgentType, error) {
	if err := s.db.WithContext(ctx).Model(&agent.CustomAgentType{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}

	var customAgent agent.CustomAgentType
	if err := s.db.WithContext(ctx).First(&customAgent, id).Error; err != nil {
		return nil, err
	}
	return &customAgent, nil
}

// DeleteCustomAgentType deletes a custom agent type
func (s *AgentTypeService) DeleteCustomAgentType(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&agent.CustomAgentType{}, id).Error
}

// ListCustomAgentTypes returns custom agent types for an organization
func (s *AgentTypeService) ListCustomAgentTypes(ctx context.Context, orgID int64) ([]*agent.CustomAgentType, error) {
	var types []*agent.CustomAgentType
	err := s.db.WithContext(ctx).Where("organization_id = ? AND is_active = ?", orgID, true).Find(&types).Error
	return types, err
}

// GetCustomAgentType returns a custom agent type by ID
func (s *AgentTypeService) GetCustomAgentType(ctx context.Context, id int64) (*agent.CustomAgentType, error) {
	var customAgent agent.CustomAgentType
	if err := s.db.WithContext(ctx).First(&customAgent, id).Error; err != nil {
		return nil, ErrAgentTypeNotFound
	}
	return &customAgent, nil
}
