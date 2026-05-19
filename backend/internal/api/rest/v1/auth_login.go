package v1

import (
	"errors"
	"log/slog"
	"net/http"

	domainUser "github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/internal/service/auth"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	result, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "Invalid email or password")
			return
		}
		if errors.Is(err, auth.ErrUserDisabled) {
			apierr.ForbiddenDisabled(c)
			return
		}
		if errors.Is(err, auth.ErrSSOEnforced) {
			apierr.Forbidden(c, apierr.SSO_REQUIRED, "SSO login is required for this domain")
			return
		}
		apierr.InternalError(c, "Authentication failed")
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

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	if err := domainUser.ValidateUsername(req.Username); err != nil {
		apierr.RespondWithExtra(c, http.StatusBadRequest, apierr.VALIDATION_FAILED,
			err.Error(),
			gin.H{"field": "username"})
		return
	}

	result, err := h.authService.Register(c.Request.Context(), &auth.RegisterRequest{
		Email:    req.Email,
		Username: req.Username,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		if errors.Is(err, auth.ErrEmailExists) {
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Email already registered")
			return
		}
		if errors.Is(err, auth.ErrUsernameExists) {
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Username already taken")
			return
		}
		apierr.InternalError(c, "Registration failed")
		return
	}

	verificationToken, err := h.userService.SetEmailVerificationToken(c.Request.Context(), result.User.ID)
	if err != nil {
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

	// 邮件失败不阻塞注册，但 MUST 落日志。
	if err := h.emailService.SendVerificationEmail(c.Request.Context(), result.User.Email, verificationToken); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to send verification email after registration",
			"user_id", result.User.ID, "email", result.User.Email, "error", err)
	}

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

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	result, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidToken) || errors.Is(err, auth.ErrInvalidRefreshToken) {
			apierr.Unauthorized(c, apierr.INVALID_TOKEN, "Invalid refresh token")
			return
		}
		apierr.InternalError(c, "Failed to refresh token")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":         result.Token,
		"refresh_token": result.RefreshToken,
		"expires_in":    result.ExpiresIn,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token != "" && len(token) > 7 {
		token = token[7:]
		h.authService.RevokeToken(c.Request.Context(), token)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}
