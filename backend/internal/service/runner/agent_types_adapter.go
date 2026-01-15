package runner

import (
	"github.com/anthropics/agentmesh/backend/internal/service/agent"
)

// AgentTypeServiceAdapter adapts agent.AgentTypeService to AgentTypesProvider interface
type AgentTypeServiceAdapter struct {
	agentTypeSvc *agent.AgentTypeService
}

// NewAgentTypeServiceAdapter creates a new adapter
func NewAgentTypeServiceAdapter(agentTypeSvc *agent.AgentTypeService) *AgentTypeServiceAdapter {
	return &AgentTypeServiceAdapter{agentTypeSvc: agentTypeSvc}
}

// GetAgentTypesForRunner implements AgentTypesProvider interface
func (a *AgentTypeServiceAdapter) GetAgentTypesForRunner() []AgentTypeInfo {
	// Get agent types from service
	types := a.agentTypeSvc.GetAgentTypesForRunner()

	// Convert to runner.AgentTypeInfo
	result := make([]AgentTypeInfo, len(types))
	for i, t := range types {
		result[i] = AgentTypeInfo{
			Slug:          t.Slug,
			Name:          t.Name,
			Executable:    t.Executable,
			LaunchCommand: t.LaunchCommand,
		}
	}
	return result
}
