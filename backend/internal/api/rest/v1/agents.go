package v1

import (
	"context"
	"net/http"
	"strconv"

	agentDomain "github.com/anthropics/agentmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentmesh/backend/internal/middleware"
	"github.com/anthropics/agentmesh/backend/internal/service/agent"
	"github.com/gin-gonic/gin"
)

// AgentHandler handles agent-related requests
type AgentHandler struct {
	agentTypeSvc      *agent.AgentTypeService
	credentialSvc     *agent.CredentialProfileService
	userConfigSvc     *agent.UserConfigService
	configBuilder     *agent.ConfigBuilder
}

// NewAgentHandler creates a new agent handler
func NewAgentHandler(
	agentTypeSvc *agent.AgentTypeService,
	credentialSvc *agent.CredentialProfileService,
	userConfigSvc *agent.UserConfigService,
) *AgentHandler {
	return &AgentHandler{
		agentTypeSvc:  agentTypeSvc,
		credentialSvc: credentialSvc,
		userConfigSvc: userConfigSvc,
		configBuilder: agent.NewConfigBuilder(&compositeProvider{
			agentTypeSvc:  agentTypeSvc,
			credentialSvc: credentialSvc,
			userConfigSvc: userConfigSvc,
		}),
	}
}

// compositeProvider implements AgentConfigProvider by combining sub-services
type compositeProvider struct {
	agentTypeSvc  *agent.AgentTypeService
	credentialSvc *agent.CredentialProfileService
	userConfigSvc *agent.UserConfigService
}

func (p *compositeProvider) GetAgentType(ctx context.Context, id int64) (*agentDomain.AgentType, error) {
	return p.agentTypeSvc.GetAgentType(ctx, id)
}

func (p *compositeProvider) GetUserEffectiveConfig(ctx context.Context, userID, agentTypeID int64, overrides agentDomain.ConfigValues) agentDomain.ConfigValues {
	return p.userConfigSvc.GetUserEffectiveConfig(ctx, userID, agentTypeID, overrides)
}

func (p *compositeProvider) GetEffectiveCredentialsForPod(ctx context.Context, userID, agentTypeID int64, profileID *int64) (agentDomain.EncryptedCredentials, bool, error) {
	return p.credentialSvc.GetEffectiveCredentialsForPod(ctx, userID, agentTypeID, profileID)
}

// ListAgentTypes lists available agent types
// GET /api/v1/organizations/:slug/agents/types
func (h *AgentHandler) ListAgentTypes(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	// Get builtin types
	builtinTypes, err := h.agentTypeSvc.ListBuiltinAgentTypes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list builtin agent types"})
		return
	}

	// Get custom types for organization
	customTypes, err := h.agentTypeSvc.ListCustomAgentTypes(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list custom agent types"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"builtin_types": builtinTypes,
		"custom_types":  customTypes,
	})
}

// GetUserCredentials returns user's agent credentials
// GET /api/v1/users/me/agents/credentials
func (h *AgentHandler) GetUserCredentials(c *gin.Context) {
	userID := middleware.GetUserID(c)

	// Get builtin types to list credentials for
	builtinTypes, err := h.agentTypeSvc.ListBuiltinAgentTypes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list agent types"})
		return
	}

	credentials := make(map[string]bool)
	for _, t := range builtinTypes {
		creds, err := h.credentialSvc.GetUserCredentials(c.Request.Context(), userID, t.ID)
		credentials[t.Slug] = err == nil && creds != nil
	}

	c.JSON(http.StatusOK, gin.H{"credentials": credentials})
}

// SetUserCredentialsRequest represents user credentials request
type SetUserCredentialsRequest struct {
	Credentials map[string]interface{} `json:"credentials" binding:"required"`
}

// SetUserCredentials sets user-level credentials
// PUT /api/v1/users/me/agents/credentials/:agent_type_id
func (h *AgentHandler) SetUserCredentials(c *gin.Context) {
	agentTypeID, err := strconv.ParseInt(c.Param("agent_type_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent type ID"})
		return
	}

	var req SetUserCredentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserID(c)

	// Convert map[string]interface{} to EncryptedCredentials
	creds := make(agentDomain.EncryptedCredentials)
	for k, v := range req.Credentials {
		if s, ok := v.(string); ok {
			creds[k] = s
		}
	}

	if err := h.credentialSvc.SetUserCredentials(c.Request.Context(), userID, agentTypeID, creds); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set credentials"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Credentials updated"})
}

// DeleteUserCredentials deletes user-level credentials
// DELETE /api/v1/users/me/agents/credentials/:agent_type_id
func (h *AgentHandler) DeleteUserCredentials(c *gin.Context) {
	agentTypeID, err := strconv.ParseInt(c.Param("agent_type_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent type ID"})
		return
	}

	userID := middleware.GetUserID(c)

	if err := h.credentialSvc.DeleteUserCredentials(c.Request.Context(), userID, agentTypeID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete credentials"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Credentials deleted"})
}

// CreateCustomAgentRequest represents custom agent creation request
type CreateCustomAgentRequest struct {
	Slug             string                 `json:"slug" binding:"required,min=2,max=50,alphanum"`
	Name             string                 `json:"name" binding:"required,min=2,max=100"`
	Description      string                 `json:"description"`
	LaunchCommand    string                 `json:"launch_command" binding:"required"`
	DefaultArgs      string                 `json:"default_args"`
	CredentialSchema map[string]interface{} `json:"credential_schema"`
	StatusDetection  map[string]interface{} `json:"status_detection"`
}

