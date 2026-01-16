package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/promocode"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	promocodeSvc "github.com/anthropics/agentsmesh/backend/internal/service/promocode"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupPromoCodeHandlerTest(t *testing.T) (*PromoCodeHandler, *gorm.DB, *gin.Engine) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Create tables
	setupPromoCodeTestDB(t, db)

	service := promocodeSvc.NewService(db)
	handler := NewPromoCodeHandler(service)
	router := gin.New()

	return handler, db, router
}

func setupPromoCodeTestDB(t *testing.T, db *gorm.DB) {
	// Create users table
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			username TEXT NOT NULL UNIQUE,
			name TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error; err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}

	// Create organizations table
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS organizations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			subscription_plan TEXT NOT NULL DEFAULT 'free',
			subscription_status TEXT NOT NULL DEFAULT 'active',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error; err != nil {
		t.Fatalf("failed to create organizations table: %v", err)
	}

	// Create subscription_plans table
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS subscription_plans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL,
			price_per_seat_monthly REAL NOT NULL DEFAULT 0,
			included_pod_minutes INTEGER NOT NULL DEFAULT 0,
			price_per_extra_minute REAL NOT NULL DEFAULT 0,
			max_users INTEGER NOT NULL,
			max_runners INTEGER NOT NULL,
			max_concurrent_pods INTEGER NOT NULL DEFAULT 0,
			max_repositories INTEGER NOT NULL,
			features BLOB,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error; err != nil {
		t.Fatalf("failed to create subscription_plans table: %v", err)
	}

	// Create subscriptions table
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS subscriptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL UNIQUE,
			plan_id INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			billing_cycle TEXT NOT NULL DEFAULT 'monthly',
			current_period_start DATETIME NOT NULL,
			current_period_end DATETIME NOT NULL,
			stripe_customer_id TEXT,
			stripe_subscription_id TEXT,
			custom_quotas TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error; err != nil {
		t.Fatalf("failed to create subscriptions table: %v", err)
	}

	// Create promo_codes table
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS promo_codes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			description TEXT,
			type TEXT NOT NULL,
			plan_name TEXT NOT NULL,
			duration_months INTEGER NOT NULL,
			max_uses INTEGER,
			used_count INTEGER NOT NULL DEFAULT 0,
			max_uses_per_org INTEGER NOT NULL DEFAULT 1,
			starts_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_by_id INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error; err != nil {
		t.Fatalf("failed to create promo_codes table: %v", err)
	}

	// Create promo_code_redemptions table
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS promo_code_redemptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			promo_code_id INTEGER NOT NULL,
			organization_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			plan_name TEXT NOT NULL,
			duration_months INTEGER NOT NULL,
			previous_plan_name TEXT,
			previous_period_end DATETIME,
			new_period_end DATETIME NOT NULL,
			ip_address TEXT,
			user_agent TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error; err != nil {
		t.Fatalf("failed to create promo_code_redemptions table: %v", err)
	}

	// Insert test data
	db.Exec(`INSERT INTO users (id, email, username, name) VALUES (1, 'test@example.com', 'testuser', 'Test User')`)
	db.Exec(`INSERT INTO organizations (id, name, slug) VALUES (1, 'Test Org', 'test-org')`)
	db.Exec(`INSERT INTO subscription_plans (id, name, display_name, max_users, max_runners, max_repositories, features) VALUES (1, 'free', 'Free', 1, 1, 3, X'7B7D')`)
	db.Exec(`INSERT INTO subscription_plans (id, name, display_name, max_users, max_runners, max_repositories, price_per_seat_monthly, features) VALUES (2, 'pro', 'Pro', 5, 10, 10, 20, X'7B7D')`)
	db.Exec(`INSERT INTO subscription_plans (id, name, display_name, max_users, max_runners, max_repositories, price_per_seat_monthly, features) VALUES (3, 'enterprise', 'Enterprise', 50, 100, -1, 40, X'7B7D')`)
}

func setPromoCodeTenantContext(c *gin.Context, orgID int64, userID int64, role string) {
	tc := &middleware.TenantContext{
		OrganizationID:   orgID,
		OrganizationSlug: "test-org",
		UserID:           userID,
		UserRole:         role,
	}
	c.Set("tenant", tc)
}

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
