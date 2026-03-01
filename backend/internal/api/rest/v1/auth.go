package v1

import (
	"github.com/AgentsMesh/AgentsMesh/backend/internal/config"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/infra/email"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/auth"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/user"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/auth/oauth"
	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	authService  *auth.Service
	userService  *user.Service
	emailService email.Service
	config       *config.Config
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authSvc *auth.Service, userSvc *user.Service, emailSvc email.Service, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService:  authSvc,
		userService:  userSvc,
		emailService: emailSvc,
		config:       cfg,
	}
}

// RegisterRoutes registers authentication routes
func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/login", h.Login)
	rg.POST("/register", h.Register)
	rg.POST("/refresh", h.RefreshToken)
	rg.POST("/logout", h.Logout)

	// Email verification routes
	rg.POST("/verify-email", h.VerifyEmail)
	rg.POST("/resend-verification", h.ResendVerification)

	// Password reset routes
	rg.POST("/forgot-password", h.ForgotPassword)
	rg.POST("/reset-password", h.ResetPassword)

	// OAuth routes
	oauthGroup := rg.Group("/oauth")
	{
		// GitHub
		oauthGroup.GET("/github", h.OAuthRedirect("github"))
		oauthGroup.GET("/github/callback", h.OAuthCallback("github"))

		// Google
		oauthGroup.GET("/google", h.OAuthRedirect("google"))
		oauthGroup.GET("/google/callback", h.OAuthCallback("google"))

		// GitLab
		oauthGroup.GET("/gitlab", h.OAuthRedirect("gitlab"))
		oauthGroup.GET("/gitlab/callback", h.OAuthCallback("gitlab"))

		// Gitee
		oauthGroup.GET("/gitee", h.OAuthRedirect("gitee"))
		oauthGroup.GET("/gitee/callback", h.OAuthCallback("gitee"))
	}
}

// getOAuthConfig returns OAuth configuration for a provider
func (h *AuthHandler) getOAuthConfig(provider string) *oauth.Config {
	// RedirectURLs are derived from PrimaryDomain
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
