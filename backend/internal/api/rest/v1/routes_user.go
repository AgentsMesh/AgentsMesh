package v1

import (
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/service/organization"
	"github.com/anthropics/agentsmesh/backend/internal/service/user"
	"github.com/gin-gonic/gin"
)

// RegisterUserRoutes registers user routes
func RegisterUserRoutes(rg *gin.RouterGroup, userSvc *user.Service, orgSvc *organization.Service, agentSvc *agent.AgentService, credentialSvc *agent.CredentialProfileService, userConfigSvc *agent.UserConfigService, agentpodSettingsSvc *agentpod.SettingsService, agentpodAIProviderSvc *agentpod.AIProviderService) {
	userHandler := NewUserHandler(userSvc, orgSvc)
	agentHandler := NewAgentHandler(agentSvc, credentialSvc, userConfigSvc)

	// Profile routes
	rg.GET("/me", userHandler.GetCurrentUser)
	rg.PUT("/me", userHandler.UpdateCurrentUser)
	rg.POST("/me/password", userHandler.ChangePassword)
	rg.GET("/me/organizations", userHandler.ListUserOrganizations)
	rg.GET("/me/identities", userHandler.ListIdentities)
	rg.DELETE("/me/identities/:provider", userHandler.DeleteIdentity)

	// User agent configs (personal runtime configuration)
	rg.GET("/me/agent-configs", agentHandler.ListUserAgentConfigs)
	rg.GET("/me/agent-configs/:slug", agentHandler.GetUserAgentConfig)
	rg.PUT("/me/agent-configs/:slug", agentHandler.SetUserAgentConfig)
	rg.DELETE("/me/agent-configs/:slug", agentHandler.DeleteUserAgentConfig)

	// AgentPod settings routes
	if agentpodSettingsSvc != nil && agentpodAIProviderSvc != nil {
		agentpodHandler := NewAgentPodHandler(agentpodSettingsSvc, agentpodAIProviderSvc)
		agentpodGroup := rg.Group("/me/agentpod")
		{
			// Settings
			agentpodGroup.GET("/settings", agentpodHandler.GetSettings)
			agentpodGroup.PUT("/settings", agentpodHandler.UpdateSettings)

			// AI Providers
			providers := agentpodGroup.Group("/providers")
			{
				providers.GET("", agentpodHandler.ListProviders)
				providers.POST("", agentpodHandler.CreateProvider)
				providers.PUT("/:id", agentpodHandler.UpdateProvider)
				providers.DELETE("/:id", agentpodHandler.DeleteProvider)
				providers.POST("/:id/default", agentpodHandler.SetDefaultProvider)
			}
		}
	}

	// User Repository Providers (for importing repositories)
	repositoryProviderHandler := NewUserRepositoryProviderHandler(userSvc)
	repositoryProviderHandler.RegisterRoutes(rg)

	// User Git Credentials (for Git operations)
	gitCredentialHandler := NewUserGitCredentialHandler(userSvc)
	gitCredentialHandler.RegisterRoutes(rg)

	// User Agent Credential Profiles (for agent API credentials)
	agentCredentialHandler := NewUserAgentCredentialHandler(credentialSvc)
	agentCredentialHandler.RegisterRoutes(rg)

	// User search
	rg.GET("/search", userHandler.SearchUsers)
}

// RegisterOrganizationRoutes registers organization routes
func RegisterOrganizationRoutes(rg *gin.RouterGroup, orgSvc *organization.Service, userSvc *user.Service) {
	handler := NewOrganizationHandler(orgSvc, userSvc)

	// Organization CRUD
	rg.GET("", handler.ListOrganizations)
	rg.POST("", handler.CreateOrganization)
	rg.GET("/:slug", handler.GetOrganization)
	rg.PUT("/:slug", handler.UpdateOrganization)
	rg.DELETE("/:slug", handler.DeleteOrganization)

	// Member management
	rg.GET("/:slug/members", handler.ListMembers)
	rg.POST("/:slug/members", handler.InviteMember)
	rg.PUT("/:slug/members/:user_id", handler.UpdateMemberRole)
	rg.DELETE("/:slug/members/:user_id", handler.RemoveMember)
}
