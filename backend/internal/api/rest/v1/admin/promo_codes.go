package admin

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/promocode"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/admin"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// PromoCodeHandler handles admin promo code requests
type PromoCodeHandler struct {
	service *admin.Service
}

// NewPromoCodeHandler creates a new promo code handler
func NewPromoCodeHandler(service *admin.Service) *PromoCodeHandler {
	return &PromoCodeHandler{service: service}
}

// RegisterRoutes registers promo code routes
func (h *PromoCodeHandler) RegisterRoutes(rg *gin.RouterGroup) {
	promoCodes := rg.Group("/promo-codes")
	{
		promoCodes.GET("", h.List)
		promoCodes.POST("", h.Create)
		promoCodes.GET("/:id", h.Get)
		promoCodes.PUT("/:id", h.Update)
		promoCodes.POST("/:id/activate", h.Activate)
		promoCodes.POST("/:id/deactivate", h.Deactivate)
		promoCodes.DELETE("/:id", h.Delete)
		promoCodes.GET("/:id/redemptions", h.ListRedemptions)
	}
}

// List lists promo codes
// GET /api/v1/admin/promo-codes
func (h *PromoCodeHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := &admin.PromoCodeListFilter{
		Page:     page,
		PageSize: pageSize,
	}

	if t := c.Query("type"); t != "" {
		pt := promocode.PromoCodeType(t)
		filter.Type = &pt
	}
	if p := c.Query("plan_name"); p != "" {
		filter.PlanName = &p
	}
	if a := c.Query("is_active"); a != "" {
		isActive := a == "true"
		filter.IsActive = &isActive
	}
	if s := c.Query("search"); s != "" {
		filter.Search = &s
	}

	result, err := h.service.ListPromoCodes(c.Request.Context(), filter)
	if err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreatePromoCodeRequest represents create promo code request body
type CreatePromoCodeRequest struct {
	Code           string `json:"code" binding:"required,min=4,max=50"`
	Name           string `json:"name" binding:"required,min=1,max=100"`
	Description    string `json:"description"`
	Type           string `json:"type" binding:"required,oneof=media partner campaign internal referral"`
	PlanName       string `json:"plan_name" binding:"required,oneof=pro enterprise"`
	DurationMonths int    `json:"duration_months" binding:"required,min=1,max=24"`
	MaxUses        *int   `json:"max_uses"`
	MaxUsesPerOrg  int    `json:"max_uses_per_org"`
	StartsAt       string `json:"starts_at"`
	ExpiresAt      string `json:"expires_at"`
}

// Create creates a new promo code
// POST /api/v1/admin/promo-codes
func (h *PromoCodeHandler) Create(c *gin.Context) {
	adminUserID := c.MustGet("user_id").(int64)

	var req CreatePromoCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	// Parse times
	var startsAt time.Time
	var expiresAt *time.Time

	if req.StartsAt != "" {
		t, err := time.Parse(time.RFC3339, req.StartsAt)
		if err != nil {
			apierr.InvalidInput(c, "invalid starts_at format, use RFC3339")
			return
		}
		startsAt = t
	} else {
		startsAt = time.Now()
	}

	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			apierr.InvalidInput(c, "invalid expires_at format, use RFC3339")
			return
		}
		expiresAt = &t
	}

	maxUsesPerOrg := req.MaxUsesPerOrg
	if maxUsesPerOrg <= 0 {
		maxUsesPerOrg = 1
	}

	promoCode := &promocode.PromoCode{
		Code:           strings.ToUpper(req.Code),
		Name:           req.Name,
		Description:    req.Description,
		Type:           promocode.PromoCodeType(req.Type),
		PlanName:       req.PlanName,
		DurationMonths: req.DurationMonths,
		MaxUses:        req.MaxUses,
		MaxUsesPerOrg:  maxUsesPerOrg,
		StartsAt:       startsAt,
		ExpiresAt:      expiresAt,
		IsActive:       true,
		CreatedByID:    &adminUserID,
	}

	if err := h.service.CreatePromoCode(c.Request.Context(), promoCode, adminUserID); err != nil {
		if err == admin.ErrPromoCodeAlreadyExists {
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "promo code already exists")
			return
		}
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusCreated, promoCode)
}

