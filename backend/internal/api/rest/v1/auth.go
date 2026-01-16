package v1

import (
	"net/http"
	"net/url"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra/email"
	"github.com/anthropics/agentsmesh/backend/internal/service/auth"
	"github.com/anthropics/agentsmesh/backend/internal/service/user"
	"github.com/anthropics/agentsmesh/backend/pkg/auth/oauth"
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

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login handles email/password login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Authenticate user
	result, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if err == auth.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}
		if err == auth.ErrUserDisabled {
			c.JSON(http.StatusForbidden, gin.H{"error": "Account is disabled"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":         result.Token,
		"refresh_token": result.RefreshToken,
		"expires_in":    result.ExpiresIn,
		"user": gin.H{
			"id":         result.User.ID,
			"email":      result.User.Email,
			"username":   result.User.Username,
			"name":       result.User.Name,
			"avatar_url": result.User.AvatarURL,
		},
	})
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name"`
}

// Register handles user registration
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Register user
	result, err := h.authService.Register(c.Request.Context(), &auth.RegisterRequest{
		Email:    req.Email,
		Username: req.Username,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		if err == auth.ErrEmailExists {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
			return
		}
		if err == auth.ErrUsernameExists {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already taken"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration failed"})
		return
	}

	// Generate email verification token and send verification email
	verificationToken, err := h.userService.SetEmailVerificationToken(c.Request.Context(), result.User.ID)
	if err != nil {
		// Log error but don't fail registration
		c.JSON(http.StatusCreated, gin.H{
			"token":         result.Token,
			"refresh_token": result.RefreshToken,
			"expires_in":    result.ExpiresIn,
			"user": gin.H{
				"id":                result.User.ID,
				"email":             result.User.Email,
				"username":          result.User.Username,
				"name":              result.User.Name,
				"is_email_verified": false,
			},
			"message": "Registration successful. Please verify your email.",
		})
		return
	}

	// Send verification email (don't fail registration if email fails)
	_ = h.emailService.SendVerificationEmail(c.Request.Context(), result.User.Email, verificationToken)

	c.JSON(http.StatusCreated, gin.H{
		"token":         result.Token,
		"refresh_token": result.RefreshToken,
		"expires_in":    result.ExpiresIn,
		"user": gin.H{
			"id":                result.User.ID,
			"email":             result.User.Email,
			"username":          result.User.Username,
			"name":              result.User.Name,
			"is_email_verified": false,
		},
		"message": "Registration successful. Please check your email to verify your account.",
	})
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if err == auth.ErrInvalidToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":         result.Token,
		"refresh_token": result.RefreshToken,
		"expires_in":    result.ExpiresIn,
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// Get token from header
	token := c.GetHeader("Authorization")
	if token != "" && len(token) > 7 {
		token = token[7:] // Remove "Bearer " prefix
		// Optionally blacklist the token
		h.authService.RevokeToken(c.Request.Context(), token)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// OAuthRedirect returns a handler for OAuth redirect
func (h *AuthHandler) OAuthRedirect(provider string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get redirect URL from query params (for post-auth redirect)
		redirectTo := c.Query("redirect")
		if redirectTo == "" {
			redirectTo = h.config.OAuth.DefaultRedirectURL
		}

		// Get OAuth provider configuration
		oauthCfg := h.getOAuthConfig(provider)
		if oauthCfg == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "OAuth provider not configured"})
			return
		}

		// Generate state with redirect info
		state, err := h.authService.GenerateOAuthState(c.Request.Context(), provider, redirectTo)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OAuth state"})
			return
		}

		// Build authorization URL
		authURL := oauthCfg.AuthURL(state)
		c.Redirect(http.StatusTemporaryRedirect, authURL)
	}
}

// OAuthCallback returns a handler for OAuth callback
func (h *AuthHandler) OAuthCallback(provider string) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Query("code")
		state := c.Query("state")

		if code == "" {
			errorMsg := c.Query("error")
			errorDesc := c.Query("error_description")
			c.JSON(http.StatusBadRequest, gin.H{
				"error":       errorMsg,
				"description": errorDesc,
			})
			return
		}

		// Validate state and get redirect URL
		redirectTo, err := h.authService.ValidateOAuthState(c.Request.Context(), state)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired state"})
			return
		}

		// Get OAuth provider configuration
		oauthCfg := h.getOAuthConfig(provider)
		if oauthCfg == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "OAuth provider not configured"})
			return
		}

		// Exchange code for token
		token, err := oauthCfg.Exchange(c.Request.Context(), code)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange OAuth code"})
			return
		}

		// Get user info from provider
		userInfo, err := oauthCfg.GetUserInfo(c.Request.Context(), token.AccessToken)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
			return
		}

		// Authenticate or create user
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "OAuth authentication failed"})
			return
		}

		// Redirect with token (for SPA to capture)
		redirectURL, _ := url.Parse(redirectTo)
		q := redirectURL.Query()
		q.Set("token", result.Token)
		q.Set("refresh_token", result.RefreshToken)
		redirectURL.RawQuery = q.Encode()

		c.Redirect(http.StatusTemporaryRedirect, redirectURL.String())
	}
}

