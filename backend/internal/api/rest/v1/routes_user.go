package v1

import (
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
//   - proto.agent.v1.UserAgentConfigService owns /me/agent-configs/*
//
// Only the profile routes stay until those migrate too (consumed by Rust
// AuthManager + iOS ffi).
func RegisterUserRoutes(rg *gin.RouterGroup, userSvc *user.Service, orgSvc *organization.Service) {
	userHandler := NewUserHandler(userSvc, orgSvc)

	// REST surface kept for AuthManager (Rust) + iOS ffi consumers only.
	// Profile mutation / identities / search migrated to proto.user.v1.UserService.
	rg.GET("/me", userHandler.GetCurrentUser)
	rg.GET("/me/organizations", userHandler.ListUserOrganizations)

	// User agent configs migrated to Connect-RPC proto.agent.v1.UserAgentConfigService
	// — see backend/internal/api/connect/agent. The REST handlers were
	// removed in the dual-track cleanup.

	// User Repository Providers, Git Credentials, and Agent Credential Profiles
	// migrated to Connect-RPC proto.user_credential.v1 — see
	// backend/internal/api/connect/user_credential. The REST handlers were
	// removed in the dual-track cleanup.

	// User search and other user-scoped REST removed; migrated to
	// proto.user.v1 — see backend/internal/api/connect/user.
}