// Get gets a promo code by ID
// GET /api/v1/admin/promo-codes/:id
func (h *PromoCodeHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "invalid id")
		return
	}

	promoCode, err := h.service.GetPromoCode(c.Request.Context(), id)
	if err != nil {
		if err == admin.ErrPromoCodeNotFound {
			apierr.ResourceNotFound(c, "promo code not found")
			return
		}
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, promoCode)
}

// UpdatePromoCodeRequest represents update promo code request body
type UpdatePromoCodeRequest struct {
	Name          *string `json:"name"`
	Description   *string `json:"description"`
	MaxUses       *int    `json:"max_uses"`
	MaxUsesPerOrg *int    `json:"max_uses_per_org"`
	ExpiresAt     *string `json:"expires_at"`
}

// Update updates a promo code
// PUT /api/v1/admin/promo-codes/:id
func (h *PromoCodeHandler) Update(c *gin.Context) {
	adminUserID := c.MustGet("user_id").(int64)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "invalid id")
		return
	}

	var req UpdatePromoCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	updates := &admin.PromoCodeUpdateInput{
		Name:          req.Name,
		Description:   req.Description,
		MaxUses:       req.MaxUses,
		MaxUsesPerOrg: req.MaxUsesPerOrg,
	}

	if req.ExpiresAt != nil {
		if *req.ExpiresAt == "" {
			// Clear expiration
			updates.ExpiresAt = nil
			updates.ClearExpiresAt = true
		} else {
			t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
			if err != nil {
				apierr.InvalidInput(c, "invalid expires_at format, use RFC3339")
				return
			}
			updates.ExpiresAt = &t
		}
	}

	promoCode, err := h.service.UpdatePromoCode(c.Request.Context(), id, updates, adminUserID)
	if err != nil {
		if err == admin.ErrPromoCodeNotFound {
			apierr.ResourceNotFound(c, "promo code not found")
			return
		}
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, promoCode)
}

// Activate activates a promo code
// POST /api/v1/admin/promo-codes/:id/activate
func (h *PromoCodeHandler) Activate(c *gin.Context) {
	adminUserID := c.MustGet("user_id").(int64)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "invalid id")
		return
	}

	if err := h.service.ActivatePromoCode(c.Request.Context(), id, adminUserID); err != nil {
		if err == admin.ErrPromoCodeNotFound {
			apierr.ResourceNotFound(c, "promo code not found")
			return
		}
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "promo code activated"})
}

// Deactivate deactivates a promo code
// POST /api/v1/admin/promo-codes/:id/deactivate
func (h *PromoCodeHandler) Deactivate(c *gin.Context) {
	adminUserID := c.MustGet("user_id").(int64)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "invalid id")
		return
	}

	if err := h.service.DeactivatePromoCode(c.Request.Context(), id, adminUserID); err != nil {
		if err == admin.ErrPromoCodeNotFound {
			apierr.ResourceNotFound(c, "promo code not found")
			return
		}
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "promo code deactivated"})
}

// Delete deletes a promo code
// DELETE /api/v1/admin/promo-codes/:id
func (h *PromoCodeHandler) Delete(c *gin.Context) {
	adminUserID := c.MustGet("user_id").(int64)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "invalid id")
		return
	}

	if err := h.service.DeletePromoCode(c.Request.Context(), id, adminUserID); err != nil {
		if err == admin.ErrPromoCodeNotFound {
			apierr.ResourceNotFound(c, "promo code not found")
			return
		}
		if err == admin.ErrPromoCodeHasRedemptions {
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "cannot delete promo code with redemptions")
			return
		}
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "promo code deleted"})
}

// ListRedemptions lists redemptions for a promo code
// GET /api/v1/admin/promo-codes/:id/redemptions
func (h *PromoCodeHandler) ListRedemptions(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "invalid id")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.service.ListPromoCodeRedemptions(c.Request.Context(), id, page, pageSize)
	if err != nil {
		if err == admin.ErrPromoCodeNotFound {
			apierr.ResourceNotFound(c, "promo code not found")
			return
		}
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, result)
}
