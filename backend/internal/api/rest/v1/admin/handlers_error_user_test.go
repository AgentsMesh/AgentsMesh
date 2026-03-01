package admin

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	adminservice "github.com/AgentsMesh/AgentsMesh/backend/internal/service/admin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestUserHandler_ListUsers_WithFilters(t *testing.T) {
	t.Run("should filter by is_active", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[1] = &user.User{ID: 1, Email: "active@example.com", IsActive: true}
		db.totalCount = 1

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/users?is_active=true", nil)

		handler.ListUsers(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should filter by is_admin", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[1] = &user.User{ID: 1, Email: "admin@example.com", IsSystemAdmin: true}
		db.totalCount = 1

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/users?is_admin=true", nil)

		handler.ListUsers(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestUserHandler_DisableUser_NotFound(t *testing.T) {
	t.Run("should return 404 when user not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/999/disable", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.DisableUser(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUserHandler_EnableUser_NotFound(t *testing.T) {
	t.Run("should return 404 when user not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/999/enable", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.EnableUser(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUserHandler_GrantAdmin_NotFound(t *testing.T) {
	t.Run("should return 404 when user not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/999/grant-admin", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.GrantAdmin(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUserHandler_RevokeAdmin_NotFound(t *testing.T) {
	t.Run("should return 404 when user not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/999/revoke-admin", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.RevokeAdmin(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUserHandler_UpdateUser_NotFound(t *testing.T) {
	t.Run("should return 404 when user not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		body := bytes.NewBufferString(`{"name": "New Name"}`)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("PUT", "/users/999", body)
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.UpdateUser(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUserHandler_UpdateUser_InvalidID(t *testing.T) {
	t.Run("should return 400 for invalid user ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		body := bytes.NewBufferString(`{"name": "New Name"}`)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("PUT", "/users/invalid", body)
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.UpdateUser(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("should return 400 for invalid JSON body", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[1] = &user.User{ID: 1, Email: "test@example.com"}

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		body := bytes.NewBufferString(`{invalid json}`)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("PUT", "/users/1", body)
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.UpdateUser(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUserHandler_DisableUser_InvalidID(t *testing.T) {
	t.Run("should return 400 for invalid user ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/invalid/disable", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.DisableUser(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUserHandler_EnableUser_InvalidID(t *testing.T) {
	t.Run("should return 400 for invalid user ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/invalid/enable", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.EnableUser(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUserHandler_GrantAdmin_InvalidID(t *testing.T) {
	t.Run("should return 400 for invalid user ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/invalid/grant-admin", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.GrantAdmin(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUserHandler_RevokeAdmin_InvalidID(t *testing.T) {
	t.Run("should return 400 for invalid user ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/invalid/revoke-admin", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.RevokeAdmin(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
