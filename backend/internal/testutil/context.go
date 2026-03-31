package testutil

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// NewGinContext creates a Gin test context with an HTTP recorder.
func NewGinContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	return c, w
}

// tenantKey matches middleware.TenantContext key.
type tenantCtxKey struct{}

// TenantContext mirrors middleware.TenantContext for test use.
type TenantContext struct {
	OrganizationID   int64
	OrganizationSlug string
	UserID           int64
	UserRole         string
}

// SetGinTenantContext sets tenant context on a Gin context (same keys as middleware).
func SetGinTenantContext(c *gin.Context, orgID, userID int64, role string) {
	tc := &TenantContext{
		OrganizationID:   orgID,
		OrganizationSlug: "test-org",
		UserID:           userID,
		UserRole:         role,
	}
	c.Set("tenant", tc)
	c.Set("user_id", userID)
}

// SetTenantContext returns a context.Context with tenant info embedded.
func SetTenantContext(ctx context.Context, orgID, userID int64, role string) context.Context {
	tc := &TenantContext{
		OrganizationID:   orgID,
		OrganizationSlug: "test-org",
		UserID:           userID,
		UserRole:         role,
	}
	return context.WithValue(ctx, tenantCtxKey{}, tc)
}
