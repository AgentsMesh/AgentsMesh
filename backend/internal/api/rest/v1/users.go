package v1

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/organization"
	"github.com/anthropics/agentsmesh/backend/internal/service/user"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService user.Interface
	orgService  organization.Interface
}

func NewUserHandler(userService user.Interface, orgService organization.Interface) *UserHandler {
	return &UserHandler{
		userService: userService,
		orgService:  orgService,
	}
}

func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID := middleware.GetUserID(c)

	u, err := h.userService.GetByID(c.Request.Context(), userID)
	if err != nil {
		apierr.ResourceNotFound(c, "User not found")
		return
	}

	u.PasswordHash = nil

	c.JSON(http.StatusOK, gin.H{"user": u})
}

type UpdateProfileRequest struct {
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

func (h *UserHandler) UpdateCurrentUser(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.AvatarURL != "" {
		updates["avatar_url"] = req.AvatarURL
	}

	u, err := h.userService.Update(c.Request.Context(), userID, updates)
	if err != nil {
		apierr.InternalError(c, "Failed to update profile")
		return
	}

	u.PasswordHash = nil

	c.JSON(http.StatusOK, gin.H{"user": u})
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)

	u, err := h.userService.GetByID(c.Request.Context(), userID)
	if err != nil {
		apierr.ResourceNotFound(c, "User not found")
		return
	}

	_, err = h.userService.Authenticate(c.Request.Context(), u.Email, req.CurrentPassword)
	if err != nil {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "Current password is incorrect")
		return
	}

	if err := h.userService.UpdatePassword(c.Request.Context(), userID, req.NewPassword); err != nil {
		apierr.InternalError(c, "Failed to change password")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

func (h *UserHandler) ListUserOrganizations(c *gin.Context) {
	userID := middleware.GetUserID(c)

	orgs, err := h.orgService.ListByUser(c.Request.Context(), userID)
	if err != nil {
		apierr.InternalError(c, "Failed to list organizations")
		return
	}

	c.JSON(http.StatusOK, gin.H{"organizations": orgs})
}

func (h *UserHandler) ListIdentities(c *gin.Context) {
	userID := middleware.GetUserID(c)

	identities, err := h.userService.ListIdentities(c.Request.Context(), userID)
	if err != nil {
		apierr.InternalError(c, "Failed to list identities")
		return
	}

	for _, identity := range identities {
		identity.AccessTokenEncrypted = nil
		identity.RefreshTokenEncrypted = nil
	}

	c.JSON(http.StatusOK, gin.H{"identities": identities})
}

func (h *UserHandler) DeleteIdentity(c *gin.Context) {
	provider := c.Param("provider")
	userID := middleware.GetUserID(c)

	u, err := h.userService.GetByID(c.Request.Context(), userID)
	if err != nil {
		apierr.ResourceNotFound(c, "User not found")
		return
	}

	identities, err := h.userService.ListIdentities(c.Request.Context(), userID)
	if err != nil {
		apierr.InternalError(c, "Failed to check identities")
		return
	}

	if u.PasswordHash == nil && len(identities) <= 1 {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Cannot remove last login method")
		return
	}

	if err := h.userService.DeleteIdentity(c.Request.Context(), userID, provider); err != nil {
		apierr.InternalError(c, "Failed to remove identity")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Identity removed"})
}

type SearchUsersRequest struct {
	Query string `form:"q" binding:"required,min=2"`
	Limit int    `form:"limit" binding:"omitempty,min=1,max=50"`
}

func (h *UserHandler) SearchUsers(c *gin.Context) {
	var req SearchUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	limit := req.Limit
	if limit == 0 {
		limit = 10
	}

	users, err := h.userService.Search(c.Request.Context(), req.Query, limit)
	if err != nil {
		apierr.InternalError(c, "Failed to search users")
		return
	}

	for _, u := range users {
		u.PasswordHash = nil
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}