// getOAuthConfig returns OAuth configuration for a provider
func (h *AuthHandler) getOAuthConfig(provider string) *oauth.Config {
	switch provider {
	case "github":
		if h.config.OAuth.GitHub.ClientID == "" {
			return nil
		}
		return oauth.NewGitHubConfig(
			h.config.OAuth.GitHub.ClientID,
			h.config.OAuth.GitHub.ClientSecret,
			h.config.OAuth.GitHub.RedirectURL,
		)
	case "google":
		if h.config.OAuth.Google.ClientID == "" {
			return nil
		}
		return oauth.NewGoogleConfig(
			h.config.OAuth.Google.ClientID,
			h.config.OAuth.Google.ClientSecret,
			h.config.OAuth.Google.RedirectURL,
		)
	case "gitlab":
		if h.config.OAuth.GitLab.ClientID == "" {
			return nil
		}
		return oauth.NewGitLabConfig(
			h.config.OAuth.GitLab.ClientID,
			h.config.OAuth.GitLab.ClientSecret,
			h.config.OAuth.GitLab.RedirectURL,
			h.config.OAuth.GitLab.BaseURL,
		)
	case "gitee":
		if h.config.OAuth.Gitee.ClientID == "" {
			return nil
		}
		return oauth.NewGiteeConfig(
			h.config.OAuth.Gitee.ClientID,
			h.config.OAuth.Gitee.ClientSecret,
			h.config.OAuth.Gitee.RedirectURL,
		)
	default:
		return nil
	}
}

// VerifyEmail handles email verification
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	verifiedUser, err := h.userService.VerifyEmail(c.Request.Context(), req.Token)
	if err != nil {
		if err == user.ErrInvalidVerificationToken {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired verification token"})
			return
		}
		if err == user.ErrEmailAlreadyVerified {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email already verified"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify email"})
		return
	}

	// Generate new tokens for the verified user
	result, err := h.authService.GenerateTokens(c.Request.Context(), verifiedUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Email verified successfully",
		"token":         result.Token,
		"refresh_token": result.RefreshToken,
		"expires_in":    result.ExpiresIn,
		"user": gin.H{
			"id":                verifiedUser.ID,
			"email":             verifiedUser.Email,
			"username":          verifiedUser.Username,
			"name":              verifiedUser.Name,
			"is_email_verified": true,
		},
	})
}

// ResendVerification resends the verification email
func (h *AuthHandler) ResendVerification(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user by email
	u, err := h.userService.GetByEmail(c.Request.Context(), req.Email)
	if err != nil {
		// Don't reveal if email exists
		c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a verification link will be sent"})
		return
	}

	// Check if already verified
	if u.IsEmailVerified {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email already verified"})
		return
	}

	// Generate new verification token
	token, err := h.userService.SetEmailVerificationToken(c.Request.Context(), u.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate verification token"})
		return
	}

	// Send verification email
	if err := h.emailService.SendVerificationEmail(c.Request.Context(), u.Email, token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send verification email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Verification email sent"})
}

// ForgotPassword initiates the password reset process
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate password reset token (don't reveal if email exists)
	token, u, err := h.userService.SetPasswordResetToken(c.Request.Context(), req.Email)
	if err != nil {
		// Don't reveal if email exists
		c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a password reset link will be sent"})
		return
	}

	// Send password reset email
	if err := h.emailService.SendPasswordResetEmail(c.Request.Context(), u.Email, token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send password reset email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a password reset link will be sent"})
}

// ResetPassword completes the password reset process
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.userService.ResetPassword(c.Request.Context(), req.Token, req.NewPassword)
	if err != nil {
		if err == user.ErrInvalidResetToken {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired reset token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// Backward compatible function
func RegisterAuthRoutes(rg *gin.RouterGroup, cfg *config.Config, authSvc *auth.Service, userSvc *user.Service, emailSvc email.Service) {
	handler := NewAuthHandler(authSvc, userSvc, emailSvc, cfg)
	handler.RegisterRoutes(rg)
}
