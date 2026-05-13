package v1

import (
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	extensionservice "github.com/anthropics/agentsmesh/backend/internal/service/extension"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// maxSkillUploadSize is the maximum file size allowed for skill uploads (50MB)
const maxSkillUploadSize = 50 << 20

// handleServiceError maps service-layer errors to appropriate HTTP error responses
func handleServiceError(c *gin.Context, err error, fallbackMsg string) {
	switch {
	case errors.Is(err, extensionservice.ErrNotFound):
		apierr.ResourceNotFound(c, err.Error())
	case errors.Is(err, extensionservice.ErrForbidden):
		apierr.ForbiddenAdmin(c)
	case errors.Is(err, extensionservice.ErrInvalidScope), errors.Is(err, extensionservice.ErrInvalidInput):
		apierr.ValidationError(c, err.Error())
	case errors.Is(err, extensionservice.ErrAlreadyInstalled):
		apierr.Conflict(c, apierr.ALREADY_EXISTS, err.Error())
	default:
		apierr.InternalError(c, fallbackMsg)
	}
}

// requireOrgAdmin checks if the current user has admin or owner role.
// Returns true if the user is authorized, false otherwise (and sends 403 response).
func requireOrgAdmin(c *gin.Context) bool {
	tenant := middleware.GetTenant(c)
	if tenant.UserRole != "admin" && tenant.UserRole != "owner" {
		apierr.ForbiddenAdmin(c)
		return false
	}
	return true
}

// ExtensionHandler handles extension-related API endpoints (skill / mcp install
// surface — market list RPCs migrated to Connect).
type ExtensionHandler struct {
	extensionSvc *extensionservice.Service
}

// NewExtensionHandler creates a new extension handler
func NewExtensionHandler(extensionSvc *extensionservice.Service) *ExtensionHandler {
	return &ExtensionHandler{
		extensionSvc: extensionSvc,
	}
}

