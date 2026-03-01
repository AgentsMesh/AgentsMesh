package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/infra/database"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockAdminDB implements database.DB interface for admin middleware testing
type mockAdminDB struct {
	user    *user.User
	findErr error
}

func (m *mockAdminDB) Transaction(fc func(tx database.DB) error) error {
	return fc(m)
}

func (m *mockAdminDB) WithContext(ctx context.Context) database.DB {
	return m
}

func (m *mockAdminDB) Create(value interface{}) error {
	return nil
}

func (m *mockAdminDB) First(dest interface{}, conds ...interface{}) error {
	if m.findErr != nil {
		return m.findErr
	}
	if m.user == nil {
		return gorm.ErrRecordNotFound
	}
	if u, ok := dest.(*user.User); ok {
		*u = *m.user
		return nil
	}
	return gorm.ErrRecordNotFound
}

func (m *mockAdminDB) Find(dest interface{}, conds ...interface{}) error {
	return nil
}

func (m *mockAdminDB) Save(value interface{}) error {
	return nil
}

func (m *mockAdminDB) Delete(value interface{}, conds ...interface{}) error {
	return nil
}

func (m *mockAdminDB) Updates(model interface{}, values interface{}) error {
	return nil
}

func (m *mockAdminDB) Model(value interface{}) database.DB {
	return m
}

func (m *mockAdminDB) Table(name string) database.DB {
	return m
}

func (m *mockAdminDB) Where(query interface{}, args ...interface{}) database.DB {
	return m
}

func (m *mockAdminDB) Select(query interface{}, args ...interface{}) database.DB {
	return m
}

func (m *mockAdminDB) Joins(query string, args ...interface{}) database.DB {
	return m
}

func (m *mockAdminDB) Preload(query string, args ...interface{}) database.DB {
	return m
}

func (m *mockAdminDB) Order(value interface{}) database.DB {
	return m
}

func (m *mockAdminDB) Limit(limit int) database.DB {
	return m
}

func (m *mockAdminDB) Offset(offset int) database.DB {
	return m
}

func (m *mockAdminDB) Group(name string) database.DB {
	return m
}

func (m *mockAdminDB) Count(count *int64) error {
	return nil
}

func (m *mockAdminDB) Scan(dest interface{}) error {
	return nil
}

func (m *mockAdminDB) GormDB() *gorm.DB {
	return nil
}

// Ensure mockAdminDB implements database.DB
var _ database.DB = (*mockAdminDB)(nil)

func TestAdminMiddleware(t *testing.T) {
	t.Run("should allow system admin user", func(t *testing.T) {
		db := &mockAdminDB{
			user: &user.User{
				ID:            1,
				Email:         "admin@example.com",
				IsSystemAdmin: true,
				IsActive:      true,
			},
		}
		middleware := AdminMiddleware(db)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/admin/dashboard", nil)
		c.Set("user_id", int64(1))

		middleware(c)

		assert.False(t, c.IsAborted())
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify admin user is set in context
		adminUser := GetAdminUser(c)
		assert.NotNil(t, adminUser)
		assert.Equal(t, int64(1), adminUser.ID)
		assert.True(t, adminUser.IsSystemAdmin)
	})

	t.Run("should reject user without authentication", func(t *testing.T) {
		db := &mockAdminDB{}
		middleware := AdminMiddleware(db)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/admin/dashboard", nil)
		// No user_id set

		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("should reject non-existent user", func(t *testing.T) {
		db := &mockAdminDB{
			findErr: gorm.ErrRecordNotFound,
		}
		middleware := AdminMiddleware(db)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/admin/dashboard", nil)
		c.Set("user_id", int64(999))

		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("should reject non-admin user", func(t *testing.T) {
		db := &mockAdminDB{
			user: &user.User{
				ID:            1,
				Email:         "user@example.com",
				IsSystemAdmin: false, // Not an admin
				IsActive:      true,
			},
		}
		middleware := AdminMiddleware(db)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/admin/dashboard", nil)
		c.Set("user_id", int64(1))

		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("should reject disabled admin user", func(t *testing.T) {
		db := &mockAdminDB{
			user: &user.User{
				ID:            1,
				Email:         "admin@example.com",
				IsSystemAdmin: true,
				IsActive:      false, // Disabled
			},
		}
		middleware := AdminMiddleware(db)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/admin/dashboard", nil)
		c.Set("user_id", int64(1))

		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestGetAdminUser(t *testing.T) {
	t.Run("should return admin user from context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		expectedUser := &user.User{
			ID:            1,
			Email:         "admin@example.com",
			IsSystemAdmin: true,
		}
		c.Set("admin_user", expectedUser)

		result := GetAdminUser(c)

		assert.NotNil(t, result)
		assert.Equal(t, int64(1), result.ID)
		assert.Equal(t, "admin@example.com", result.Email)
	})

	t.Run("should return nil when admin user not in context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		result := GetAdminUser(c)

		assert.Nil(t, result)
	})

	t.Run("should return nil when wrong type in context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("admin_user", "not a user")

		result := GetAdminUser(c)

		assert.Nil(t, result)
	})
}

func TestGetAdminUserID(t *testing.T) {
	t.Run("should return admin user ID from context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("admin_user_id", int64(123))

		result := GetAdminUserID(c)

		assert.Equal(t, int64(123), result)
	})

	t.Run("should return 0 when admin user ID not in context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		result := GetAdminUserID(c)

		assert.Equal(t, int64(0), result)
	})

	t.Run("should return 0 when wrong type in context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("admin_user_id", "not an int")

		result := GetAdminUserID(c)

		assert.Equal(t, int64(0), result)
	})
}
