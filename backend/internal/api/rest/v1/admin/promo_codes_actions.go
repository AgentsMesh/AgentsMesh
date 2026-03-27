package admin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/service/admin"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

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
