package v1

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func setupBillingTestDB(t *testing.T) *gorm.DB {
	db := testkit.SetupTestDB(t)
	seedBillingTestData(db)
	return db
}

func seedBillingTestData(db *gorm.DB) {
	db.Exec(`INSERT INTO subscription_plans (id, name, display_name, price_per_seat_monthly, max_users, max_runners, max_repositories, max_concurrent_pods, features, is_active)
		VALUES (1, 'based', 'Based', 0, 1, 1, 3, 1, '{}', 1)`)
	db.Exec(`INSERT INTO subscription_plans (id, name, display_name, price_per_seat_monthly, max_users, max_runners, max_repositories, max_concurrent_pods, features, is_active)
		VALUES (2, 'pro', 'Pro', 20, 10, 10, 20, 5, '{}', 1)`)
	db.Exec(`INSERT INTO subscription_plans (id, name, display_name, price_per_seat_monthly, max_users, max_runners, max_repositories, max_concurrent_pods, features, is_active)
		VALUES (3, 'enterprise', 'Enterprise', 40, 50, 100, -1, 20, '{}', 1)`)

	db.Exec(`INSERT INTO plan_prices (plan_id, currency, price_monthly, price_yearly)
		VALUES (1, 'usd', 0, 0)`)
	db.Exec(`INSERT INTO plan_prices (plan_id, currency, price_monthly, price_yearly)
		VALUES (2, 'usd', 20, 200)`)
	db.Exec(`INSERT INTO plan_prices (plan_id, currency, price_monthly, price_yearly)
		VALUES (3, 'usd', 40, 400)`)
}

func setupBillingHandler(t *testing.T) (*BillingHandler, *gorm.DB, *gin.Engine) {
	db := setupBillingTestDB(t)
	billingSvc := billing.NewService(infra.NewBillingRepository(db), "")
	handler := NewBillingHandler(billingSvc)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	return handler, db, router
}

func setBillingTenantContext(c *gin.Context, orgID int64, userID int64, role string) {
	tc := &middleware.TenantContext{
		OrganizationID:   orgID,
		OrganizationSlug: "test-org",
		UserID:           userID,
		UserRole:         role,
	}
	c.Set("tenant", tc)
}
