package grpc

import (
	"context"

	agentDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
)

// ==================== Discovery MCP Methods ====================

// mcpListAvailablePods handles the "list_available_pods" MCP method.
func (a *GRPCRunnerAdapter) mcpListAvailablePods(ctx context.Context, tc *middleware.TenantContext) (interface{}, *mcpError) {
	pods, _, err := a.mcpPodService.ListPods(ctx, tc.OrganizationID, "active", 100, 0)
	if err != nil {
		return nil, newMcpError(500, "failed to list pods")
	}

	// Convert to simplified format for MCP
	type podSummary struct {
		PodKey string `json:"pod_key"`
		Status string `json:"status"`
	}

	result := make([]podSummary, 0, len(pods))
	for _, p := range pods {
		result = append(result, podSummary{
			PodKey: p.PodKey,
			Status: p.Status,
		})
	}

	return map[string]interface{}{"pods": result}, nil
}

// mcpListRunners handles the "list_runners" MCP method.
func (a *GRPCRunnerAdapter) mcpListRunners(ctx context.Context, tc *middleware.TenantContext) (interface{}, *mcpError) {
	runners, err := a.runnerMcpService.ListRunners(ctx, tc.OrganizationID)
	if err != nil {
		return nil, newMcpError(500, "failed to list runners")
	}

	// Get agent types for enrichment
	builtinTypes, _ := a.agentTypeSvc.ListBuiltinAgentTypes(ctx)
	customTypes, _ := a.agentTypeSvc.ListCustomAgentTypes(ctx, tc.OrganizationID)

	// Build slug -> AgentType map
	agentMap := make(map[string]*agentDomain.AgentType)
	for _, at := range builtinTypes {
		agentMap[at.Slug] = at
	}

	customAgentMap := make(map[string]*agentDomain.CustomAgentType)
	for _, cat := range customTypes {
		customAgentMap[cat.Slug] = cat
	}

	// Get user's agent configs
	userConfigs, _ := a.userConfigSvc.ListUserAgentConfigs(ctx, tc.UserID)
	userConfigMap := make(map[int64]agentDomain.ConfigValues)
	for _, cfg := range userConfigs {
		userConfigMap[cfg.AgentTypeID] = cfg.ConfigValues
	}

	// Build result
	type configFieldSummary struct {
		Name     string      `json:"name"`
		Type     string      `json:"type"`
		Default  interface{} `json:"default,omitempty"`
		Options  []string    `json:"options,omitempty"`
		Required bool        `json:"required,omitempty"`
	}

	type agentTypeSummary struct {
		ID          int64                  `json:"id"`
		Slug        string                 `json:"slug"`
		Name        string                 `json:"name"`
		Description string                 `json:"description,omitempty"`
		Config      []configFieldSummary   `json:"config,omitempty"`
		UserConfig  map[string]interface{} `json:"user_config,omitempty"`
	}

	type runnerSummary struct {
		ID                int64              `json:"id"`
		NodeID            string             `json:"node_id"`
		Description       string             `json:"description,omitempty"`
		Status            string             `json:"status"`
		CurrentPods       int                `json:"current_pods"`
		MaxConcurrentPods int                `json:"max_concurrent_pods"`
		AvailableAgents   []agentTypeSummary `json:"available_agents"`
	}

	result := make([]runnerSummary, 0, len(runners))
	for _, r := range runners {
		summary := runnerSummary{
			ID:                r.ID,
			NodeID:            r.NodeID,
			Description:       r.Description,
			Status:            r.Status,
			CurrentPods:       r.CurrentPods,
			MaxConcurrentPods: r.MaxConcurrentPods,
			AvailableAgents:   make([]agentTypeSummary, 0),
		}

		for _, slug := range r.AvailableAgents {
			if at, ok := agentMap[slug]; ok {
				desc := ""
				if at.Description != nil {
					desc = *at.Description
				}

				configFields := make([]configFieldSummary, 0, len(at.ConfigSchema.Fields))
				for _, f := range at.ConfigSchema.Fields {
					field := configFieldSummary{
						Name:     f.Name,
						Type:     f.Type,
						Default:  f.Default,
						Required: f.Required,
					}
					for _, opt := range f.Options {
						field.Options = append(field.Options, opt.Value)
					}
					configFields = append(configFields, field)
				}

				userCfg := userConfigMap[at.ID]
				if userCfg == nil {
					userCfg = make(map[string]interface{})
				}

				summary.AvailableAgents = append(summary.AvailableAgents, agentTypeSummary{
					ID:          at.ID,
					Slug:        at.Slug,
					Name:        at.Name,
					Description: desc,
					Config:      configFields,
					UserConfig:  userCfg,
				})
				continue
			}

			if cat, ok := customAgentMap[slug]; ok {
				desc := ""
				if cat.Description != nil {
					desc = *cat.Description
				}
				summary.AvailableAgents = append(summary.AvailableAgents, agentTypeSummary{
					ID:          cat.ID,
					Slug:        cat.Slug,
					Name:        cat.Name,
					Description: desc,
				})
			}
		}

		result = append(result, summary)
	}

	return map[string]interface{}{"runners": result}, nil
}

// mcpListRepositories handles the "list_repositories" MCP method.
func (a *GRPCRunnerAdapter) mcpListRepositories(ctx context.Context, tc *middleware.TenantContext) (interface{}, *mcpError) {
	repos, err := a.repositoryService.ListByOrganization(ctx, tc.OrganizationID)
	if err != nil {
		return nil, newMcpError(500, "failed to list repositories")
	}

	return map[string]interface{}{"repositories": repos}, nil
}
