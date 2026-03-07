package agent

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

// Errors for AgentTypeService
var (
	ErrAgentTypeNotFound    = errors.New("agent type not found")
	ErrAgentSlugExists      = errors.New("agent type slug already exists")
	ErrAgentTypeHasLoopRefs = errors.New("cannot delete: agent type is referenced by one or more loops")
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
	repo agent.AgentTypeRepository
}

// NewAgentTypeService creates a new agent type service
func NewAgentTypeService(repo agent.AgentTypeRepository) *AgentTypeService {
	return &AgentTypeService{repo: repo}
}

// ListBuiltinAgentTypes returns all builtin agent types
func (s *AgentTypeService) ListBuiltinAgentTypes(ctx context.Context) ([]*agent.AgentType, error) {
	return s.repo.ListBuiltinActive(ctx)
}

// GetAgentTypesForRunner returns agent types for Runner initialization handshake
// This implements the runner.AgentTypesProvider interface
func (s *AgentTypeService) GetAgentTypesForRunner() []AgentTypeInfo {
	types, err := s.repo.ListAllActive(context.Background())
	if err != nil {
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
	at, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if at == nil {
		return nil, ErrAgentTypeNotFound
	}
	return at, nil
}

// GetAgentTypeBySlug returns an agent type by slug
func (s *AgentTypeService) GetAgentTypeBySlug(ctx context.Context, slug string) (*agent.AgentType, error) {
	at, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if at == nil {
		return nil, ErrAgentTypeNotFound
	}
	return at, nil
}

// CreateCustomAgentType creates a custom agent type for an organization
func (s *AgentTypeService) CreateCustomAgentType(ctx context.Context, orgID int64, req *CreateCustomAgentRequest) (*agent.CustomAgentType, error) {
	exists, err := s.repo.CustomSlugExists(ctx, orgID, req.Slug)
	if err != nil {
		return nil, err
	}
	if exists {
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

	if err := s.repo.CreateCustom(ctx, customAgent); err != nil {
		return nil, err
	}

	return customAgent, nil
}

// UpdateCustomAgentType updates a custom agent type
func (s *AgentTypeService) UpdateCustomAgentType(ctx context.Context, id int64, updates map[string]interface{}) (*agent.CustomAgentType, error) {
	return s.repo.UpdateCustom(ctx, id, updates)
}

// DeleteCustomAgentType deletes a custom agent type.
// Blocks deletion if any loops reference this agent type (application-level RESTRICT).
func (s *AgentTypeService) DeleteCustomAgentType(ctx context.Context, id int64) error {
	loopCount, err := s.repo.CountLoopReferences(ctx, id)
	if err != nil {
		return err
	}
	if loopCount > 0 {
		return ErrAgentTypeHasLoopRefs
	}
	return s.repo.DeleteCustom(ctx, id)
}

// ListCustomAgentTypes returns custom agent types for an organization
func (s *AgentTypeService) ListCustomAgentTypes(ctx context.Context, orgID int64) ([]*agent.CustomAgentType, error) {
	return s.repo.ListCustomByOrg(ctx, orgID)
}

// GetCustomAgentType returns a custom agent type by ID
func (s *AgentTypeService) GetCustomAgentType(ctx context.Context, id int64) (*agent.CustomAgentType, error) {
	custom, err := s.repo.GetCustomByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if custom == nil {
		return nil, ErrAgentTypeNotFound
	}
	return custom, nil
}
