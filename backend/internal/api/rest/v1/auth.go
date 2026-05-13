package v1

import (
	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/service/auth"
	"github.com/anthropics/agentsmesh/backend/pkg/auth/oauth"
	"github.com/gin-gonic/gin"
)

// AuthHandler hosts the OAuth browser-redirect surface that Connect-RPC
// cannot replace: the IdP-initiated GET on `/oauth/:provider` issues a
// 307 to the provider's authorization endpoint, and `/oauth/:provider/
// callback` consumes the IdP's redirect and 307s the SPA back. Connect's
// unary contract cannot return `Location:` headers, so these two
// endpoints stay on REST permanently.
//
// All other auth flows (login / register / refresh / logout / verify /
// resend / forgot / reset) moved to proto.auth.v1.AuthService —
// see backend/internal/api/connect/auth.
type AuthHandler struct {
	authService *auth.Service
	config      *config.Config
}

func NewAuthHandler(authSvc *auth.Service, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authSvc,
		config:      cfg,
	}
}

// RegisterRoutes mounts the OAuth browser-redirect endpoints. Login /
// register / refresh / logout / verify-email / resend-verification /
// forgot-password / reset-password are owned by Connect-RPC.
func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	oauthGroup := rg.Group("/oauth")
	{
		oauthGroup.GET("/github", h.OAuthRedirect("github"))
		oauthGroup.GET("/github/callback", h.OAuthCallback("github"))

		oauthGroup.GET("/google", h.OAuthRedirect("google"))
		oauthGroup.GET("/google/callback", h.OAuthCallback("google"))

		oauthGroup.GET("/gitlab", h.OAuthRedirect("gitlab"))
		oauthGroup.GET("/gitlab/callback", h.OAuthCallback("gitlab"))

		oauthGroup.GET("/gitee", h.OAuthRedirect("gitee"))
		oauthGroup.GET("/gitee/callback", h.OAuthCallback("gitee"))
	}
}

// getOAuthConfig resolves the provider's OAuth config from the loaded
// server config. Returns nil when the provider has no client_id —
// the caller treats that as an unconfigured provider and responds 400.
func (h *AuthHandler) getOAuthConfig(provider string) *oauth.Config {
	switch provider {
	case "github":
		if h.config.OAuth.GitHub.ClientID == "" {
			return nil
		}
		return oauth.NewGitHubConfig(
			h.config.OAuth.GitHub.ClientID,
			h.config.OAuth.GitHub.ClientSecret,
			h.config.GitHubRedirectURL(),
		)
	case "google":
		if h.config.OAuth.Google.ClientID == "" {
			return nil
		}
		return oauth.NewGoogleConfig(
			h.config.OAuth.Google.ClientID,
			h.config.OAuth.Google.ClientSecret,
			h.config.GoogleRedirectURL(),
		)
	case "gitlab":
		if h.config.OAuth.GitLab.ClientID == "" {
			return nil
		}
		return oauth.NewGitLabConfig(
			h.config.OAuth.GitLab.ClientID,
			h.config.OAuth.GitLab.ClientSecret,
			h.config.GitLabRedirectURL(),
			h.config.OAuth.GitLab.BaseURL,
		)
	case "gitee":
		if h.config.OAuth.Gitee.ClientID == "" {
			return nil
		}
		return oauth.NewGiteeConfig(
			h.config.OAuth.Gitee.ClientID,
			h.config.OAuth.Gitee.ClientSecret,
			h.config.GiteeRedirectURL(),
		)
	default:
		return nil
	}
}
