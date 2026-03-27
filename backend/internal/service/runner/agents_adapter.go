package runner

import (
	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
)

// AgentTypeServiceAdapter adapts agent.AgentTypeService to interfaces.AgentTypesProvider interface
type AgentTypeServiceAdapter struct {
	agentTypeSvc *agent.AgentTypeService
}

// NewAgentTypeServiceAdapter creates a new adapter
func NewAgentTypeServiceAdapter(agentTypeSvc *agent.AgentTypeService) *AgentTypeServiceAdapter {
	return &AgentTypeServiceAdapter{agentTypeSvc: agentTypeSvc}
}

// GetAgentTypesForRunner implements interfaces.AgentTypesProvider interface
func (a *AgentTypeServiceAdapter) GetAgentTypesForRunner() []interfaces.AgentTypeInfo {
	// Get agent types from service
	types := a.agentTypeSvc.GetAgentTypesForRunner()

	// Convert to interfaces.AgentTypeInfo
	result := make([]interfaces.AgentTypeInfo, len(types))
	for i, t := range types {
		result[i] = interfaces.AgentTypeInfo{
			Slug:          t.Slug,
			Name:          t.Name,
			Executable:    t.Executable,
			LaunchCommand: t.LaunchCommand,
		}
	}
	return result
}
