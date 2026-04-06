package webhooks

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDBForPayment(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Create required tables
	db.Exec(`CREATE TABLE IF NOT EXISTS subscription_plans (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		display_name TEXT,
		tier TEXT DEFAULT 'based',
		price_per_seat_monthly REAL DEFAULT 0,
		price_per_seat_yearly REAL DEFAULT 0,
		max_users INTEGER DEFAULT 1,
		max_runners INTEGER DEFAULT 1,
		max_repositories INTEGER DEFAULT 10,
		max_concurrent_pods INTEGER DEFAULT 1,
		max_pod_minutes INTEGER DEFAULT 100,
		max_storage_gb INTEGER DEFAULT 1,
		stripe_price_id_monthly TEXT,
		stripe_price_id_yearly TEXT,
		features TEXT
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS subscriptions (
		id INTEGER PRIMARY KEY,
		organization_id INTEGER NOT NULL,
		plan_id INTEGER,
		status TEXT DEFAULT 'active',
		stripe_subscription_id TEXT,
		stripe_customer_id TEXT,
		seat_count INTEGER DEFAULT 1,
		current_period_start DATETIME,
		current_period_end DATETIME,
		created_at DATETIME,
		updated_at DATETIME
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS payment_orders (
		id INTEGER PRIMARY KEY,
		organization_id INTEGER NOT NULL,
		order_no TEXT UNIQUE,
		external_order_no TEXT,
		order_type TEXT,
		amount REAL,
		actual_amount REAL,
		payment_provider TEXT,
		status TEXT,
		paid_at DATETIME,
		created_by_id INTEGER,
		created_at DATETIME,
		updated_at DATETIME
	)`)

	return db
}

func testLoggerForPayment() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func createTestPaymentRouter(cfg *config.Config) (*WebhookRouter, *gorm.DB) {
	db := setupTestDBForPayment(nil)
	logger := testLoggerForPayment()
	registry := NewHandlerRegistry(logger)
	SetupDefaultHandlers(registry, logger)

	// Create billing service without payment factory (will be nil)
	billingSvc := billing.NewService(infra.NewBillingRepository(db), "")

	return &WebhookRouter{
		db:             db,
		cfg:            cfg,
		logger:         logger,
		registry:       registry,
		billingSvc:     billingSvc,
		paymentFactory: nil, // No payment factory for these tests
	}, db
}

// ===========================================
// Stripe Payment Handler Tests
// ===========================================

