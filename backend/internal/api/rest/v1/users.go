package v1

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/organization"
	"github.com/anthropics/agentsmesh/backend/internal/service/user"
	"github.com/gin-gonic/gin"
)

// UserHandler handles user-related requests
type UserHandler struct {
	userService user.Interface
	orgService  organization.Interface
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService user.Interface, orgService organization.Interface) *UserHandler {
	return &UserHandler{
		userService: userService,
		orgService:  orgService,
	}
}

// GetCurrentUser returns current user profile
// GET /api/v1/users/me
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID := middleware.GetUserID(c)

	u, err := h.userService.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Don't return password hash
	u.PasswordHash = nil

	c.JSON(http.StatusOK, gin.H{"user": u})
}

// UpdateProfileRequest represents profile update request
type UpdateProfileRequest struct {
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// UpdateCurrentUser updates current user profile
// PUT /api/v1/users/me
func (h *UserHandler) UpdateCurrentUser(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	u.PasswordHash = nil

	c.JSON(http.StatusOK, gin.H{"user": u})
}

// ChangePasswordRequest represents password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// ChangePassword changes user's password
// POST /api/v1/users/me/password
func (h *UserHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserID(c)

	u, err := h.userService.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Verify current password
	_, err = h.userService.Authenticate(c.Request.Context(), u.Email, req.CurrentPassword)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect"})
		return
	}

	if err := h.userService.UpdatePassword(c.Request.Context(), userID, req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to change password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// ListUserOrganizations lists organizations user belongs to
// GET /api/v1/users/me/organizations
func (h *UserHandler) ListUserOrganizations(c *gin.Context) {
	userID := middleware.GetUserID(c)

	orgs, err := h.orgService.ListByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list organizations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"organizations": orgs})
}

// ListIdentities lists user's OAuth identities
// GET /api/v1/users/me/identities
func (h *UserHandler) ListIdentities(c *gin.Context) {
	userID := middleware.GetUserID(c)

	identities, err := h.userService.ListIdentities(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list identities"})
		return
	}

	// Don't return tokens
	for _, identity := range identities {
		identity.AccessTokenEncrypted = nil
		identity.RefreshTokenEncrypted = nil
	}

	c.JSON(http.StatusOK, gin.H{"identities": identities})
}

// DeleteIdentity removes an OAuth identity
// DELETE /api/v1/users/me/identities/:provider
func (h *UserHandler) DeleteIdentity(c *gin.Context) {
	provider := c.Param("provider")
	userID := middleware.GetUserID(c)

	// Check that user has another way to login
	u, err := h.userService.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	identities, err := h.userService.ListIdentities(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check identities"})
		return
	}

	// Must have password or another identity
	if u.PasswordHash == nil && len(identities) <= 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot remove last login method"})
		return
	}

	if err := h.userService.DeleteIdentity(c.Request.Context(), userID, provider); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove identity"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Identity removed"})
}

// SearchUsersRequest represents user search request
type SearchUsersRequest struct {
	Query string `form:"q" binding:"required,min=2"`
	Limit int    `form:"limit" binding:"omitempty,min=1,max=50"`
}

// SearchUsers searches for users
// GET /api/v1/users/search
func (h *UserHandler) SearchUsers(c *gin.Context) {
	var req SearchUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	limit := req.Limit
	if limit == 0 {
		limit = 10
	}

	users, err := h.userService.Search(c.Request.Context(), req.Query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search users"})
		return
	}

	// Don't return sensitive data
	for _, u := range users {
		u.PasswordHash = nil
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}
