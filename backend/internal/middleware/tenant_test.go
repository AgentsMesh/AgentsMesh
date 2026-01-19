package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestGetTenant(t *testing.T) {
	t.Run("should get tenant from gin context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		tc := &TenantContext{
			OrganizationID:   123,
			OrganizationSlug: "test-org",
			UserID:           456,
			UserRole:         "admin",
		}
		c.Set("tenant", tc)

		result := GetTenant(c)
		assert.NotNil(t, result)
		assert.Equal(t, int64(123), result.OrganizationID)
		assert.Equal(t, "test-org", result.OrganizationSlug)
		assert.Equal(t, int64(456), result.UserID)
		assert.Equal(t, "admin", result.UserRole)
	})

	t.Run("should get tenant from request context", func(t *testing.T) {
		tc := &TenantContext{
			OrganizationID: 789,
			UserID:         101,
		}
		ctx := SetTenant(context.Background(), tc)

		result := GetTenant(ctx)
		assert.NotNil(t, result)
		assert.Equal(t, int64(789), result.OrganizationID)
		assert.Equal(t, int64(101), result.UserID)
	})

	t.Run("should return nil when no tenant context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/", nil)

		result := GetTenant(c)
		assert.Nil(t, result)
	})

	t.Run("should return nil for plain context without tenant", func(t *testing.T) {
		ctx := context.Background()

		result := GetTenant(ctx)
		assert.Nil(t, result)
	})
}

func TestGetUserID(t *testing.T) {
	t.Run("should get user ID from gin context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", int64(123))

		userID := GetUserID(c)
		assert.Equal(t, int64(123), userID)
	})

	t.Run("should return 0 when user ID not set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		userID := GetUserID(c)
		assert.Equal(t, int64(0), userID)
	})

	t.Run("should return 0 when user ID is wrong type", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", "not-an-int")

		userID := GetUserID(c)
		assert.Equal(t, int64(0), userID)
	})
}

func TestSetTenant(t *testing.T) {
	t.Run("should set tenant in context", func(t *testing.T) {
		tc := &TenantContext{
			OrganizationID: 123,
			UserID:         456,
		}

		ctx := SetTenant(context.Background(), tc)

		result := GetTenant(ctx)
		assert.NotNil(t, result)
		assert.Equal(t, int64(123), result.OrganizationID)
	})

	t.Run("should overwrite existing tenant", func(t *testing.T) {
		tc1 := &TenantContext{OrganizationID: 111}
		tc2 := &TenantContext{OrganizationID: 222}

		ctx := SetTenant(context.Background(), tc1)
		ctx = SetTenant(ctx, tc2)

		result := GetTenant(ctx)
		assert.Equal(t, int64(222), result.OrganizationID)
	})
}

// Mock organization service for testing
type mockOrgService struct {
	org       *mockOrg
	isMember  bool
	role      string
	getErr    error
	memberErr error
	roleErr   error
}

type mockOrg struct {
	id   int64
	slug string
	name string
}

func (o *mockOrg) GetID() int64   { return o.id }
func (o *mockOrg) GetSlug() string { return o.slug }
func (o *mockOrg) GetName() string { return o.name }

func (m *mockOrgService) GetBySlug(ctx context.Context, slug string) (OrganizationGetter, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.org, nil
}

func (m *mockOrgService) IsMember(ctx context.Context, orgID, userID int64) (bool, error) {
	return m.isMember, m.memberErr
}

func (m *mockOrgService) GetMemberRole(ctx context.Context, orgID, userID int64) (string, error) {
	return m.role, m.roleErr
}

