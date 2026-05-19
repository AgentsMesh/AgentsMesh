package v1

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/service/auth"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

func (h *AuthHandler) OAuthRedirect(provider string) gin.HandlerFunc {
	return func(c *gin.Context) {
		redirectTo := c.Query("redirect")
		if redirectTo == "" {
			redirectTo = h.config.OAuth.DefaultRedirectURL
		}

		if !h.isAllowedRedirect(redirectTo) {
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Invalid redirect URL")
			return
		}

		oauthCfg := h.getOAuthConfig(provider)
		if oauthCfg == nil {
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "OAuth provider not configured")
			return
		}

		state, err := h.authService.GenerateOAuthState(c.Request.Context(), provider, redirectTo)
		if err != nil {
			apierr.InternalError(c, "Failed to generate OAuth state")
			return
		}

		authURL := oauthCfg.AuthURL(state)
		c.Redirect(http.StatusTemporaryRedirect, authURL)
	}
}

func (h *AuthHandler) isAllowedRedirect(redirectTo string) bool {
	if strings.HasPrefix(redirectTo, "/") && !strings.HasPrefix(redirectTo, "//") {
		return true
	}

	parsed, err := url.Parse(redirectTo)
	if err != nil {
		return false
	}

	if parsed.Scheme == "agentsmesh" {
		return parsed.Host == "oauth" && parsed.Path == "/callback"
	}

	// Port ignored: frontend may run on a different port than API (dev: Next.js :3000, API :80).
	allowedHost, _, _ := strings.Cut(h.config.PrimaryDomain, ":")
	return parsed.Hostname() == allowedHost
}

func (h *AuthHandler) OAuthCallback(provider string) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Query("code")
		state := c.Query("state")

		if code == "" {
			errorMsg := c.Query("error")
			errorDesc := c.Query("error_description")
			apierr.RespondWithExtra(c, http.StatusBadRequest, apierr.VALIDATION_FAILED, errorMsg, gin.H{
				"description": errorDesc,
			})
			return
		}

		redirectTo, err := h.authService.ValidateOAuthState(c.Request.Context(), state)
		if err != nil {
			apierr.InvalidInput(c, "Invalid or expired state")
			return
		}

		oauthCfg := h.getOAuthConfig(provider)
		if oauthCfg == nil {
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "OAuth provider not configured")
			return
		}

		token, err := oauthCfg.Exchange(c.Request.Context(), code)
		if err != nil {
			apierr.InternalError(c, "Failed to exchange OAuth code")
			return
		}

		userInfo, err := oauthCfg.GetUserInfo(c.Request.Context(), token.AccessToken)
		if err != nil {
			apierr.InternalError(c, "Failed to get user info")
			return
		}

		result, err := h.authService.OAuthLogin(c.Request.Context(), &auth.OAuthLoginRequest{
			Provider:       provider,
			ProviderUserID: userInfo.ID,
			Email:          userInfo.Email,
			Username:       userInfo.Username,
			Name:           userInfo.Name,
			AvatarURL:      userInfo.AvatarURL,
			AccessToken:    token.AccessToken,
			RefreshToken:   token.RefreshToken,
			ExpiresAt:      &token.ExpiresAt,
		})
		if err != nil {
			apierr.InternalError(c, "OAuth authentication failed")
			return
		}

		redirectURL, _ := url.Parse(redirectTo)
		q := redirectURL.Query()
		q.Set("token", result.Token)
		q.Set("refresh_token", result.RefreshToken)
		redirectURL.RawQuery = q.Encode()

		c.Redirect(http.StatusTemporaryRedirect, redirectURL.String())
	}
}