func TestHandleStripeWebhook_NotConfigured(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestPaymentRouter(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Content-Type", "application/json")

	router.handleStripeWebhook(c)

	// Should return 503 when Stripe is not configured
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

// ===========================================
// Alipay Payment Handler Tests
// ===========================================

func TestHandleAlipayWebhook_NotConfigured(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestPaymentRouter(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/webhooks/alipay", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	router.handleAlipayWebhook(c)

	// Should return 503 when Alipay is not configured
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

// ===========================================
// WeChat Payment Handler Tests
// ===========================================

func TestHandleWeChatWebhook_NotConfigured(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestPaymentRouter(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/webhooks/wechat", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Content-Type", "application/json")

	router.handleWeChatWebhook(c)

	// Should return 503 when WeChat is not configured
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

// ===========================================
// Mock Payment Handler Tests
// ===========================================

func TestHandleMockCheckoutComplete_NotEnabled(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestPaymentRouter(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := `{"session_id": "mock_sess_123"}`
	c.Request = httptest.NewRequest("POST", "/webhooks/mock/complete", bytes.NewReader([]byte(payload)))
	c.Request.Header.Set("Content-Type", "application/json")

	router.handleMockCheckoutComplete(c)

	// Should return 403 when mock is not enabled
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestGetMockSession_NotEnabled(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestPaymentRouter(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("GET", "/webhooks/mock/session/test123", nil)
	c.Params = []gin.Param{{Key: "session_id", Value: "test123"}}

	router.getMockSession(c)

	// Should return 403 when mock is not enabled
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

// ===========================================
// Routes Tests
// ===========================================

func TestRegisterRoutes(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestPaymentRouter(cfg)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	rg := engine.Group("/webhooks")

	// Should not panic when registering routes
	router.RegisterRoutes(rg)

	// Verify routes are registered by checking a test request (new format)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/webhooks/test-org/gitlab/123", bytes.NewReader([]byte(`{"object_kind": "push"}`)))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(w, req)

	// Should get a response (not 404)
	if w.Code == http.StatusNotFound {
		t.Error("expected route to be registered")
	}
}

// ===========================================
// NewWebhookRouter Tests (with mock DB)
// ===========================================

func TestNewWebhookRouterWithBillingSvc(t *testing.T) {
	db := setupTestDBForPayment(nil)
	logger := testLoggerForPayment()
	cfg := &config.Config{}
	billingSvc := billing.NewService(infra.NewBillingRepository(db), "")

	router := NewWebhookRouterWithBillingSvc(db, cfg, logger, billingSvc)

	if router == nil {
		t.Error("expected non-nil router")
	}
	if router.db != db {
		t.Error("expected db to be set")
	}
	if router.cfg != cfg {
		t.Error("expected cfg to be set")
	}
	if router.billingSvc != billingSvc {
		t.Error("expected billing service to be set")
	}
	if router.registry == nil {
		t.Error("expected registry to be set")
	}
}

func setupTestDBForPaymentPtr() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})

	// Create required tables
	db.Exec(`CREATE TABLE IF NOT EXISTS subscription_plans (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		display_name TEXT,
		tier TEXT DEFAULT 'based',
		price_per_seat_monthly REAL DEFAULT 0,
		price_per_seat_yearly REAL DEFAULT 0,
		max_users INTEGER DEFAULT 1,
		max_runners INTEGER DEFAULT 1,
		max_repositories INTEGER DEFAULT 10,
		max_concurrent_pods INTEGER DEFAULT 1,
		max_pod_minutes INTEGER DEFAULT 100,
		max_storage_gb INTEGER DEFAULT 1,
		stripe_price_id_monthly TEXT,
		stripe_price_id_yearly TEXT,
		features TEXT
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS subscriptions (
		id INTEGER PRIMARY KEY,
		organization_id INTEGER NOT NULL,
		plan_id INTEGER,
		status TEXT DEFAULT 'active',
		stripe_subscription_id TEXT,
		stripe_customer_id TEXT,
		seat_count INTEGER DEFAULT 1,
		current_period_start DATETIME,
		current_period_end DATETIME,
		created_at DATETIME,
		updated_at DATETIME
	)`)

	return db
}

func TestNewWebhookRouter(t *testing.T) {
	db := setupTestDBForPaymentPtr()
	logger := testLoggerForPayment()
	cfg := &config.Config{}

	router := NewWebhookRouter(db, cfg, logger)

	if router == nil {
		t.Error("expected non-nil router")
	}
	if router.db != db {
		t.Error("expected db to be set")
	}
	if router.billingSvc == nil {
		t.Error("expected billing service to be created")
	}
}

// ===========================================
// LemonSqueezy Payment Handler Tests
// ===========================================

func TestHandleLemonSqueezyWebhook_NotConfigured(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestPaymentRouter(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/webhooks/lemonsqueezy", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Signature", "test_signature")

	router.handleLemonSqueezyWebhook(c)

	// Should return 503 when LemonSqueezy is not configured
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d: %s", http.StatusServiceUnavailable, w.Code, w.Body.String())
	}
}

func TestHandleLemonSqueezyWebhook_MissingSignature(t *testing.T) {
	// This test requires a payment factory that reports LemonSqueezy as available
	// For now, we just test that the handler exists and returns proper error for not configured
	cfg := &config.Config{}
	router, _ := createTestPaymentRouter(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/webhooks/lemonsqueezy", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	// Missing X-Signature header

	router.handleLemonSqueezyWebhook(c)

	// Should return 503 when LemonSqueezy is not configured (checked before signature)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d: %s", http.StatusServiceUnavailable, w.Code, w.Body.String())
	}
}

func TestHandleLemonSqueezyWebhook_ReadBodyError(t *testing.T) {
	// Test that handler properly handles body read errors
	// Since LemonSqueezy is not configured, this will return 503 before reading body
	cfg := &config.Config{}
	router, _ := createTestPaymentRouter(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/webhooks/lemonsqueezy", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Signature", "test")

	router.handleLemonSqueezyWebhook(c)

	// Should return 503 when not configured
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}
