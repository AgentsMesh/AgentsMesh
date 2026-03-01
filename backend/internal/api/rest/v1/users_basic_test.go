package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	orgService "github.com/AgentsMesh/AgentsMesh/backend/internal/service/organization"
	userService "github.com/AgentsMesh/AgentsMesh/backend/internal/service/user"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupUserHandlerTest() (*UserHandler, *userService.MockService, *orgService.MockService, *gin.Engine) {
	mockUserSvc := userService.NewMockService()
	mockOrgSvc := orgService.NewMockService()
	handler := NewUserHandler(mockUserSvc, mockOrgSvc)

	router := gin.New()
	return handler, mockUserSvc, mockOrgSvc, router
}

func setUserContext(c *gin.Context, userID int64) {
	c.Set("user_id", userID)
}

func TestNewUserHandler(t *testing.T) {
	handler, _, _, _ := setupUserHandlerTest()
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestGetCurrentUser(t *testing.T) {
	handler, mockUserSvc, _, router := setupUserHandlerTest()

	// Add test user
	testUser := &user.User{
		ID:       1,
		Email:    "test@example.com",
		Username: "testuser",
		IsActive: true,
	}
	mockUserSvc.AddUser(testUser)

	router.GET("/users/me", func(c *gin.Context) {
		setUserContext(c, 1)
		handler.GetCurrentUser(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	userData := resp["user"].(map[string]interface{})
	if userData["email"] != "test@example.com" {
		t.Errorf("expected email test@example.com, got %v", userData["email"])
	}
}

func TestGetCurrentUserNotFound(t *testing.T) {
	handler, _, _, router := setupUserHandlerTest()

	router.GET("/users/me", func(c *gin.Context) {
		setUserContext(c, 999) // Non-existent user
		handler.GetCurrentUser(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestUpdateCurrentUser(t *testing.T) {
	handler, mockUserSvc, _, router := setupUserHandlerTest()

	testUser := &user.User{
		ID:       1,
		Email:    "test@example.com",
		Username: "testuser",
		IsActive: true,
	}
	mockUserSvc.AddUser(testUser)

	router.PUT("/users/me", func(c *gin.Context) {
		setUserContext(c, 1)
		handler.UpdateCurrentUser(c)
	})

	body := `{"name": "New Name", "avatar_url": "https://example.com/avatar.png"}`
	req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify update was called
	if len(mockUserSvc.UpdatedUsers) != 1 {
		t.Errorf("expected 1 update call, got %d", len(mockUserSvc.UpdatedUsers))
	}
}

func TestUpdateCurrentUserInvalidJSON(t *testing.T) {
	handler, mockUserSvc, _, router := setupUserHandlerTest()

	testUser := &user.User{ID: 1, Email: "test@example.com", Username: "testuser"}
	mockUserSvc.AddUser(testUser)

	router.PUT("/users/me", func(c *gin.Context) {
		setUserContext(c, 1)
		handler.UpdateCurrentUser(c)
	})

	req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}
