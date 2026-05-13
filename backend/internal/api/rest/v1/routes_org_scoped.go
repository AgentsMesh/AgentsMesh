package v1

import (
	"github.com/gin-gonic/gin"
)

// RegisterOrgScopedRoutes registers organization-scoped routes (require tenant context).
func RegisterOrgScopedRoutes(rg *gin.RouterGroup, svc *Services) {
	registerRunnerRoutes(rg, svc)
	registerTicketRoutes(rg, svc)
	registerBillingRoutes(rg, svc)
	registerBindingRoutes(rg, svc)
	registerMessageRoutes(rg, svc)
	registerInvitationRoutes(rg, svc)
	registerFileRoutes(rg, svc)
	registerExtensionRoutes(rg, svc)
}
