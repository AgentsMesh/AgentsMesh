package v1

import (
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/anthropics/agentsmesh/backend/internal/service/organization"
	"github.com/anthropics/agentsmesh/backend/internal/service/user"
	"github.com/gin-gonic/gin"
)

// RegisterUserRoutes registers the residual /me/* REST surface.
// The data-plane migration has moved several sub-surfaces out of this
// file:
//   - proto.pod.v1.AgentPodSettingsService owns /me/agentpod/*
//   - proto.user_credential.v1 owns /me/{repository-providers,
//     git-credentials,agent-credentials}/*
//
// Only the profile + agent-configs + search routes stay until those
// migrate too.
func RegisterUserRoutes(rg *gin.RouterGroup, userSvc *user.Service, orgSvc *organization.Service, agentSvc *agent.AgentService, credentialSvc *agent.CredentialProfileService, userConfigSvc *agent.UserConfigService) {
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

	// User Repository Providers, Git Credentials, and Agent Credential Profiles
	// migrated to Connect-RPC proto.user_credential.v1 — see
	// backend/internal/api/connect/user_credential. The REST handlers were
	// removed in the dual-track cleanup.

	// User search
	rg.GET("/search", userHandler.SearchUsers)
}
