package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/promocode"
	"github.com/gin-gonic/gin"
)

// ===========================================
// Admin Management Tests
// ===========================================

func TestPromoCodeHandler_AdminCreate(t *testing.T) {
	handler, _, router := setupPromoCodeHandlerTest(t)

	router.POST("/admin/promo-codes", func(c *gin.Context) {
		c.Set("user_id", int64(1))
		handler.AdminCreate(c)
	})

	tests := []struct {
		name       string
		body       CreatePromoCodeRequest
		wantStatus int
	}{
		{
			name: "create valid promo code",
			body: CreatePromoCodeRequest{
				Code:           "NEWCODE2024",
				Name:           "New Code",
				Type:           "media",
				PlanName:       "pro",
				DurationMonths: 3,
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "create with invalid plan",
			body: CreatePromoCodeRequest{
				Code:           "BADPLAN",
				Name:           "Bad Plan",
				Type:           "media",
				PlanName:       "nonexistent",
				DurationMonths: 1,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing required fields",
			body: CreatePromoCodeRequest{
				Code: "INCOMPLETE",
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/admin/promo-codes", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestPromoCodeHandler_AdminList(t *testing.T) {
	handler, db, router := setupPromoCodeHandlerTest(t)

	// Create test promo codes
	now := time.Now()
	for i := 1; i <= 5; i++ {
		db.Create(&promocode.PromoCode{
			Code:           "LIST" + string(rune('A'+i-1)),
			Name:           "List Test " + string(rune('A'+i-1)),
			Type:           promocode.PromoTypeMedia,
			PlanName:       "pro",
			DurationMonths: i,
			IsActive:       true,
			StartsAt:       now,
		})
	}

	router.GET("/admin/promo-codes", func(c *gin.Context) {
		c.Set("user_id", int64(1))
		handler.AdminList(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/promo-codes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	codes := resp["promo_codes"].([]interface{})
	if len(codes) != 5 {
		t.Errorf("expected 5 codes, got %d", len(codes))
	}

	if resp["total"].(float64) != 5 {
		t.Errorf("expected total=5, got %v", resp["total"])
	}
}

func TestPromoCodeHandler_AdminDeactivate(t *testing.T) {
	handler, db, router := setupPromoCodeHandlerTest(t)

	// Create test promo code
	code := &promocode.PromoCode{
		Code:           "DEACTIVATE",
		Name:           "Deactivate Test",
		Type:           promocode.PromoTypeMedia,
		PlanName:       "pro",
		DurationMonths: 1,
		IsActive:       true,
		StartsAt:       time.Now(),
	}
	db.Create(code)

	router.POST("/admin/promo-codes/:id/deactivate", func(c *gin.Context) {
		c.Set("user_id", int64(1))
		handler.AdminDeactivate(c)
	})

	tests := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{
			name:       "deactivate existing code",
			id:         "1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "deactivate nonexistent code",
			id:         "99999",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid id",
			id:         "abc",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/admin/promo-codes/"+tt.id+"/deactivate", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}
