package v1

import (
	"github.com/gin-gonic/gin"
)

// RegisterOrgScopedRoutes registers organization-scoped routes (require tenant context).
func RegisterOrgScopedRoutes(rg *gin.RouterGroup, svc *Services) {
	registerAgentRoutes(rg, svc)
	registerRepositoryRoutes(rg, svc)
	registerRunnerRoutes(rg, svc)
	registerPodRoutes(rg, svc)
	registerChannelRoutes(rg, svc)
	registerTicketRoutes(rg, svc)
	registerBillingRoutes(rg, svc)
	registerBindingRoutes(rg, svc)
	registerMessageRoutes(rg, svc)
	registerInvitationRoutes(rg, svc)
	registerFileRoutes(rg, svc)
	registerAPIKeyManagementRoutes(rg, svc)
	registerExtensionRoutes(rg, svc)
	registerLoopRoutes(rg, svc)
	registerNotificationRoutes(rg, svc)
	registerTokenUsageRoutes(rg, svc)
	registerBlockstoreRoutes(rg, svc)
}
