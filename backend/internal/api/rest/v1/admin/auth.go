package admin

import (
	"context"
	"net/http"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/config"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/auth"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"

	"github.com/gin-gonic/gin"
)

// authServiceInterface defines the interface for auth service used by AuthHandler
// This enables mocking in tests
type authServiceInterface interface {
	Login(ctx context.Context, email, password string) (*auth.LoginResult, error)
}

// AuthHandler handles admin authentication requests
type AuthHandler struct {
	authService authServiceInterface
	config      *config.Config
}

// NewAuthHandler creates a new admin auth handler
func NewAuthHandler(authSvc *auth.Service, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authSvc,
		config:      cfg,
	}
}

// NewAuthHandlerWithInterface creates a new admin auth handler with custom auth service interface (for testing)
func NewAuthHandlerWithInterface(authSvc authServiceInterface, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authSvc,
		config:      cfg,
	}
}

// RegisterRoutes registers admin authentication routes
func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	authGroup := rg.Group("/auth")
	{
		// Username/password login
		authGroup.POST("/login", h.Login)
	}
}

// LoginRequest represents the admin login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login handles admin login with email and password
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.InvalidInput(c, "Invalid request: email and password required")
		return
	}

	// Authenticate user
	result, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "Invalid email or password")
		return
	}

	// Verify user is a system admin
	if !result.User.IsSystemAdmin {
		apierr.Forbidden(c, apierr.SYSTEM_ADMIN_REQUIRED, "Your account does not have system administrator privileges")
		return
	}

	// Verify user is active
	if !result.User.IsActive {
		apierr.ForbiddenDisabled(c)
		return
	}

	// Return tokens and user info
	c.JSON(http.StatusOK, gin.H{
		"token":         result.Token,
		"refresh_token": result.RefreshToken,
		"user":          adminUserResponse(result.User),
	})
}

// GetMe returns the current admin user's information
func (h *AuthHandler) GetMe(c *gin.Context) {
	u := middleware.GetAdminUser(c)
	if u == nil {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "Not authenticated")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":              u.ID,
		"email":           u.Email,
		"username":        u.Username,
		"name":            u.Name,
		"avatar_url":      u.AvatarURL,
		"is_system_admin": u.IsSystemAdmin,
	})
}

// adminUserResponse creates a sanitized user response for admin API
func adminUserResponse(u *user.User) gin.H {
	return gin.H{
		"id":                u.ID,
		"email":             u.Email,
		"username":          u.Username,
		"name":              u.Name,
		"avatar_url":        u.AvatarURL,
		"is_active":         u.IsActive,
		"is_system_admin":   u.IsSystemAdmin,
		"is_email_verified": u.IsEmailVerified,
		"last_login_at":     u.LastLoginAt,
		"created_at":        u.CreatedAt,
		"updated_at":        u.UpdatedAt,
	}
}
