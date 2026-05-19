package admin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	domainUser "github.com/anthropics/agentsmesh/backend/internal/domain/user"
	adminservice "github.com/anthropics/agentsmesh/backend/internal/service/admin"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	adminService *adminservice.Service
}

func NewUserHandler(adminSvc *adminservice.Service) *UserHandler {
	return &UserHandler{
		adminService: adminSvc,
	}
}

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
		usersGroup.POST("/:id/verify-email", h.VerifyUserEmail)
		usersGroup.POST("/:id/unverify-email", h.UnverifyUserEmail)
	}
}

func (h *UserHandler) logAction(c *gin.Context, action admin.AuditAction, targetType admin.TargetType, targetID int64, oldData, newData interface{}) {
	LogAdminAction(c, h.adminService, action, targetType, targetID, oldData, newData)
}

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

func (h *UserHandler) GetUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid user ID")
		return
	}

	user, err := h.adminService.GetUser(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, adminservice.ErrUserNotFound) {
			apierr.ResourceNotFound(c, "User not found")
			return
		}
		apierr.InternalError(c, "Failed to get user")
		return
	}

	h.logAction(c, admin.AuditActionUserView, admin.TargetTypeUser, userID, nil, nil)

	c.JSON(http.StatusOK, adminUserResponse(user))
}

type UpdateUserRequest struct {
	Name     *string `json:"name"`
	Username *string `json:"username"`
	Email    *string `json:"email"`
}

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

	oldUser, _ := h.adminService.GetUser(c.Request.Context(), userID)

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Username != nil {
		if err := domainUser.ValidateUsername(*req.Username); err != nil {
			apierr.RespondWithExtra(c, http.StatusBadRequest, apierr.VALIDATION_FAILED,
				err.Error(),
				gin.H{"field": "username"})
			return
		}
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
		if errors.Is(err, adminservice.ErrUserNotFound) {
			apierr.ResourceNotFound(c, "User not found")
			return
		}
		if errors.Is(err, adminservice.ErrUsernameAlreadyExists) {
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Username already taken")
			return
		}
		if errors.Is(err, adminservice.ErrEmailAlreadyExists) {
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Email already in use")
			return
		}
		apierr.InternalError(c, "Failed to update user")
		return
	}

	h.logAction(c, admin.AuditActionUserUpdate, admin.TargetTypeUser, userID, oldUser, user)

	c.JSON(http.StatusOK, adminUserResponse(user))
}
