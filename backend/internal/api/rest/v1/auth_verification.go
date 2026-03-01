package v1

import (
	"net/http"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/user"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// VerifyEmail handles email verification
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	verifiedUser, err := h.userService.VerifyEmail(c.Request.Context(), req.Token)
	if err != nil {
		if err == user.ErrInvalidVerificationToken {
			apierr.InvalidInput(c, "Invalid or expired verification token")
			return
		}
		if err == user.ErrEmailAlreadyVerified {
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Email already verified")
			return
		}
		apierr.InternalError(c, "Failed to verify email")
		return
	}

	// Generate new tokens for the verified user
	result, err := h.authService.GenerateTokens(c.Request.Context(), verifiedUser)
	if err != nil {
		apierr.InternalError(c, "Failed to generate tokens")
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
		apierr.ValidationError(c, err.Error())
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
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Email already verified")
		return
	}

	// Generate new verification token
	token, err := h.userService.SetEmailVerificationToken(c.Request.Context(), u.ID)
	if err != nil {
		apierr.InternalError(c, "Failed to generate verification token")
		return
	}

	// Send verification email
	if err := h.emailService.SendVerificationEmail(c.Request.Context(), u.Email, token); err != nil {
		apierr.InternalError(c, "Failed to send verification email")
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
		apierr.ValidationError(c, err.Error())
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
		apierr.InternalError(c, "Failed to send password reset email")
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
		apierr.ValidationError(c, err.Error())
		return
	}

	_, err := h.userService.ResetPassword(c.Request.Context(), req.Token, req.NewPassword)
	if err != nil {
		if err == user.ErrInvalidResetToken {
			apierr.InvalidInput(c, "Invalid or expired reset token")
			return
		}
		apierr.InternalError(c, "Failed to reset password")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}
