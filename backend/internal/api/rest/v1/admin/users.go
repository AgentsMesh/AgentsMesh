package admin

import (
	"net/http"
	"strconv"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/admin"
	adminservice "github.com/AgentsMesh/AgentsMesh/backend/internal/service/admin"

	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// UserHandler handles user management requests
type UserHandler struct {
	adminService *adminservice.Service
}

// NewUserHandler creates a new user handler
func NewUserHandler(adminSvc *adminservice.Service) *UserHandler {
	return &UserHandler{
		adminService: adminSvc,
	}
}

// RegisterRoutes registers user management routes
func (h *UserHandler) RegisterRoutes(rg *gin.RouterGroup) {
	usersGroup := rg.Group("/users")
	{
		usersGroup.GET("", h.ListUsers)
		usersGroup.GET("/:id", h.GetUser)
		usersGroup.PUT("/:id", h.UpdateUser)
		usersGroup.POST("/:id/disable", h.DisableUser)
		usersGroup.POST("/:id/enable", h.EnableUser)
		usersGroup.POST("/:id/grant-admin", h.GrantAdmin)
		usersGroup.POST("/:id/revoke-admin", h.RevokeAdmin)
	}
}

// logAction is a helper method that delegates to the shared LogAdminAction function
func (h *UserHandler) logAction(c *gin.Context, action admin.AuditAction, targetType admin.TargetType, targetID int64, oldData, newData interface{}) {
	LogAdminAction(c, h.adminService, action, targetType, targetID, oldData, newData)
}

// ListUsers returns a list of users with pagination
func (h *UserHandler) ListUsers(c *gin.Context) {
	query := &adminservice.UserListQuery{
		Search:   c.Query("search"),
		Page:     1,
		PageSize: 20,
	}

	if page, err := strconv.Atoi(c.Query("page")); err == nil {
		query.Page = page
	}
	if pageSize, err := strconv.Atoi(c.Query("page_size")); err == nil {
		query.PageSize = pageSize
	}
	if isActive := c.Query("is_active"); isActive != "" {
		active := isActive == "true"
		query.IsActive = &active
	}
	if isAdmin := c.Query("is_admin"); isAdmin != "" {
		adminFlag := isAdmin == "true"
		query.IsAdmin = &adminFlag
	}

	result, err := h.adminService.ListUsers(c.Request.Context(), query)
	if err != nil {
		apierr.InternalError(c, "Failed to list users")
		return
	}

	// Convert to response format
	users := make([]gin.H, len(result.Data))
	for i, u := range result.Data {
		users[i] = adminUserResponse(&u)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        users,
		"total":       result.Total,
		"page":        result.Page,
		"page_size":   result.PageSize,
		"total_pages": result.TotalPages,
	})
}

// GetUser returns a single user
func (h *UserHandler) GetUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid user ID")
		return
	}

	user, err := h.adminService.GetUser(c.Request.Context(), userID)
	if err != nil {
		if err == adminservice.ErrUserNotFound {
			apierr.ResourceNotFound(c, "User not found")
			return
		}
		apierr.InternalError(c, "Failed to get user")
		return
	}

	// Log view action
	h.logAction(c, admin.AuditActionUserView, admin.TargetTypeUser, userID, nil, nil)

	c.JSON(http.StatusOK, adminUserResponse(user))
}

// UpdateUserRequest represents the request body for updating a user
type UpdateUserRequest struct {
	Name     *string `json:"name"`
	Username *string `json:"username"`
	Email    *string `json:"email"`
}

// UpdateUser updates a user's profile
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid user ID")
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	// Get old data for audit log
	oldUser, _ := h.adminService.GetUser(c.Request.Context(), userID)

	// Build updates map
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Username != nil {
		updates["username"] = *req.Username
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}

	if len(updates) == 0 {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "No updates provided")
		return
	}

	user, err := h.adminService.UpdateUser(c.Request.Context(), userID, updates)
	if err != nil {
		if err == adminservice.ErrUserNotFound {
			apierr.ResourceNotFound(c, "User not found")
			return
		}
		apierr.InternalError(c, "Failed to update user")
		return
	}

	// Log update action
	h.logAction(c, admin.AuditActionUserUpdate, admin.TargetTypeUser, userID, oldUser, user)

	c.JSON(http.StatusOK, adminUserResponse(user))
}
