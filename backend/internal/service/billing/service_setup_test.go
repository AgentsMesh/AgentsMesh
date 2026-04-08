package billing

import (
	"net/http/httptest"
	"testing"

	billingdomain "github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing.
// Delegates to testkit.SetupTestDB for shared schema.
func setupTestDB(t *testing.T) *gorm.DB {
	return testkit.SetupTestDB(t)
}

// newTestRepo creates a BillingRepository from *gorm.DB for tests.
func newTestRepo(db *gorm.DB) billingdomain.BillingRepository {
	return infra.NewBillingRepository(db)
}

// setupTestService creates a test service with seeded plans
func setupTestService(t *testing.T) (*Service, *gorm.DB) {
	db := setupTestDB(t)
	svc := NewService(newTestRepo(db), "")

	// Seed standard plans: based (ID=1), pro (ID=2), enterprise (ID=3)
	seedTestPlan(t, db)
	seedProPlan(t, db)
	seedEnterprisePlan(t, db)

	return svc, db
}

func createTestGinContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/", nil)
	return c, w
}
