package v1

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/organization"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/anthropics/agentsmesh/backend/pkg/slugkit"
	"github.com/gin-gonic/gin"
)

func (h *OrganizationHandler) CreatePersonalOrganization(c *gin.Context) {
	ctx := c.Request.Context()
	userID := middleware.GetUserID(c)

	u, err := h.userService.GetByID(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "create personal: user lookup failed",
			"user_id", userID, "error", err)
		apierr.InternalError(c, "Failed to load user")
		return
	}

	displayName := ""
	if u.Name != nil {
		displayName = *u.Name
	}

	org, err := h.orgService.CreatePersonal(ctx, userID, u.Username, displayName)
	if err != nil {
		switch {
		case errors.Is(err, slugkit.ErrCollisionExhausted):
			slog.ErrorContext(ctx, "create personal: slug collision exhausted",
				"user_id", userID, "username", u.Username, "error", err)
			apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE,
				"Could not allocate a unique workspace slug after retries. Please try again later.")
		case errors.Is(err, organization.ErrSlugAlreadyExists):
			slog.ErrorContext(ctx, "create personal: race lost on slug insert",
				"user_id", userID, "username", u.Username, "error", err)
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Workspace slug just got taken, please retry.")
		default:
			slog.ErrorContext(ctx, "create personal: unexpected failure",
				"user_id", userID, "username", u.Username, "error", err)
			apierr.InternalError(c, "Failed to create personal workspace")
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"organization": org})
}
