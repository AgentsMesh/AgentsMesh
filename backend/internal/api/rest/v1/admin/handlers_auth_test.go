package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/auth"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAuthService implements authServiceInterface for testing
type mockAuthService struct {
	loginResult *auth.LoginResult
	loginErr    error
}

func (m *mockAuthService) Login(ctx context.Context, email, password string) (*auth.LoginResult, error) {
	if m.loginErr != nil {
		return nil, m.loginErr
	}
	return m.loginResult, nil
}

func TestAuthHandler_GetMe(t *testing.T) {
	t.Run("should return admin user info", func(t *testing.T) {
		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/me", nil)

		handler := NewAuthHandler(nil, nil)
		handler.GetMe(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should return 401 when admin user not in context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/me", nil)
		// No admin_user set

		handler := NewAuthHandler(nil, nil)
		handler.GetMe(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestAuthHandler_Login(t *testing.T) {
	t.Run("should return 400 for invalid request body", func(t *testing.T) {
		handler := NewAuthHandler(nil, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(`{}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "Invalid request")
		assert.NotEmpty(t, response["code"], "expected 'code' field in error response")
	})

	t.Run("should return 400 for missing email", func(t *testing.T) {
		handler := NewAuthHandler(nil, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(`{"password":"test123"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("should return 400 for invalid email format", func(t *testing.T) {
		handler := NewAuthHandler(nil, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(`{"email":"invalid-email","password":"test123"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("should return 400 for missing password", func(t *testing.T) {
		handler := NewAuthHandler(nil, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(`{"email":"test@example.com"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("should return 400 for malformed JSON", func(t *testing.T) {
		handler := NewAuthHandler(nil, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(`{invalid json}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("should return 401 when auth service returns error", func(t *testing.T) {
		mockSvc := &mockAuthService{
			loginErr: errors.New("invalid credentials"),
		}
		handler := NewAuthHandlerWithInterface(mockSvc, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(`{"email":"test@example.com","password":"wrongpass"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "Invalid email or password")
		assert.NotEmpty(t, response["code"], "expected 'code' field in error response")
	})

	t.Run("should return 403 when user is not system admin", func(t *testing.T) {
		mockSvc := &mockAuthService{
			loginResult: &auth.LoginResult{
				User: &user.User{
					ID:            1,
					Email:         "user@example.com",
					IsSystemAdmin: false,
					IsActive:      true,
				},
				Token:        "test-token",
				RefreshToken: "test-refresh",
			},
		}
		handler := NewAuthHandlerWithInterface(mockSvc, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(`{"email":"user@example.com","password":"password123"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "system administrator privileges")
		assert.NotEmpty(t, response["code"], "expected 'code' field in error response")
	})

	t.Run("should return 403 when admin user is disabled", func(t *testing.T) {
		mockSvc := &mockAuthService{
			loginResult: &auth.LoginResult{
				User: &user.User{
					ID:            1,
					Email:         "admin@example.com",
					IsSystemAdmin: true,
					IsActive:      false, // Disabled user
				},
				Token:        "test-token",
				RefreshToken: "test-refresh",
			},
		}
		handler := NewAuthHandlerWithInterface(mockSvc, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(`{"email":"admin@example.com","password":"password123"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "disabled")
		assert.NotEmpty(t, response["code"], "expected 'code' field in error response")
	})

	t.Run("should return 200 and tokens for successful admin login", func(t *testing.T) {
		mockSvc := &mockAuthService{
			loginResult: &auth.LoginResult{
				User: &user.User{
					ID:            1,
					Email:         "admin@example.com",
					Username:      "admin",
					IsSystemAdmin: true,
					IsActive:      true,
				},
				Token:        "test-access-token",
				RefreshToken: "test-refresh-token",
				ExpiresIn:    3600,
			},
		}
		handler := NewAuthHandlerWithInterface(mockSvc, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(`{"email":"admin@example.com","password":"password123"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "test-access-token", response["token"])
		assert.Equal(t, "test-refresh-token", response["refresh_token"])
		assert.NotNil(t, response["user"])

		userResp := response["user"].(map[string]interface{})
		assert.Equal(t, "admin@example.com", userResp["email"])
		assert.Equal(t, true, userResp["is_system_admin"])
	})
}
