package v1

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	promocodeSvc "github.com/anthropics/agentsmesh/backend/internal/service/promocode"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func setupPromoCodeHandlerTest(t *testing.T) (*PromoCodeHandler, *gorm.DB, *gin.Engine) {
	db := testkit.SetupTestDB(t)

	// Seed test data
	seedPromoCodeTestData(t, db)

	promoRepo := infra.NewPromocodeRepository(db)
	service := promocodeSvc.NewService(promoRepo, infra.NewGormBillingProvider(db))
	handler := NewPromoCodeHandler(service)
	router := gin.New()

	return handler, db, router
}

func seedPromoCodeTestData(t *testing.T, db *gorm.DB) {
	t.Helper()
	db.Exec(`INSERT INTO users (id, email, username, name) VALUES (1, 'test@example.com', 'testuser', 'Test User')`)
	db.Exec(`INSERT INTO organizations (id, name, slug) VALUES (1, 'Test Org', 'test-org')`)
	db.Exec(`INSERT INTO subscription_plans (id, name, display_name, max_users, max_runners, max_repositories, features) VALUES (1, 'based', 'Based', 1, 1, 3, X'7B7D')`)
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
