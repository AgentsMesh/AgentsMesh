package v1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/gin-gonic/gin"
)

func TestNewBillingHandler(t *testing.T) {
	db := setupBillingTestDB(t)
	billingSvc := billing.NewService(infra.NewBillingRepository(db), "")

	handler := NewBillingHandler(billingSvc)

	if handler == nil {
		t.Error("expected non-nil handler")
	}
	if handler.billingService != billingSvc {
		t.Error("expected billing service to be set")
	}
}

func TestBillingHandler_ListPlans(t *testing.T) {
	handler, _, router := setupBillingHandler(t)

	router.GET("/plans", func(c *gin.Context) {
		setBillingTenantContext(c, 1, 1, "owner")
		handler.ListPlans(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/plans", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	plans, ok := resp["plans"].([]interface{})
	if !ok {
		t.Fatal("expected plans array in response")
	}
	if len(plans) != 3 {
		t.Errorf("expected 3 plans, got %d", len(plans))
	}
}

func TestBillingHandler_ListPlansWithPrices(t *testing.T) {
	handler, _, router := setupBillingHandler(t)

	router.GET("/plans/prices", func(c *gin.Context) {
		setBillingTenantContext(c, 1, 1, "owner")
		handler.ListPlansWithPrices(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/plans/prices", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if _, ok := resp["plans"]; !ok {
		t.Error("expected plans in response")
	}
}

func TestBillingHandler_GetDeploymentInfo(t *testing.T) {
	handler, _, router := setupBillingHandler(t)

	router.GET("/deployment", func(c *gin.Context) {
		handler.GetDeploymentInfo(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/deployment", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if _, ok := resp["deployment_type"]; !ok {
		t.Error("expected deployment_type in response")
	}
}

func TestBillingHandler_GetPublicPricing(t *testing.T) {
	handler, _, router := setupBillingHandler(t)

	router.GET("/pricing", func(c *gin.Context) {
		handler.GetPublicPricing(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/pricing", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if _, ok := resp["plans"]; !ok {
		t.Error("expected plans in response")
	}
}

func TestBillingHandler_GetOverview_NoSubscription(t *testing.T) {
	handler, _, router := setupBillingHandler(t)

	router.GET("/overview", func(c *gin.Context) {
		setBillingTenantContext(c, 1, 1, "owner")
		handler.GetOverview(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/overview", nil)
	router.ServeHTTP(w, req)

	// When no subscription exists, returns 500 with error
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBillingHandler_GetOverview_WithSubscription(t *testing.T) {
	handler, db, router := setupBillingHandler(t)

	// Create a subscription for org 1
	db.Exec(`INSERT INTO subscriptions (organization_id, plan_id, status, billing_cycle, current_period_start, current_period_end, seat_count)
		VALUES (1, 2, 'active', 'monthly', datetime('now'), datetime('now', '+30 days'), 1)`)

	router.GET("/overview", func(c *gin.Context) {
		setBillingTenantContext(c, 1, 1, "owner")
		handler.GetOverview(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/overview", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBillingHandler_GetPlanPrices(t *testing.T) {
	handler, _, router := setupBillingHandler(t)

	router.GET("/plans/:plan_name/prices", func(c *gin.Context) {
		setBillingTenantContext(c, 1, 1, "owner")
		handler.GetPlanPrices(c)
	})

	// Test invalid plan
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/plans/nonexistent/prices", nil)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusNotFound {
		t.Errorf("expected status 404 for invalid plan, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestBillingHandler_GetAllPlanPrices(t *testing.T) {
	handler, _, router := setupBillingHandler(t)

	router.GET("/plans/:name/all-prices", func(c *gin.Context) {
		setBillingTenantContext(c, 1, 1, "owner")
		handler.GetAllPlanPrices(c)
	})

	// Test with valid plan name
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/plans/pro/all-prices", nil)
	router.ServeHTTP(w, req)

	// Returns 200 with prices array
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Test with invalid plan name
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/plans/nonexistent/all-prices", nil)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusNotFound {
		t.Errorf("expected status 404 for invalid plan, got %d: %s", w2.Code, w2.Body.String())
	}
}
