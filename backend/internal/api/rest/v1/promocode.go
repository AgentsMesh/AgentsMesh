package v1

import (
	"net/http"
	"strconv"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/promocode"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	promocodeSvc "github.com/anthropics/agentsmesh/backend/internal/service/promocode"
	"github.com/gin-gonic/gin"
)

// PromoCodeHandler handles promo code HTTP requests
type PromoCodeHandler struct {
	service *promocodeSvc.Service
}

// NewPromoCodeHandler creates a new promo code handler
func NewPromoCodeHandler(service *promocodeSvc.Service) *PromoCodeHandler {
	return &PromoCodeHandler{service: service}
}

// ============ User API ============

// ValidatePromoCodeRequest represents validate request body
type ValidatePromoCodeRequest struct {
	Code string `json:"code" binding:"required"`
}

// Validate validates a promo code
// POST /api/v1/orgs/:slug/billing/promo-codes/validate
func (h *PromoCodeHandler) Validate(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	var req ValidatePromoCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.Validate(c.Request.Context(), &promocodeSvc.ValidateRequest{
		Code:           req.Code,
		OrganizationID: tenant.OrganizationID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// RedeemPromoCodeRequest represents redeem request body
type RedeemPromoCodeRequest struct {
	Code string `json:"code" binding:"required"`
}

// Redeem redeems a promo code
// POST /api/v1/orgs/:slug/billing/promo-codes/redeem
func (h *PromoCodeHandler) Redeem(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	var req RedeemPromoCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.Redeem(c.Request.Context(), &promocodeSvc.RedeemRequest{
		Code:           req.Code,
		OrganizationID: tenant.OrganizationID,
		UserID:         tenant.UserID,
		UserRole:       tenant.UserRole,
		IPAddress:      c.ClientIP(),
		UserAgent:      c.Request.UserAgent(),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": resp.MessageCode, "message_code": resp.MessageCode})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetRedemptionHistory gets redemption history
// GET /api/v1/orgs/:slug/billing/promo-codes/history
func (h *PromoCodeHandler) GetRedemptionHistory(c *gin.Context) {
	tenant := c.MustGet("tenant").(*middleware.TenantContext)

	history, err := h.service.GetRedemptionHistory(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"redemptions": history})
}

// ============ Admin API ============

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

// AdminCreate creates a promo code (admin)
// POST /api/v1/admin/promo-codes
func (h *PromoCodeHandler) AdminCreate(c *gin.Context) {
	userID := c.MustGet("user_id").(int64)

	var req CreatePromoCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse times
	var startsAt, expiresAt *time.Time
	if req.StartsAt != "" {
		t, err := time.Parse(time.RFC3339, req.StartsAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid starts_at format, use RFC3339"})
			return
		}
		startsAt = &t
	}
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expires_at format, use RFC3339"})
			return
		}
		expiresAt = &t
	}

	promoCode, err := h.service.Create(c.Request.Context(), &promocodeSvc.CreateRequest{
		Code:           req.Code,
		Name:           req.Name,
		Description:    req.Description,
		Type:           promocode.PromoCodeType(req.Type),
		PlanName:       req.PlanName,
		DurationMonths: req.DurationMonths,
		MaxUses:        req.MaxUses,
		MaxUsesPerOrg:  req.MaxUsesPerOrg,
		StartsAt:       startsAt,
		ExpiresAt:      expiresAt,
		CreatedByID:    userID,
	})
	if err != nil {
		if err == promocodeSvc.ErrPromoCodeAlreadyExists {
			c.JSON(http.StatusConflict, gin.H{"error": "promo code already exists"})
			return
		}
		if err == promocodeSvc.ErrInvalidPlan {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan name"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"promo_code": promoCode})
}

// AdminList lists promo codes (admin)
// GET /api/v1/admin/promo-codes
func (h *PromoCodeHandler) AdminList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := &promocode.ListFilter{
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

	codes, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"promo_codes": codes,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
	})
}

// AdminGet gets a promo code by ID (admin)
// GET /api/v1/admin/promo-codes/:id
func (h *PromoCodeHandler) AdminGet(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	promoCode, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "promo code not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"promo_code": promoCode})
}

// AdminDeactivate deactivates a promo code (admin)
// POST /api/v1/admin/promo-codes/:id/deactivate
func (h *PromoCodeHandler) AdminDeactivate(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.service.Deactivate(c.Request.Context(), id); err != nil {
		if err == promocodeSvc.ErrPromoCodeNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "promo code not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "promo code deactivated"})
}

// AdminActivate activates a promo code (admin)
// POST /api/v1/admin/promo-codes/:id/activate
func (h *PromoCodeHandler) AdminActivate(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.service.Activate(c.Request.Context(), id); err != nil {
		if err == promocodeSvc.ErrPromoCodeNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "promo code not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "promo code activated"})
}

// RegisterPromoCodeRoutes registers promo code routes for org-scoped billing
func RegisterPromoCodeRoutes(rg *gin.RouterGroup, service *promocodeSvc.Service) {
	handler := NewPromoCodeHandler(service)

	rg.POST("/validate", handler.Validate)
	rg.POST("/redeem", handler.Redeem)
	rg.GET("/history", handler.GetRedemptionHistory)
}

// RegisterAdminPromoCodeRoutes registers admin promo code routes
func RegisterAdminPromoCodeRoutes(rg *gin.RouterGroup, service *promocodeSvc.Service) {
	handler := NewPromoCodeHandler(service)

	rg.POST("", handler.AdminCreate)
	rg.GET("", handler.AdminList)
	rg.GET("/:id", handler.AdminGet)
	rg.POST("/:id/deactivate", handler.AdminDeactivate)
	rg.POST("/:id/activate", handler.AdminActivate)
}
