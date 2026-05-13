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

	// User Repository Providers, Git Credentials, and Agent Credential Profiles
	// migrated to Connect-RPC proto.user_credential.v1 — see
	// backend/internal/api/connect/user_credential. The REST handlers were
	// removed in the dual-track cleanup.

	// User search
	rg.GET("/search", userHandler.SearchUsers)
}
