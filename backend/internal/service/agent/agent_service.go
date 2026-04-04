package agent

import (
	"context"
	"errors"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

// Errors for AgentService
var (
	ErrAgentNotFound    = errors.New("agent not found")
	ErrAgentSlugExists  = errors.New("agent slug already exists")
	ErrAgentHasLoopRefs = errors.New("cannot delete: agent is referenced by one or more loops")
)

// AgentInfo is a simplified agent descriptor for Runner initialization
type AgentInfo struct {
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	Executable    string `json:"executable"`
	LaunchCommand string `json:"launch_command"`
}

// CreateCustomAgentRequest represents a custom agent creation request
type CreateCustomAgentRequest struct {
	Slug          string
	Name          string
	Description   *string
	LaunchCommand string
	DefaultArgs   *string
	AgentfileSource *string
}

// AgentService handles agent operations
type AgentService struct {
	repo agent.AgentRepository
}

// NewAgentService creates a new agent service
func NewAgentService(repo agent.AgentRepository) *AgentService {
	return &AgentService{repo: repo}
}

// ListBuiltinAgents returns all builtin agents
func (s *AgentService) ListBuiltinAgents(ctx context.Context) ([]*agent.Agent, error) {
	return s.repo.ListBuiltinActive(ctx)
}

// GetAgentsForRunner returns agents for Runner initialization handshake
func (s *AgentService) GetAgentsForRunner() []AgentInfo {
	types, err := s.repo.ListAllActive(context.Background())
	if err != nil {
		return nil
	}

	result := make([]AgentInfo, 0, len(types))
	for _, t := range types {
		result = append(result, AgentInfo{
			Slug:          t.Slug,
			Name:          t.Name,
			Executable:    t.Executable,
			LaunchCommand: t.LaunchCommand,
		})
	}
	return result
}

// GetBySlug returns an agent by slug, or ErrAgentNotFound if not found.
func (s *AgentService) GetBySlug(ctx context.Context, slug string) (*agent.Agent, error) {
	at, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if at == nil {
		return nil, ErrAgentNotFound
	}
	return at, nil
}

// GetAgent is an alias for GetBySlug.
func (s *AgentService) GetAgent(ctx context.Context, slug string) (*agent.Agent, error) {
	return s.GetBySlug(ctx, slug)
}

// CreateCustomAgent creates a custom agent for an organization
func (s *AgentService) CreateCustomAgent(ctx context.Context, orgID int64, req *CreateCustomAgentRequest) (*agent.CustomAgent, error) {
	exists, err := s.repo.CustomSlugExists(ctx, orgID, req.Slug)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrAgentSlugExists
	}

	customAgent := &agent.CustomAgent{
		OrganizationID: orgID,
		Slug:           req.Slug,
		Name:           req.Name,
		Description:    req.Description,
		LaunchCommand:  req.LaunchCommand,
		DefaultArgs:    req.DefaultArgs,
		AgentfileSource:  req.AgentfileSource,
		IsActive:       true,
	}

	if err := s.repo.CreateCustom(ctx, customAgent); err != nil {
		slog.Error("failed to create custom agent", "org_id", orgID, "slug", req.Slug, "error", err)
		return nil, err
	}

	slog.Info("custom agent created", "org_id", orgID, "slug", req.Slug)
	return customAgent, nil
}

// UpdateCustomAgent updates a custom agent
func (s *AgentService) UpdateCustomAgent(ctx context.Context, orgID int64, slug string, updates map[string]interface{}) (*agent.CustomAgent, error) {
	result, err := s.repo.UpdateCustom(ctx, orgID, slug, updates)
	if err != nil {
		slog.Error("failed to update custom agent", "org_id", orgID, "slug", slug, "error", err)
		return nil, err
	}
	slog.Info("custom agent updated", "org_id", orgID, "slug", slug)
	return result, nil
}

// DeleteCustomAgent deletes a custom agent.
// Blocks deletion if any loops reference this agent (application-level RESTRICT).
func (s *AgentService) DeleteCustomAgent(ctx context.Context, orgID int64, slug string) error {
	loopCount, err := s.repo.CountLoopReferences(ctx, orgID, slug)
	if err != nil {
		return err
	}
	if loopCount > 0 {
		return ErrAgentHasLoopRefs
	}
	if err := s.repo.DeleteCustom(ctx, orgID, slug); err != nil {
		slog.Error("failed to delete custom agent", "org_id", orgID, "slug", slug, "error", err)
		return err
	}
	slog.Info("custom agent deleted", "org_id", orgID, "slug", slug)
	return nil
}

// ListCustomAgents returns custom agents for an organization
func (s *AgentService) ListCustomAgents(ctx context.Context, orgID int64) ([]*agent.CustomAgent, error) {
	return s.repo.ListCustomByOrg(ctx, orgID)
}

// GetCustomAgent returns a custom agent by slug
func (s *AgentService) GetCustomAgent(ctx context.Context, orgID int64, slug string) (*agent.CustomAgent, error) {
	custom, err := s.repo.GetCustomBySlug(ctx, orgID, slug)
	if err != nil {
		return nil, err
	}
	if custom == nil {
		return nil, ErrAgentNotFound
	}
	return custom, nil
}
