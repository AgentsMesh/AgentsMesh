package admin

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/promocode"
	"github.com/anthropics/agentsmesh/backend/internal/service/admin"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
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
