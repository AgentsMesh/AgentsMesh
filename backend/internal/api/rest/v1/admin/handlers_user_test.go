package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	adminservice "github.com/AgentsMesh/AgentsMesh/backend/internal/service/admin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserHandler_ListUsers(t *testing.T) {
	t.Run("should list users successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[1] = &user.User{ID: 1, Email: "user1@example.com", IsActive: true}
		db.users[2] = &user.User{ID: 2, Email: "user2@example.com", IsActive: true}
		db.totalCount = 2

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/users?page=1&page_size=20", nil)

		handler.ListUsers(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(2), response["total"])
	})
}

func TestUserHandler_GetUser(t *testing.T) {
	t.Run("should get user successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		testName := "Test User"
		db.users[1] = &user.User{ID: 1, Email: "test@example.com", Name: &testName}

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/users/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.GetUser(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "test@example.com", response["email"])
	})

	t.Run("should return 404 when user not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/users/999", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.GetUser(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("should return 400 for invalid user ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/users/invalid", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.GetUser(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUserHandler_DisableUser(t *testing.T) {
	t.Run("should disable user successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[2] = &user.User{ID: 2, Email: "user@example.com", IsActive: true}

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/2/disable", nil)
		c.Params = gin.Params{{Key: "id", Value: "2"}}

		handler.DisableUser(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should prevent disabling self", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[1] = &user.User{ID: 1, Email: "admin@example.com", IsActive: true}

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w) // admin_user_id is 1
		c.Request = httptest.NewRequest("POST", "/users/1/disable", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.DisableUser(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUserHandler_RevokeAdmin(t *testing.T) {
	t.Run("should revoke admin successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[2] = &user.User{ID: 2, Email: "other@example.com", IsSystemAdmin: true}

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/2/revoke-admin", nil)
		c.Params = gin.Params{{Key: "id", Value: "2"}}

		handler.RevokeAdmin(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should prevent revoking own admin", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[1] = &user.User{ID: 1, Email: "admin@example.com", IsSystemAdmin: true}

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/1/revoke-admin", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.RevokeAdmin(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUserHandler_UpdateUser(t *testing.T) {
	t.Run("should update user successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		oldName := "Old Name"
		db.users[1] = &user.User{ID: 1, Email: "old@example.com", Name: &oldName}

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		body := bytes.NewBufferString(`{"name": "New Name"}`)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("PUT", "/users/1", body)
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.UpdateUser(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should return 400 for empty updates", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[1] = &user.User{ID: 1, Email: "test@example.com"}

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		body := bytes.NewBufferString(`{}`)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("PUT", "/users/1", body)
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.UpdateUser(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUserHandler_EnableUser(t *testing.T) {
	t.Run("should enable user successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[2] = &user.User{ID: 2, Email: "user@example.com", IsActive: false}

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/2/enable", nil)
		c.Params = gin.Params{{Key: "id", Value: "2"}}

		handler.EnableUser(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestUserHandler_GrantAdmin(t *testing.T) {
	t.Run("should grant admin successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[2] = &user.User{ID: 2, Email: "user@example.com", IsSystemAdmin: false}

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/2/grant-admin", nil)
		c.Params = gin.Params{{Key: "id", Value: "2"}}

		handler.GrantAdmin(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
