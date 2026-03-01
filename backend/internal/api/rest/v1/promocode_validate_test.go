package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/promocode"
	promocodeSvc "github.com/AgentsMesh/AgentsMesh/backend/internal/service/promocode"
	"github.com/gin-gonic/gin"
)

// ===========================================
// Validation and Redemption Tests
// ===========================================

func TestPromoCodeHandler_Validate(t *testing.T) {
	handler, db, router := setupPromoCodeHandlerTest(t)

	// Create test promo code
	now := time.Now()
	past := now.AddDate(0, -1, 0)
	future := now.AddDate(0, 1, 0)

	db.Create(&promocode.PromoCode{
		Code:           "TESTCODE2024",
		Name:           "Test Promo",
		Type:           promocode.PromoTypeMedia,
		PlanName:       "pro",
		DurationMonths: 3,
		IsActive:       true,
		StartsAt:       past,
		ExpiresAt:      &future,
	})

	router.POST("/validate", func(c *gin.Context) {
		setPromoCodeTenantContext(c, 1, 1, "owner")
		handler.Validate(c)
	})

	tests := []struct {
		name       string
		body       map[string]string
		wantStatus int
		wantValid  bool
	}{
		{
			name:       "valid code",
			body:       map[string]string{"code": "TESTCODE2024"},
			wantStatus: http.StatusOK,
			wantValid:  true,
		},
		{
			name:       "nonexistent code",
			body:       map[string]string{"code": "NOTEXIST"},
			wantStatus: http.StatusOK,
			wantValid:  false,
		},
		{
			name:       "missing code",
			body:       map[string]string{},
			wantStatus: http.StatusBadRequest,
			wantValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
				return
			}

			if tt.wantStatus == http.StatusOK {
				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to parse response: %v", err)
				}
				if resp["valid"].(bool) != tt.wantValid {
					t.Errorf("expected valid=%v, got %v", tt.wantValid, resp["valid"])
				}
			}
		})
	}
}

func TestPromoCodeHandler_Redeem(t *testing.T) {
	handler, db, _ := setupPromoCodeHandlerTest(t)

	// Create test promo code
	now := time.Now()
	past := now.AddDate(0, -1, 0)
	future := now.AddDate(0, 1, 0)

	db.Create(&promocode.PromoCode{
		Code:           "REDEEMTEST",
		Name:           "Redeem Test",
		Type:           promocode.PromoTypeMedia,
		PlanName:       "pro",
		DurationMonths: 3,
		IsActive:       true,
		StartsAt:       past,
		ExpiresAt:      &future,
		MaxUsesPerOrg:  1,
	})

	tests := []struct {
		name            string
		userRole        string
		code            string
		wantStatus      int
		wantMessageCode string
	}{
		{
			name:            "non-owner cannot redeem",
			userRole:        "member",
			code:            "REDEEMTEST",
			wantStatus:      http.StatusBadRequest,
			wantMessageCode: promocodeSvc.ErrCodeNotOwner,
		},
		{
			name:       "owner can redeem",
			userRole:   "owner",
			code:       "REDEEMTEST",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localRouter := gin.New()
			localRouter.POST("/redeem", func(c *gin.Context) {
				setPromoCodeTenantContext(c, 1, 1, tt.userRole)
				handler.Redeem(c)
			})

			bodyBytes, _ := json.Marshal(map[string]string{"code": tt.code})
			req := httptest.NewRequest(http.MethodPost, "/redeem", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			localRouter.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}

			// Verify error response contains message_code
			if tt.wantStatus == http.StatusBadRequest && tt.wantMessageCode != "" {
				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to parse response: %v", err)
				}
				if resp["message_code"] != tt.wantMessageCode {
					t.Errorf("expected message_code=%s, got %v", tt.wantMessageCode, resp["message_code"])
				}
			}
		})
	}
}