// CreateCustomAgent creates a custom agent type
// POST /api/v1/organizations/:slug/agents/custom
func (h *AgentHandler) CreateCustomAgent(c *gin.Context) {
	var req CreateCustomAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permission required"})
		return
	}

	// Convert request to service request
	var desc *string
	if req.Description != "" {
		desc = &req.Description
	}
	var args *string
	if req.DefaultArgs != "" {
		args = &req.DefaultArgs
	}

	// Convert credential schema
	var credSchema agentDomain.CredentialSchema
	if req.CredentialSchema != nil {
		// TODO: properly convert credential schema from map to CredentialSchema
	}

	// Convert status detection
	var statusDetection agentDomain.StatusDetection
	if req.StatusDetection != nil {
		statusDetection = make(agentDomain.StatusDetection)
		for k, v := range req.StatusDetection {
			statusDetection[k] = v
		}
	}

	customAgent, err := h.agentTypeSvc.CreateCustomAgentType(c.Request.Context(), tenant.OrganizationID, &agent.CreateCustomAgentRequest{
		Slug:             req.Slug,
		Name:             req.Name,
		Description:      desc,
		LaunchCommand:    req.LaunchCommand,
		DefaultArgs:      args,
		CredentialSchema: credSchema,
		StatusDetection:  statusDetection,
	})
	if err != nil {
		if err == agent.ErrAgentSlugExists {
			c.JSON(http.StatusConflict, gin.H{"error": "Agent slug already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create custom agent"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"custom_agent": customAgent})
}

// UpdateCustomAgent updates a custom agent type
// PUT /api/v1/organizations/:slug/agents/custom/:id
func (h *AgentHandler) UpdateCustomAgent(c *gin.Context) {
	customAgentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid custom agent ID"})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permission required"})
		return
	}

	customAgent, err := h.agentTypeSvc.UpdateCustomAgentType(c.Request.Context(), customAgentID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update custom agent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"custom_agent": customAgent})
}

// DeleteCustomAgent deletes a custom agent type
// DELETE /api/v1/organizations/:slug/agents/custom/:id
func (h *AgentHandler) DeleteCustomAgent(c *gin.Context) {
	customAgentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid custom agent ID"})
		return
	}

	tenant := middleware.GetTenant(c)

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permission required"})
		return
	}

	if err := h.agentTypeSvc.DeleteCustomAgentType(c.Request.Context(), customAgentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete custom agent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Custom agent deleted"})
}

// GetAgentTypeConfigSchema returns the raw config schema for an agent type
// Frontend is responsible for i18n translation using: agent.{slug}.fields.{field.name}.label
// GET /api/v1/organizations/:slug/agents/:agent_type_id/config-schema
func (h *AgentHandler) GetAgentTypeConfigSchema(c *gin.Context) {
	agentTypeID, err := strconv.ParseInt(c.Param("agent_type_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent type ID"})
		return
	}

	schema, err := h.configBuilder.GetConfigSchema(c.Request.Context(), agentTypeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get config schema"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"schema": schema})
}

// GetAgentType returns details of a specific agent type
// GET /api/v1/organizations/:slug/agents/types/:agent_type_id
func (h *AgentHandler) GetAgentType(c *gin.Context) {
	agentTypeID, err := strconv.ParseInt(c.Param("agent_type_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent type ID"})
		return
	}

	agentType, err := h.agentTypeSvc.GetAgentType(c.Request.Context(), agentTypeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent type not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"agent_type": agentType})
}

// ============================================================================
// User Agent Config API (Personal Runtime Configuration)
// ============================================================================

// ListUserAgentConfigs returns all personal configs for the current user
// GET /api/v1/users/me/agent-configs
func (h *AgentHandler) ListUserAgentConfigs(c *gin.Context) {
	userID := middleware.GetUserID(c)

	configs, err := h.userConfigSvc.ListUserAgentConfigs(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list user configs"})
		return
	}

	// Convert to response format
	responses := make([]*agentDomain.UserAgentConfigResponse, len(configs))
	for i, cfg := range configs {
		responses[i] = cfg.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"configs": responses})
}

// GetUserAgentConfig returns the user's personal config for an agent type
// GET /api/v1/users/me/agent-configs/:agent_type_id
func (h *AgentHandler) GetUserAgentConfig(c *gin.Context) {
	agentTypeID, err := strconv.ParseInt(c.Param("agent_type_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent type ID"})
		return
	}

	userID := middleware.GetUserID(c)

	config, err := h.userConfigSvc.GetUserAgentConfig(c.Request.Context(), userID, agentTypeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"config": config.ToResponse()})
}

// SetUserAgentConfigRequest represents a request to set user's personal config
type SetUserAgentConfigRequest struct {
	ConfigValues map[string]interface{} `json:"config_values" binding:"required"`
}

// SetUserAgentConfig sets the user's personal config for an agent type
// PUT /api/v1/users/me/agent-configs/:agent_type_id
func (h *AgentHandler) SetUserAgentConfig(c *gin.Context) {
	agentTypeID, err := strconv.ParseInt(c.Param("agent_type_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent type ID"})
		return
	}

	var req SetUserAgentConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserID(c)

	// Convert to ConfigValues
	configValues := make(agentDomain.ConfigValues)
	for k, v := range req.ConfigValues {
		configValues[k] = v
	}

	config, err := h.userConfigSvc.SetUserAgentConfig(c.Request.Context(), userID, agentTypeID, configValues)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set user config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"config": config.ToResponse()})
}

// DeleteUserAgentConfig deletes the user's personal config for an agent type
// DELETE /api/v1/users/me/agent-configs/:agent_type_id
func (h *AgentHandler) DeleteUserAgentConfig(c *gin.Context) {
	agentTypeID, err := strconv.ParseInt(c.Param("agent_type_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent type ID"})
		return
	}

	userID := middleware.GetUserID(c)

	if err := h.userConfigSvc.DeleteUserAgentConfig(c.Request.Context(), userID, agentTypeID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User config deleted"})
}
