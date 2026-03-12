package v1

import (
	"errors"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	ssoservice "github.com/anthropics/agentsmesh/backend/internal/service/sso"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// LDAPAuthRequest represents LDAP authentication request
type LDAPAuthRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LDAPAuth handles LDAP authentication
func (h *SSOAuthHandler) LDAPAuth(c *gin.Context) {
	domain, ok := validateDomain(c)
	if !ok {
		return
	}

	var req LDAPAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, "Username and password are required")
		return
	}

	userInfo, configID, err := h.ssoService.AuthenticateLDAP(c.Request.Context(), domain, req.Username, req.Password)
	if err != nil {
		if errors.Is(err, ssoservice.ErrConfigNotFound) {
			apierr.ResourceNotFound(c, "SSO config not found")
			return
		}
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "LDAP authentication failed")
		return
	}

	// Authenticate, create/get user, and generate tokens
	u, tokens, err := h.authenticateSSO(c, sso.ProtocolLDAP, configID, userInfo)
	if err != nil {
		if errors.Is(err, errAccountDisabled) {
			apierr.Forbidden(c, apierr.ACCOUNT_DISABLED, "Account is disabled")
		} else {
			apierr.InternalError(c, "Failed to process authentication")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":         tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_at":    tokens.ExpiresAt,
		"token_type":    "Bearer",
		"user": gin.H{
			"id":       u.ID,
			"email":    u.Email,
			"username": u.Username,
			"name":     u.Name,
		},
	})
}
