package v1

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	"github.com/anthropics/agentsmesh/backend/internal/service/auth"
	ssoservice "github.com/anthropics/agentsmesh/backend/internal/service/sso"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

func (h *SSOAuthHandler) SAMLRedirect(c *gin.Context) {
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

	state, err := h.authService.GenerateOAuthState(c.Request.Context(), "sso_saml_"+domain, redirectTo)
	if err != nil {
		apierr.InternalError(c, "Failed to generate state")
		return
	}

	authURL, err := h.ssoService.GetAuthURL(c.Request.Context(), domain, sso.ProtocolSAML, state)
	if err != nil {
		if errors.Is(err, ssoservice.ErrConfigNotFound) {
			apierr.ResourceNotFound(c, "SSO config not found")
			return
		}
		apierr.InternalError(c, "Failed to get SAML auth URL")
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

func (h *SSOAuthHandler) SAMLACS(c *gin.Context) {
	domain, ok := validateDomain(c)
	if !ok {
		return
	}
	samlResponse := c.PostForm("SAMLResponse")
	relayState := c.PostForm("RelayState")

	if samlResponse == "" {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Missing SAMLResponse")
		return
	}

	var redirectTo string
	if relayState != "" {
		var err error
		redirectTo, err = h.authService.ValidateOAuthState(c.Request.Context(), relayState)
		if err != nil {
			redirectTo = h.config.FrontendURL() + "/auth/sso/callback"
			h.redirectWithError(c, redirectTo, "invalid_state")
			return
		}
	} else {
		redirectTo = h.config.FrontendURL() + "/auth/sso/callback"
	}

	params := map[string]string{
		"SAMLResponse": samlResponse,
		"RelayState":   relayState,
	}
	userInfo, configID, err := h.ssoService.HandleCallback(c.Request.Context(), domain, sso.ProtocolSAML, params)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "SAML callback handling failed", "domain", domain, "error", err)
		h.redirectWithError(c, redirectTo, "authentication_failed")
		return
	}

	_, tokens, err := h.authenticateSSO(c, sso.ProtocolSAML, configID, userInfo)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "SAML user authentication failed", "domain", domain, "error", err)
		errorCode := "authentication_failed"
		if errors.Is(err, auth.ErrUserDisabled) {
			errorCode = "account_disabled"
		}
		h.redirectWithError(c, redirectTo, errorCode)
		return
	}
	h.redirectWithTokens(c, redirectTo, tokens)
}

func (h *SSOAuthHandler) SAMLMetadata(c *gin.Context) {
	domain, ok := validateDomain(c)
	if !ok {
		return
	}

	metadata, err := h.ssoService.GetSAMLMetadata(c.Request.Context(), domain)
	if err != nil {
		if errors.Is(err, ssoservice.ErrConfigNotFound) {
			apierr.ResourceNotFound(c, "SSO config not found")
			return
		}
		apierr.InternalError(c, "Failed to generate metadata")
		return
	}

	c.Data(http.StatusOK, "application/xml", metadata)
}
