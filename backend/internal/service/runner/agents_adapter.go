package runner

import (
	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
)

type AgentServiceAdapter struct {
	agentSvc *agent.AgentService
}

func NewAgentServiceAdapter(agentSvc *agent.AgentService) *AgentServiceAdapter {
	return &AgentServiceAdapter{agentSvc: agentSvc}
}

func (a *AgentServiceAdapter) GetAgentsForRunner() []interfaces.AgentInfo {
	agents := a.agentSvc.GetAgentsForRunner()

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
