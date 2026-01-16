package v1

import (
	"net/http"

	agentDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/gin-gonic/gin"
)

// MCPDiscoveryHandler handles MCP tool discovery requests.
// Uses PodAuthMiddleware (X-Pod-Key authentication).
type MCPDiscoveryHandler struct {
	runnerService *runner.Service
	agentTypeSvc  *agent.AgentTypeService
	userConfigSvc *agent.UserConfigService
}

// NewMCPDiscoveryHandler creates a new MCP discovery handler.
func NewMCPDiscoveryHandler(
	runnerSvc *runner.Service,
	agentTypeSvc *agent.AgentTypeService,
	userConfigSvc *agent.UserConfigService,
) *MCPDiscoveryHandler {
	return &MCPDiscoveryHandler{
		runnerService: runnerSvc,
		agentTypeSvc:  agentTypeSvc,
		userConfigSvc: userConfigSvc,
	}
}

// ConfigFieldSummary is a simplified config field for LLM consumption.
// Removes validation and show_when fields that are only used by frontend.
type ConfigFieldSummary struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Default  interface{} `json:"default,omitempty"`
	Options  []string    `json:"options,omitempty"`
	Required bool        `json:"required,omitempty"`
}

// AgentTypeSummary is a simplified AgentType for LLM consumption.
type AgentTypeSummary struct {
	ID          int64                  `json:"id"`
	Slug        string                 `json:"slug"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Config      []ConfigFieldSummary   `json:"config,omitempty"`
	UserConfig  map[string]interface{} `json:"user_config,omitempty"`
}

// RunnerSummary is a simplified Runner with nested Agent details.
type RunnerSummary struct {
	ID                int64              `json:"id"`
	NodeID            string             `json:"node_id"`
	Description       string             `json:"description,omitempty"`
	Status            string             `json:"status"`
	CurrentPods       int                `json:"current_pods"`
	MaxConcurrentPods int                `json:"max_concurrent_pods"`
	AvailableAgents   []AgentTypeSummary `json:"available_agents"`
}

// ListRunnersForMCP returns simplified Runner list with nested Agent info and user config.
// GET /api/v1/orgs/:slug/pod/runners
func (h *MCPDiscoveryHandler) ListRunnersForMCP(c *gin.Context) {
	tenant := middleware.GetTenant(c)
	userID := middleware.GetUserID(c)

	// 1. Get all runners
	runners, err := h.runnerService.ListRunners(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list runners"})
		return
	}

	// 2. Get all agent types (builtin + custom)
	builtinTypes, _ := h.agentTypeSvc.ListBuiltinAgentTypes(c.Request.Context())
	customTypes, _ := h.agentTypeSvc.ListCustomAgentTypes(c.Request.Context(), tenant.OrganizationID)

	// 3. Build slug -> AgentType map (only builtin types have ConfigSchema)
	agentMap := make(map[string]*agentDomain.AgentType)
	for _, at := range builtinTypes {
		agentMap[at.Slug] = at
	}

	// Build slug -> CustomAgentType map (no ConfigSchema)
	customAgentMap := make(map[string]*agentDomain.CustomAgentType)
	for _, cat := range customTypes {
		customAgentMap[cat.Slug] = cat
	}

	// 4. Get user's agent configs
	userConfigs, _ := h.userConfigSvc.ListUserAgentConfigs(c.Request.Context(), userID)
	userConfigMap := make(map[int64]agentDomain.ConfigValues)
	for _, cfg := range userConfigs {
		userConfigMap[cfg.AgentTypeID] = cfg.ConfigValues
	}

	// 5. Convert to simplified format
	result := make([]RunnerSummary, 0, len(runners))
	for _, r := range runners {
		summary := RunnerSummary{
			ID:                r.ID,
			NodeID:            r.NodeID,
			Description:       r.Description,
			Status:            string(r.Status),
			CurrentPods:       r.CurrentPods,
			MaxConcurrentPods: r.MaxConcurrentPods,
			AvailableAgents:   make([]AgentTypeSummary, 0),
		}

		// Fill agent info from available_agents slugs
		for _, slug := range r.AvailableAgents {
			// Check builtin types first
			if at, ok := agentMap[slug]; ok {
				desc := ""
				if at.Description != nil {
					desc = *at.Description
				}

				// Simplify ConfigSchema
				configFields := make([]ConfigFieldSummary, 0, len(at.ConfigSchema.Fields))
				for _, f := range at.ConfigSchema.Fields {
					field := ConfigFieldSummary{
						Name:     f.Name,
						Type:     f.Type,
						Default:  f.Default,
						Required: f.Required,
					}
					// Simplify options to string array
					for _, opt := range f.Options {
						field.Options = append(field.Options, opt.Value)
					}
					configFields = append(configFields, field)
				}

				// Get user config
				userCfg := userConfigMap[at.ID]
				if userCfg == nil {
					userCfg = make(map[string]interface{})
				}

				summary.AvailableAgents = append(summary.AvailableAgents, AgentTypeSummary{
					ID:          at.ID,
					Slug:        at.Slug,
					Name:        at.Name,
					Description: desc,
					Config:      configFields,
					UserConfig:  userCfg,
				})
				continue
			}

			// Check custom types (no ConfigSchema)
			if cat, ok := customAgentMap[slug]; ok {
				desc := ""
				if cat.Description != nil {
					desc = *cat.Description
				}

				summary.AvailableAgents = append(summary.AvailableAgents, AgentTypeSummary{
					ID:          cat.ID,
					Slug:        cat.Slug,
					Name:        cat.Name,
					Description: desc,
					Config:      nil, // Custom types don't have ConfigSchema
					UserConfig:  nil,
				})
			}
		}

		result = append(result, summary)
	}

	c.JSON(http.StatusOK, gin.H{"runners": result})
}
