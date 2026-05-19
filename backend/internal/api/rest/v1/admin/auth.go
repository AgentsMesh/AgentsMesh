package admin

import (
	"context"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/auth"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"

	"github.com/gin-gonic/gin"
)

type authServiceInterface interface {
	Login(ctx context.Context, email, password string) (*auth.LoginResult, error)
}

type AuthHandler struct {
	authService authServiceInterface
	config      *config.Config
}

func NewAuthHandler(authSvc *auth.Service, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authSvc,
		config:      cfg,
	}
}

func NewAuthHandlerWithInterface(authSvc authServiceInterface, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authSvc,
		config:      cfg,
	}
}

func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	authGroup := rg.Group("/auth")
	{
		authGroup.POST("/login", h.Login)
	}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.InvalidInput(c, "Invalid request: email and password required")
		return
	}

	result, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "Invalid email or password")
		return
	}

	if !result.User.IsSystemAdmin {
		apierr.Forbidden(c, apierr.SYSTEM_ADMIN_REQUIRED, "Your account does not have system administrator privileges")
		return
	}

	if !result.User.IsActive {
		apierr.ForbiddenDisabled(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":         result.Token,
		"refresh_token": result.RefreshToken,
		"user":          adminUserResponse(result.User),
	})
}

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
