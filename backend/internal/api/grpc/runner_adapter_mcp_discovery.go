package grpc

import (
	"context"

	agentDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/agentfile/extract"
	"github.com/anthropics/agentsmesh/agentfile/parser"
)

// ==================== Discovery MCP Methods ====================

// mcpListAvailablePods handles the "list_available_pods" MCP method.
func (a *GRPCRunnerAdapter) mcpListAvailablePods(ctx context.Context, tc *middleware.TenantContext) (interface{}, *mcpError) {
	pods, _, err := a.mcpPodService.ListPods(ctx, tc.OrganizationID, agentpod.ActiveStatuses(), 0, 100, 0)
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
	runners, err := a.runnerMcpService.ListRunners(ctx, tc.OrganizationID, tc.UserID)
	if err != nil {
		return nil, newMcpError(500, "failed to list runners")
	}

	// Get agents for enrichment
	builtinTypes, _ := a.agentSvc.ListBuiltinAgents(ctx)
	customTypes, _ := a.agentSvc.ListCustomAgents(ctx, tc.OrganizationID)

	// Build slug -> Agent map
	agentMap := make(map[string]*agentDomain.Agent)
	for _, at := range builtinTypes {
		agentMap[at.Slug] = at
	}

	customAgentMap := make(map[string]*agentDomain.CustomAgent)
	for _, cat := range customTypes {
		customAgentMap[cat.Slug] = cat
	}

	// Get user's agent configs
	userConfigs, _ := a.userConfigSvc.ListUserAgentConfigs(ctx, tc.UserID)
	userConfigMap := make(map[string]agentDomain.ConfigValues)
	for _, cfg := range userConfigs {
		userConfigMap[cfg.AgentSlug] = cfg.ConfigValues
	}

	// Build result
	type configFieldSummary struct {
		Name     string      `json:"name"`
		Type     string      `json:"type"`
		Default  interface{} `json:"default,omitempty"`
		Options  []string    `json:"options,omitempty"`
		Required bool        `json:"required,omitempty"`
	}

	type agentSummary struct {
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
		AvailableAgents   []agentSummary `json:"available_agents"`
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
			AvailableAgents:   make([]agentSummary, 0),
		}

		for _, slug := range r.AvailableAgents {
			if at, ok := agentMap[slug]; ok {
				desc := ""
				if at.Description != nil {
					desc = *at.Description
				}

				configFields := make([]configFieldSummary, 0)
				if at.AgentfileSource != nil && *at.AgentfileSource != "" {
					prog, errs := parser.Parse(*at.AgentfileSource)
					if len(errs) == 0 && prog != nil {
						spec := extract.Extract(prog)
						for _, cfg := range spec.Config {
							field := configFieldSummary{
								Name:    cfg.Name,
								Type:    cfg.Type,
								Default: cfg.Default,
							}
							field.Options = cfg.Options
							configFields = append(configFields, field)
						}
					}
				}

				userCfg := userConfigMap[at.Slug]
				if userCfg == nil {
					userCfg = make(map[string]interface{})
				}

				summary.AvailableAgents = append(summary.AvailableAgents, agentSummary{
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
				summary.AvailableAgents = append(summary.AvailableAgents, agentSummary{
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
