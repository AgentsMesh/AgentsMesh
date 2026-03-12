package v1

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	ssoservice "github.com/anthropics/agentsmesh/backend/internal/service/sso"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// OIDCRedirect initiates OIDC authentication
func (h *SSOAuthHandler) OIDCRedirect(c *gin.Context) {
	domain, ok := validateDomain(c)
	if !ok {
		return
	}
	redirectTo := c.Query("redirect")
	if redirectTo == "" {
		redirectTo = h.config.FrontendURL() + "/auth/sso/callback"
	}

	if !h.isAllowedRedirect(redirectTo) {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Invalid redirect URL")
		return
	}

	// Generate state with redirect info (reuse OAuth state mechanism)
	state, err := h.authService.GenerateOAuthState(c.Request.Context(), "sso_oidc_"+domain, redirectTo)
	if err != nil {
		apierr.InternalError(c, "Failed to generate state")
		return
	}

	authURL, err := h.ssoService.GetAuthURL(c.Request.Context(), domain, sso.ProtocolOIDC, state)
	if err != nil {
		if errors.Is(err, ssoservice.ErrConfigNotFound) {
			apierr.ResourceNotFound(c, "SSO config not found")
			return
		}
		apierr.InternalError(c, "Failed to get auth URL")
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// OIDCCallback handles OIDC callback from IdP
func (h *SSOAuthHandler) OIDCCallback(c *gin.Context) {
	domain, ok := validateDomain(c)
	if !ok {
		return
	}
	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		errorMsg := c.Query("error")
		// Log full error details server-side to avoid leaking IdP internals
		slog.Warn("OIDC callback error",
			"domain", domain,
			"error", errorMsg,
			"error_description", c.Query("error_description"),
		)
		// This is a browser redirect flow — redirect back to frontend with error
		// rather than returning JSON which the user's browser can't interpret.
		redirectTo := h.config.FrontendURL() + "/auth/sso/callback"
		if state != "" {
			if rt, err := h.authService.ValidateOAuthState(c.Request.Context(), state); err == nil {
				redirectTo = rt
			}
		}
		// Use a generic error code to avoid leaking IdP internals to the frontend.
		// The raw error is already logged server-side above.
		errorCode := "authentication_failed"
		if errorMsg == "access_denied" {
			errorCode = "access_denied"
		}
		h.redirectWithError(c, redirectTo, errorCode)
		return
	}

	if state == "" {
		redirectTo := h.config.FrontendURL() + "/auth/sso/callback"
		h.redirectWithError(c, redirectTo, "missing_state")
		return
	}

	// Validate state
	redirectTo, err := h.authService.ValidateOAuthState(c.Request.Context(), state)
	if err != nil {
		fallbackRedirect := h.config.FrontendURL() + "/auth/sso/callback"
		h.redirectWithError(c, fallbackRedirect, "invalid_state")
		return
	}

	// Handle callback
	params := map[string]string{"code": code}
	userInfo, configID, err := h.ssoService.HandleCallback(c.Request.Context(), domain, sso.ProtocolOIDC, params)
	if err != nil {
		slog.Error("OIDC callback handling failed", "domain", domain, "error", err)
		h.redirectWithError(c, redirectTo, "authentication_failed")
		return
	}

	// Authenticate, create/get user, and redirect with tokens
	_, tokens, err := h.authenticateSSO(c, sso.ProtocolOIDC, configID, userInfo)
	if err != nil {
		slog.Error("OIDC user authentication failed", "domain", domain, "error", err)
		errorCode := "authentication_failed"
		if errors.Is(err, errAccountDisabled) {
			errorCode = "account_disabled"
		}
		h.redirectWithError(c, redirectTo, errorCode)
		return
	}
	h.redirectWithTokens(c, redirectTo, tokens)
}
