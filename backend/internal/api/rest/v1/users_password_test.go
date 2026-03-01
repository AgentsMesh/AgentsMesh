package v1

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	userService "github.com/AgentsMesh/AgentsMesh/backend/internal/service/user"
	"github.com/gin-gonic/gin"
)

// ===========================================
// Password/Authentication Tests
// ===========================================

func TestChangePassword(t *testing.T) {
	handler, mockUserSvc, _, router := setupUserHandlerTest()

	testUser := &user.User{
		ID:       1,
		Email:    "test@example.com",
		Username: "testuser",
		IsActive: true,
	}
	mockUserSvc.AddUser(testUser)

	router.POST("/users/me/password", func(c *gin.Context) {
		setUserContext(c, 1)
		handler.ChangePassword(c)
	})

	body := `{"current_password": "oldpass", "new_password": "newpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/users/me/password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify authentication was attempted
	if len(mockUserSvc.AuthAttempts) != 1 {
		t.Errorf("expected 1 auth attempt, got %d", len(mockUserSvc.AuthAttempts))
	}
}

func TestChangePasswordInvalidJSON(t *testing.T) {
	handler, _, _, router := setupUserHandlerTest()

	router.POST("/users/me/password", func(c *gin.Context) {
		setUserContext(c, 1)
		handler.ChangePassword(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/users/me/password", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestChangePasswordUserNotFound(t *testing.T) {
	handler, _, _, router := setupUserHandlerTest()

	router.POST("/users/me/password", func(c *gin.Context) {
		setUserContext(c, 999)
		handler.ChangePassword(c)
	})

	body := `{"current_password": "oldpass", "new_password": "newpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/users/me/password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestChangePasswordWrongCurrentPassword(t *testing.T) {
	handler, mockUserSvc, _, router := setupUserHandlerTest()

	testUser := &user.User{
		ID:       1,
		Email:    "test@example.com",
		Username: "testuser",
		IsActive: true,
	}
	mockUserSvc.AddUser(testUser)
	mockUserSvc.AuthenticateErr = userService.ErrInvalidCredentials

	router.POST("/users/me/password", func(c *gin.Context) {
		setUserContext(c, 1)
		handler.ChangePassword(c)
	})

	body := `{"current_password": "wrongpass", "new_password": "newpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/users/me/password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}
