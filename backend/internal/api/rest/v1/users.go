package v1

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/organization"
	"github.com/anthropics/agentsmesh/backend/internal/service/user"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// UserHandler keeps REST surface that Connect-RPC cannot replace yet:
//
//   - GET /me — required by the Rust AuthManager bootstrap flow
//     (clients/core/crates/auth/src/api.rs::fetch_me) and the iOS UniFFI
//     bridge (clients/core/crates/ffi/src/services/user.rs). Both consume
//     the wrapped {user: ...} JSON shape today.
//   - GET /me/organizations — required by AuthManager's
//     fetch_organizations (clients/core/crates/auth/src/org.rs).
//
// Profile updates / password / OAuth identities / user search are owned by
// proto.user.v1.UserService (Connect handler in api/connect/user/).
type UserHandler struct {
	userService user.Interface
	orgService  organization.Interface
}

func NewUserHandler(userService user.Interface, orgService organization.Interface) *UserHandler {
	return &UserHandler{
		userService: userService,
		orgService:  orgService,
	}
}

// GetCurrentUser returns current user profile.
// GET /api/v1/users/me — consumed by AuthManager.fetch_me + iOS ffi.
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID := middleware.GetUserID(c)

	u, err := h.userService.GetByID(c.Request.Context(), userID)
	if err != nil {
		apierr.ResourceNotFound(c, "User not found")
		return
	}

	u.PasswordHash = nil

	c.JSON(http.StatusOK, gin.H{"user": u})
}

// ListUserOrganizations lists organizations the user belongs to.
// GET /api/v1/users/me/organizations — consumed by AuthManager.fetch_organizations.
func (h *UserHandler) ListUserOrganizations(c *gin.Context) {
	userID := middleware.GetUserID(c)

	orgs, err := h.orgService.ListByUser(c.Request.Context(), userID)
	if err != nil {
		apierr.InternalError(c, "Failed to list organizations")
		return
	}

	c.JSON(http.StatusOK, gin.H{"organizations": orgs})
}
