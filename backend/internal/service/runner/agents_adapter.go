package runner

import (
	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
)

// AgentServiceAdapter adapts agent.AgentService to interfaces.AgentsProvider interface
type AgentServiceAdapter struct {
	agentSvc *agent.AgentService
}

// NewAgentServiceAdapter creates a new adapter
func NewAgentServiceAdapter(agentSvc *agent.AgentService) *AgentServiceAdapter {
	return &AgentServiceAdapter{agentSvc: agentSvc}
}

// GetAgentsForRunner implements interfaces.AgentsProvider interface
func (a *AgentServiceAdapter) GetAgentsForRunner() []interfaces.AgentInfo {
	// Get agents from service
	agents := a.agentSvc.GetAgentsForRunner()

	// Convert to interfaces.AgentInfo
	result := make([]interfaces.AgentInfo, len(agents))
	for i, ag := range agents {
		result[i] = interfaces.AgentInfo{
			Slug:          ag.Slug,
			Name:          ag.Name,
			Executable:    ag.Executable,
			LaunchCommand: ag.LaunchCommand,
		}
	}
	return result
}