func TestTenantMiddleware(t *testing.T) {
	mockOrg := &mockOrg{id: 123, slug: "test-org", name: "Test Org"}

	t.Run("should set tenant context from path param", func(t *testing.T) {
		svc := &mockOrgService{org: mockOrg, isMember: true, role: "admin"}
		middleware := TenantMiddleware(svc)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations/test-org/pods", nil)
		c.Params = gin.Params{{Key: "slug", Value: "test-org"}}
		c.Set("user_id", int64(456))

		var capturedTenant *TenantContext
		middleware(c)

		// Manually get tenant since c.Next() isn't called in test
		capturedTenant = GetTenant(c)
		assert.NotNil(t, capturedTenant)
		assert.Equal(t, int64(123), capturedTenant.OrganizationID)
		assert.Equal(t, "test-org", capturedTenant.OrganizationSlug)
		assert.Equal(t, int64(456), capturedTenant.UserID)
		assert.Equal(t, "admin", capturedTenant.UserRole)
	})

	t.Run("should fail without org slug", func(t *testing.T) {
		svc := &mockOrgService{org: mockOrg, isMember: true, role: "admin"}
		middleware := TenantMiddleware(svc)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/pods", nil)
		c.Set("user_id", int64(456))

		middleware(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("should fail without user authentication", func(t *testing.T) {
		svc := &mockOrgService{org: mockOrg, isMember: true, role: "admin"}
		middleware := TenantMiddleware(svc)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations/test-org/pods", nil)
		c.Params = gin.Params{{Key: "slug", Value: "test-org"}}
		// No user_id set

		middleware(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("should fail when not a member", func(t *testing.T) {
		svc := &mockOrgService{org: mockOrg, isMember: false, role: ""}
		middleware := TenantMiddleware(svc)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations/test-org/pods", nil)
		c.Params = gin.Params{{Key: "slug", Value: "test-org"}}
		c.Set("user_id", int64(456))

		middleware(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

}

func TestRequireRole(t *testing.T) {
	t.Run("should allow matching role", func(t *testing.T) {
		middleware := RequireRole("admin", "owner")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("tenant", &TenantContext{UserRole: "admin"})

		called := false
		c.Request = httptest.NewRequest("GET", "/", nil)

		// Simulate Next() behavior
		middleware(c)

		// Check not aborted
		if !c.IsAborted() {
			called = true
		}

		assert.True(t, called)
	})

	t.Run("should deny non-matching role", func(t *testing.T) {
		middleware := RequireRole("admin", "owner")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("tenant", &TenantContext{UserRole: "member"})
		c.Request = httptest.NewRequest("GET", "/", nil)

		middleware(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.True(t, c.IsAborted())
	})

	t.Run("should deny without tenant context", func(t *testing.T) {
		middleware := RequireRole("admin")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/", nil)

		middleware(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.True(t, c.IsAborted())
	})
}

func TestRequireOwner(t *testing.T) {
	t.Run("should allow owner", func(t *testing.T) {
		middleware := RequireOwner()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("tenant", &TenantContext{UserRole: "owner"})
		c.Request = httptest.NewRequest("GET", "/", nil)

		middleware(c)

		assert.False(t, c.IsAborted())
	})

	t.Run("should deny admin", func(t *testing.T) {
		middleware := RequireOwner()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("tenant", &TenantContext{UserRole: "admin"})
		c.Request = httptest.NewRequest("GET", "/", nil)

		middleware(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestRequireAdmin(t *testing.T) {
	t.Run("should allow owner", func(t *testing.T) {
		middleware := RequireAdmin()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("tenant", &TenantContext{UserRole: "owner"})
		c.Request = httptest.NewRequest("GET", "/", nil)

		middleware(c)

		assert.False(t, c.IsAborted())
	})

	t.Run("should allow admin", func(t *testing.T) {
		middleware := RequireAdmin()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("tenant", &TenantContext{UserRole: "admin"})
		c.Request = httptest.NewRequest("GET", "/", nil)

		middleware(c)

		assert.False(t, c.IsAborted())
	})

	t.Run("should deny member", func(t *testing.T) {
		middleware := RequireAdmin()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("tenant", &TenantContext{UserRole: "member"})
		c.Request = httptest.NewRequest("GET", "/", nil)

		middleware(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}
